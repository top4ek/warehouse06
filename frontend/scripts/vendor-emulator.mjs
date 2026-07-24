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
  git(["init"], dir);
  git(["remote", "add", "origin", repo], dir);
  git(["fetch", "--depth", "1", "origin", ref], dir);
  git(["checkout", "FETCH_HEAD"], dir);
  return { dir, sha: git(["rev-parse", "HEAD"], dir) };
}

function copyFile(from, to) {
  fs.mkdirSync(path.dirname(to), { recursive: true });
  fs.copyFileSync(from, to);
}

function copyGlob(srcRoot, pattern, destRoot) {
  const [dirPart, filePart] = pattern.includes("/")
    ? [path.dirname(pattern), path.basename(pattern)]
    : [".", pattern];
  const absDir = path.join(srcRoot, dirPart);
  if (!fs.existsSync(absDir)) return;
  for (const name of fs.readdirSync(absDir)) {
    if (filePart === "*" || name === filePart) {
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
    <script src="./zip.js/WebContent/zip.js"></script>
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
    path.join(srcRoot, "WebContent"),
    path.join(destRoot, "zip.js", "WebContent"),
    (name) => name !== "tests",
  );
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
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const lock = readLock();

  if (args.ref) {
    lock.vector06js.ref = args.ref;
  }

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "warehouse06-vendor-"));
  try {
    const results = {};

    const v06 = shallowClone(lock.vector06js.repo, lock.vector06js.ref, tmp);
    results.vector06js = v06.sha;
    if (!args.check) vendorVector06js(v06.dir, destDir);

    const i8080 = shallowClone(lock["i8080-js"].repo, lock["i8080-js"].ref, tmp);
    results["i8080-js"] = i8080.sha;
    if (!args.check) vendorI8080(i8080.dir, destDir);

    const zip = shallowClone(lock["zip.js"].repo, lock["zip.js"].ref, tmp);
    results["zip.js"] = zip.sha;
    if (!args.check) vendorZipJs(zip.dir, destDir);

    const bin2wav = shallowClone(lock["bin2wav"].repo, lock["bin2wav"].ref, tmp);
    results["bin2wav"] = bin2wav.sha;
    if (!args.check) vendorBin2wav(bin2wav.dir, BIN2WAV_DEST);

    lock.vector06js.ref = results.vector06js;
    lock["i8080-js"].ref = results["i8080-js"];
    lock["zip.js"].ref = results["zip.js"];
    lock["bin2wav"].ref = results["bin2wav"];

    if (args.check) {
      const current = readLock();
      const drift = Object.keys(results).filter((k) => {
        const key = k === "vector06js" ? "vector06js" : k;
        return current[key].ref !== results[k];
      });
      if (drift.length) {
        console.error("vendor.lock.json is stale for:", drift.join(", "));
        process.exit(1);
      }
      console.log("vendor.lock.json matches upstream refs.");
      return;
    }

    writeLock(lock);
    console.log("Vendored emulator sources → emulator-src/");
    console.log("Updated vendor.lock.json:", results);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

main();
