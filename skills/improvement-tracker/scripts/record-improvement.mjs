#!/usr/bin/env node
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { basename, join, resolve } from "node:path";

const VALID_STATUSES = new Set(["not-impl", "implemented", "agent-recorded", "rejected", "archived"]);

function usage() {
  console.error(`Usage:
  node record-improvement.mjs --title "Short title" [--status not-impl] [--root .] [--body "Details"]

Statuses:
  not-impl | implemented | agent-recorded | rejected | archived`);
}

function parseArgs(argv) {
  const args = { status: "not-impl", root: ".", body: "" };
  for (let i = 0; i < argv.length; i += 1) {
    const key = argv[i];
    const value = argv[i + 1];
    if (!key.startsWith("--") || value === undefined) {
      usage();
      process.exit(2);
    }
    i += 1;
    if (key === "--title") args.title = value;
    else if (key === "--status") args.status = value;
    else if (key === "--root") args.root = value;
    else if (key === "--body") args.body = value;
    else {
      console.error(`Unknown option: ${key}`);
      usage();
      process.exit(2);
    }
  }
  return args;
}

function slugify(value) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80) || "improvement";
}

function today() {
  return new Date().toISOString().slice(0, 10);
}

function ensureReadme(targetDir) {
  const readmePath = join(targetDir, "README.md");
  if (existsSync(readmePath)) return;

  const skillRoot = resolve(new URL("..", import.meta.url).pathname);
  const templatePath = join(skillRoot, "assets", "docs", "improvements", "README.md");
  if (existsSync(templatePath)) {
    writeFileSync(readmePath, readFileSync(templatePath, "utf8"));
    return;
  }

  writeFileSync(readmePath, "# Improvements\n\nRecord agent-discovered improvement items here.\n");
}

const args = parseArgs(process.argv.slice(2));
if (!args.title) {
  console.error("Missing required --title.");
  usage();
  process.exit(2);
}
if (!VALID_STATUSES.has(args.status)) {
  console.error(`Invalid --status: ${args.status}`);
  usage();
  process.exit(2);
}

const root = resolve(args.root);
const baseDir = join(root, "docs", "improvements");
const statusDir = join(baseDir, args.status);
for (const status of VALID_STATUSES) mkdirSync(join(baseDir, status), { recursive: true });
ensureReadme(baseDir);

const date = today();
const slug = slugify(args.title);
let filePath = join(statusDir, `${date}-${slug}.md`);
let suffix = 2;
while (existsSync(filePath)) {
  filePath = join(statusDir, `${date}-${slug}-${suffix}.md`);
  suffix += 1;
}

const content = `# ${args.title}\n\n## Status\n\n${args.status}\n\n## Context\n\n${args.body || "TODO: describe what was observed."}\n\n## Problem\n\nTODO: explain why this matters.\n\n## Suggested Direction\n\nTODO: describe what should be improved.\n\n## Target Owner\n\n- skill:\n- agent:\n- docs:\n- code area:\n\n## Evidence\n\n- file:\n- command:\n- behavior:\n\n## Decision Log\n\n- ${date}: Created.\n`;

writeFileSync(filePath, content);
console.log(filePath);
