#!/usr/bin/env node
/**
 * Bundle vector06js sources from emulator-src/ into public/emulator/.
 * zip.js stays external (Web Workers); everything else is concatenated.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.join(__dirname, "..");
const srcDir = path.join(frontendDir, "emulator-src");
const outDir = path.join(frontendDir, "public", "emulator");

const PRE_ZIP = [
  "i8080-js/i8080.js",
  "i8080-js/i8080_disasm.js",
  "src/memory.js",
  "src/io.js",
  "src/tv.js",
  "src/hooks.js",
  "src/keyboard.js",
  "src/i8253.js",
  "src/ay.js",
  "src/rom.js",
  "src/debugger.js",
  "src/fili.js",
  "src/sound.js",
  "src/fd1793.js",
  "src/fddimage.js",
  "src/wavplayer.js",
];

const POST_ZIP = ["wav.js/wav.js", "src/main.js"];

const STATIC_PNG = ["cassette-32x32.png", "diskette-32x32.png", "omg-cat.png"];
const DATA_DIRS = ["boot", "fdd", "basic", "testroms"];

function concatBundle(files) {
  return files
    .map((rel) => {
      const abs = path.join(srcDir, rel);
      if (!fs.existsSync(abs)) {
        throw new Error(`Missing emulator source: ${rel}`);
      }
      const body = fs.readFileSync(abs, "utf8");
      return `;/* === ${rel} === */\n${body}`;
    })
    .join("\n;\n");
}

function copyTree(from, to) {
  fs.mkdirSync(to, { recursive: true });
  for (const entry of fs.readdirSync(from, { withFileTypes: true })) {
    const srcPath = path.join(from, entry.name);
    const dstPath = path.join(to, entry.name);
    if (entry.isDirectory()) {
      copyTree(srcPath, dstPath);
    } else {
      fs.copyFileSync(srcPath, dstPath);
    }
  }
}

function main() {
  if (!fs.existsSync(srcDir)) {
    console.error("emulator-src/ not found; run from frontend/ after vendoring sources.");
    process.exit(1);
  }

  fs.rmSync(outDir, { recursive: true, force: true });
  fs.mkdirSync(path.join(outDir, "dist"), { recursive: true });

  fs.writeFileSync(
    path.join(outDir, "dist", "emulator.pre-zip.bundle.js"),
    concatBundle(PRE_ZIP),
  );
  fs.writeFileSync(
    path.join(outDir, "dist", "emulator.post-zip.bundle.js"),
    concatBundle(POST_ZIP),
  );

  copyTree(path.join(srcDir, "zip.js"), path.join(outDir, "zip.js"));

  for (const png of STATIC_PNG) {
    fs.copyFileSync(path.join(srcDir, png), path.join(outDir, png));
  }

  for (const dir of DATA_DIRS) {
    const from = path.join(srcDir, dir);
    if (fs.existsSync(from)) {
      copyTree(from, path.join(outDir, dir));
    }
  }

  fs.copyFileSync(path.join(srcDir, "index.html"), path.join(outDir, "index.html"));

  console.log(`Built emulator → ${path.relative(frontendDir, outDir)}/`);
}

main();
