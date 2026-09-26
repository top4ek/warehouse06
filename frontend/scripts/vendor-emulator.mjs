#!/usr/bin/env node
/**
 * Refresh emulator-src/ from upstream git repos pinned in vendor.lock.json.
 *
 * Usage:
 *   node scripts/vendor-emulator.mjs
 *   node scripts/vendor-emulator.mjs --ref <commit>
 *   node scripts/vendor-emulator.mjs --check
 */
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.join(__dirname, "..");
const destDir = path.join(frontendDir, "emulator-src");
const lockPath = path.join(destDir, "vendor.lock.json");

function parseArgs(argv) {
  const args = { check: false, ref: null };
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === "--check") args.check = true;
    if (argv[i] === "--ref" && argv[i + 1]) {
      args.ref = argv[++i];
    }
  }
  return args;
}

function readLock() {
  return JSON.parse(fs.readFileSync(lockPath, "utf8"));
}

function writeLock(lock) {
  fs.writeFileSync(lockPath, `${JSON.stringify(lock, null, 2)}\n`);
}

function git(args, cwd) {
  return execFileSync("git", args, { cwd, encoding: "utf8" }).trim();
}

function shallowClone(repo, ref, tmpRoot) {
  const dir = path.join(tmpRoot, repo.replace(/[^\w]+/g, "_"));
  fs.mkdirSync(dir, { recursive: true });
  git(["init", "--quiet"], dir);
  git(["remote", "add", "origin", repo], dir);
  git(["fetch", "--quiet", "--depth", "1", "origin", ref], dir);
  git(["-c", "advice.detachedHead=false", "checkout", "--quiet", "FETCH_HEAD"], dir);
  return { dir, sha: git(["rev-parse", "HEAD"], dir) };
}

function copyFile(from, to) {
  fs.mkdirSync(path.dirname(to), { recursive: true });
  fs.copyFileSync(from, to);
}

export function copyGlob(srcRoot, pattern, destRoot) {
  const [dirPart, filePart] = pattern.includes("/")
    ? [path.dirname(pattern), path.basename(pattern)]
    : [".", pattern];
  const absDir = path.join(srcRoot, dirPart);
  if (!fs.existsSync(absDir)) return;
  const matcher = new RegExp(`^${filePart.split("*").map((part) =>
    part.replace(/[.+?^${}()|[\]\\]/g, "\\$&")).join(".*")}$`);
  for (const name of fs.readdirSync(absDir)) {
    if (matcher.test(name)) {
      const src = path.join(absDir, name);
      if (fs.statSync(src).isFile()) {
        copyFile(src, path.join(destRoot, dirPart, name));
      }
    }
  }
}

function copyTree(from, to, filter) {
  if (!fs.existsSync(from)) return;
  fs.mkdirSync(to, { recursive: true });
  for (const entry of fs.readdirSync(from, { withFileTypes: true })) {
    if (filter && !filter(entry.name)) continue;
    const src = path.join(from, entry.name);
    const dst = path.join(to, entry.name);
    if (entry.isDirectory()) {
      copyTree(src, dst, filter);
    } else if (entry.isSymbolicLink()) {
      fs.symlinkSync(fs.readlinkSync(src), dst);
    } else {
      fs.copyFileSync(src, dst);
    }
  }
}

const VECTOR06JS_DATA_DIRS = ["boot", "fdd", "basic", "testroms"];

function vendorVector06js(srcRoot, destRoot) {
  copyGlob(srcRoot, "src/*.js", destRoot);
  copyGlob(srcRoot, "*.png", destRoot);
  for (const dir of VECTOR06JS_DATA_DIRS) {
    copyTree(path.join(srcRoot, dir), path.join(destRoot, dir));
  }
  copyTree(path.join(srcRoot, "wav.js"), path.join(destRoot, "wav.js"));
  copyFile(path.join(srcRoot, "index.html"), path.join(destRoot, "index.html.raw"));

  // Adapt index.html: replace per-file scripts with bundle loaders (same as initial import).
  const raw = fs.readFileSync(path.join(destRoot, "index.html.raw"), "utf8");
  const bodyIdx = raw.indexOf("<body");
  const head = raw.slice(0, bodyIdx);
  const body = raw.slice(bodyIdx);
  const cleaned = body.replace(/\s*<script[^>]*>[\s\S]*?<\/script>\s*/g, "\n");
  const scripts = `    <script src="./dist/emulator.pre-zip.bundle.js"></script>
    <script src="./zip.js/zip.js"></script>
    <script src="./dist/emulator.post-zip.bundle.js"></script>
`;
  const bodyClose = cleaned.lastIndexOf("</body>");
  if (bodyClose === -1) throw new Error("index.html: missing </body>");
  const adapted = head + cleaned.slice(0, bodyClose) + scripts + cleaned.slice(bodyClose);
  fs.writeFileSync(path.join(destRoot, "index.html"), adapted);
  fs.unlinkSync(path.join(destRoot, "index.html.raw"));
}

function vendorI8080(srcRoot, destRoot) {
  const dest = path.join(destRoot, "i8080-js");
  fs.mkdirSync(dest, { recursive: true });
  for (const file of ["i8080.js", "i8080_disasm.js"]) {
    copyFile(path.join(srcRoot, file), path.join(dest, file));
  }
}

function vendorZipJs(srcRoot, destRoot) {
  copyTree(
    path.join(srcRoot, "dist"),
    path.join(destRoot, "zip.js"),
    (name) => ["zip.js", "zip-web-worker.js", "zip-module.wasm"].includes(name),
  );
  copyFile(path.join(srcRoot, "LICENSE"), path.join(destRoot, "zip.js", "LICENSE"));
}

// bin2wav's tape.js/makewav.js are pure-JS but CommonJS; adapt them to ESM so
// the React bundle can import them. Destination is under src/ (unlike the other
// vendored trees) because these are part of the Vite/TS module graph.
const BIN2WAV_DEST = path.join(frontendDir, "src", "vendor", "bin2wav");

function replaceOnce(text, needle, replacement, file) {
  if (!text.includes(needle)) {
    throw new Error(`bin2wav ${file}: expected snippet not found (upstream drift?):\n${needle}`);
  }
  return text.replace(needle, replacement);
}

function vendorBin2wav(srcRoot, destRoot) {
  const makewavHeader = `/*
 * Vendored from svofski/bin2wav (makewav.js). See emulator-src/vendor.lock.json
 * for the pinned commit; refresh via \`npm run vendor:emulator\`.
 * Adapted from CommonJS to ESM (module.exports -> export). Do not edit by hand.
 */
`;
  const tapeHeader = `/*
 * Vendored from svofski/bin2wav (tape.js). See emulator-src/vendor.lock.json
 * for the pinned commit; refresh via \`npm run vendor:emulator\`.
 * Adapted from CommonJS to ESM (require/module.exports -> import/export). Do not edit by hand.
 */
`;

  let makewav = fs.readFileSync(path.join(srcRoot, "makewav.js"), "utf8");
  makewav = replaceOnce(
    makewav,
    "module.exports = {\n    Wav: function(opt_params) {\n        return new Wav(opt_params);\n    }\n};",
    "function WavFactory(opt_params) {\n    return new Wav(opt_params);\n}\n\nexport { WavFactory as Wav };",
    "makewav.js",
  );
  fs.mkdirSync(destRoot, { recursive: true });
  fs.writeFileSync(path.join(destRoot, "makewav.js"), makewavHeader + makewav);

  let tape = fs.readFileSync(path.join(srcRoot, "tape.js"), "utf8");
  tape = replaceOnce(
    tape,
    "var wavmodule = require('./makewav');",
    'import * as wavmodule from "./makewav.js";',
    "tape.js",
  );
  // ESM is always strict mode; upstream's prototype.makewav writes an implicit
  // global `wav` (sloppy-mode only). Declare it so the v06c-rom path works.
  tape = replaceOnce(
    tape,
    "\n    wav = wavmodule.Wav(params);",
    "\n    var wav = wavmodule.Wav(params);",
    "tape.js",
  );
  tape = replaceOnce(
    tape,
    "module.exports = {\n    TapeFormat: function(fmt, forfile, konst, leader, sampleRate) {\n        return new TapeFormat(fmt, forfile, konst, leader, sampleRate);\n    }\n};",
    "function TapeFormatFactory(fmt, forfile, konst, leader, sampleRate) {\n    return new TapeFormat(fmt, forfile, konst, leader, sampleRate);\n}\n\nexport { TapeFormatFactory as TapeFormat };",
    "tape.js",
  );
  fs.writeFileSync(path.join(destRoot, "tape.js"), tapeHeader + tape);
  copyFile(path.join(__dirname, "bin2wav-tape.d.ts"), path.join(destRoot, "tape.d.ts"));
}

// Only these paths are generated. README and vendor.lock.json are maintained separately.
const MANAGED_PATHS = ["src", "i8080-js", "zip.js", "wav.js", "index.html",
  "cassette-32x32.png", "diskette-32x32.png", "omg-cat.png", ...VECTOR06JS_DATA_DIRS];

export function treeDifferences(expected, actual) {
  const differences = [];
  function visit(relative) {
    const left = path.join(expected, relative);
    const right = path.join(actual, relative);
    if (!fs.existsSync(left) || !fs.existsSync(right)) {
      differences.push(relative);
      return;
    }
    const leftType = fs.lstatSync(left);
    const rightType = fs.lstatSync(right);
    if (leftType.isSymbolicLink() || rightType.isSymbolicLink()) {
      if (!leftType.isSymbolicLink() || !rightType.isSymbolicLink() ||
          fs.readlinkSync(left) !== fs.readlinkSync(right)) differences.push(relative);
    } else if (leftType.isDirectory() && rightType.isDirectory()) {
      for (const name of new Set([...fs.readdirSync(left), ...fs.readdirSync(right)])) {
        visit(path.join(relative, name));
      }
    } else if (leftType.isFile() && rightType.isFile()) {
      if (!fs.readFileSync(left).equals(fs.readFileSync(right))) differences.push(relative);
    } else differences.push(relative);
  }
  visit("");
  return differences.sort();
}

export function replaceTree(from, to) {
  fs.rmSync(to, { recursive: true, force: true });
  if (fs.statSync(from).isDirectory()) copyTree(from, to);
  else copyFile(from, to);
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const lock = readLock();
  if (args.ref) lock.vector06js.ref = args.ref;

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "warehouse06-vendor-"));
  try {
    const staged = path.join(tmp, "emulator-src");
    const stagedBin2wav = path.join(tmp, "bin2wav");
    const results = {};
    for (const name of Object.keys(lock)) {
      const source = shallowClone(lock[name].repo, lock[name].ref, tmp);
      results[name] = source.sha;
      if (name === "vector06js") vendorVector06js(source.dir, staged);
      else if (name === "i8080-js") vendorI8080(source.dir, staged);
      else if (name === "zip.js") vendorZipJs(source.dir, staged);
      else if (name === "bin2wav") vendorBin2wav(source.dir, stagedBin2wav);
      else throw new Error(`Unknown vendor: ${name}`);
    }
    // Fail on upstream drift before replacing any files in the working tree.
    execFileSync("git", ["apply", "--no-index", path.join(__dirname, "emulator.patch")], { cwd: staged });

    if (args.check) {
      const differences = Object.keys(results).filter((name) => lock[name].ref !== results[name]);
      for (const name of MANAGED_PATHS) {
        differences.push(...treeDifferences(path.join(staged, name), path.join(destDir, name))
          .map((relative) => path.join(name, relative)));
      }
      differences.push(...treeDifferences(stagedBin2wav, BIN2WAV_DEST).map((name) => `bin2wav/${name}`));
      if (differences.length) throw new Error(`Vendored files differ from pinned sources and adaptations:\n${differences.join("\n")}`);
      console.log("Vendored files match pinned sources and adaptations.");
      return;
    }
    for (const name of MANAGED_PATHS) replaceTree(path.join(staged, name), path.join(destDir, name));
    replaceTree(stagedBin2wav, BIN2WAV_DEST);
    for (const name of Object.keys(results)) lock[name].ref = results[name];
    writeLock(lock);
    console.log("Updated vendored emulator:", results);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
