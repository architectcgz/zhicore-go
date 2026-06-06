#!/usr/bin/env node
import { execSync } from "node:child_process";
import { existsSync } from "node:fs";

const DEFAULT_GLOBS = [
  "--glob '!node_modules'",
  "--glob '!dist'",
  "--glob '!build'",
  "--glob '!coverage'",
  "--glob '!storybook-static'",
  "--glob '!*.lock'",
].join(" ");

function run(cmd) {
  try {
    return execSync(cmd, {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
      maxBuffer: 1024 * 1024 * 8,
    }).trim();
  } catch {
    return "";
  }
}

function rg(pattern, extra = "") {
  const output = run(`rg -n ${DEFAULT_GLOBS} ${extra} ${JSON.stringify(pattern)} .`);
  if (!output) return "";
  const lines = output.split("\n");
  const shown = lines.slice(0, 120).join("\n");
  return lines.length > 120 ? `${shown}\n... truncated ${lines.length - 120} lines` : shown;
}

function section(title, body) {
  console.log(`## ${title}\n`);
  console.log(body || "No candidates found.");
  console.log("");
}

console.log("# Frontend Boundary Inspection Report\n");
console.log(`Working directory: ${process.cwd()}\n`);

section("Repository", run("git rev-parse --show-toplevel") || "Not a git repository.");
section("Git Status", run("git status --short") || "Clean or unavailable.");

const markers = [
  "package.json",
  "vite.config.ts",
  "vite.config.js",
  "nuxt.config.ts",
  "next.config.js",
  "tsconfig.json",
  "src",
].filter(existsSync);
section("Frontend Project Markers", markers.join("\n"));

section(
  "Raw Async Event Bindings",
  rg("@(click|submit|keyup|change|input)[^=]*=\\\"[^\\\"]*(async|save|submit|delete|remove|export|download|fetch|load|retry)", "--glob '*.{vue,tsx,jsx}'")
);

section(
  "Unhandled Promise Or Floating Async Candidates",
  rg("\\b(then|catch|finally)\\s*\\(|void\\s+[A-Za-z0-9_]+\\(|async\\s+function|const\\s+[A-Za-z0-9_]+\\s*=\\s*async", "--glob '*.{vue,ts,tsx,js,jsx}'")
);

section(
  "Lifecycle Cleanup Candidates",
  rg("\\b(setInterval|setTimeout|addEventListener|watch\\(|subscribe|new\\s+[A-Z][A-Za-z0-9_]*|onMounted|onUnmounted)\\b", "--glob '*.{vue,ts,tsx,js,jsx}'")
);

section(
  "Component Contract Drift Candidates",
  rg("defineProps|defineEmits|defineModel|v-model|watch\\(|toRefs|reactive\\(|ref\\(|emit\\(", "--glob '*.{vue,ts,tsx,js,jsx}'")
);

section(
  "Hardcoded Styling Candidates",
  rg("(margin|padding|gap|font-size|color|background|border-radius|box-shadow|z-index)\\s*:\\s*[^;]*(px|#[0-9a-fA-F]{3,8}|rgba?\\(|!important)", "--glob '*.{vue,css,scss,ts,tsx,js,jsx}'")
);

section(
  "Rendered Internal Copy Candidates",
  rg("TODO|FIXME|implementation|设计意图|结构说明|占位|待办|debug", "--glob '*.{vue,tsx,jsx,html}'")
);

console.log("## Reading Guidance\n");
console.log("Treat every match as a candidate only. Confirm by reading the owning component, composable, route view, caller, styles, and tests before recommending changes.");
