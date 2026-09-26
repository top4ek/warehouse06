import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { test } from "node:test";
import { copyGlob, replaceTree, treeDifferences } from "./vendor-emulator.mjs";

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "vendor-test-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const source = path.join(root, "source");
  const destination = path.join(root, "destination");
  fs.mkdirSync(source);
  fs.mkdirSync(destination);
  return { source, destination };
}

test("vendor copies JavaScript wildcards and exact filenames only", (t) => {
  const { source, destination } = fixture(t);
  fs.mkdirSync(path.join(source, "src"));
  for (const name of ["main.js", "rom.js", "notes.txt", "mainXjs"]) {
    fs.writeFileSync(path.join(source, "src", name), name);
  }
  fs.mkdirSync(path.join(source, "src", "directory.js"));
  copyGlob(source, "src/*.js", destination);
  assert.deepEqual(fs.readdirSync(path.join(destination, "src")), ["main.js", "rom.js"]);
  fs.writeFileSync(path.join(source, "logo.png"), "image");
  fs.writeFileSync(path.join(source, "logoXpng"), "other");
  copyGlob(source, "logo.png", destination);
  assert.equal(fs.readFileSync(path.join(destination, "logo.png"), "utf8"), "image");
  assert.equal(fs.existsSync(path.join(destination, "logoXpng")), false);
});

test("vendor check detects edits, missing files and stale files", (t) => {
  const { source, destination } = fixture(t);
  fs.writeFileSync(path.join(source, "edited.js"), "expected");
  fs.writeFileSync(path.join(destination, "edited.js"), "changed");
  fs.writeFileSync(path.join(source, "missing.js"), "missing");
  fs.writeFileSync(path.join(destination, "stale.js"), "stale");
  assert.deepEqual(treeDifferences(source, destination), ["edited.js", "missing.js", "stale.js"]);
});

test("vendor replacement removes stale files and is repeatable", (t) => {
  const { source, destination } = fixture(t);
  fs.writeFileSync(path.join(source, "main.js"), "expected");
  fs.writeFileSync(path.join(destination, "stale.js"), "stale");
  replaceTree(source, destination);
  assert.deepEqual(treeDifferences(source, destination), []);
  replaceTree(source, destination);
  assert.deepEqual(treeDifferences(source, destination), []);
});

test("vendor preserves upstream symlinks and detects changes to them", (t) => {
  const { source, destination } = fixture(t);
  fs.writeFileSync(path.join(source, "disk.fdd"), "disk image");
  fs.symlinkSync("disk.fdd", path.join(source, "ryba.fdd"));
  replaceTree(source, destination);
  assert.equal(fs.readlinkSync(path.join(destination, "ryba.fdd")), "disk.fdd");
  assert.deepEqual(treeDifferences(source, destination), []);
  fs.unlinkSync(path.join(destination, "ryba.fdd"));
  fs.writeFileSync(path.join(destination, "ryba.fdd"), "disk image");
  assert.deepEqual(treeDifferences(source, destination), ["ryba.fdd"]);
});
