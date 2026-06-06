#!/usr/bin/env node
import { execSync } from "node:child_process";
import { existsSync } from "node:fs";

const DEFAULT_GLOBS = [
  "--glob '!node_modules'",
  "--glob '!dist'",
  "--glob '!build'",
  "--glob '!coverage'",
  "--glob '!vendor'",
  "--glob '!target'",
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

function section(title, body) {
  console.log(`## ${title}\n`);
  console.log(body || "No candidates found.");
  console.log("");
}

function rg(pattern, extra = "") {
  const output = run(`rg -n ${DEFAULT_GLOBS} ${extra} ${JSON.stringify(pattern)} .`);
  if (!output) return "";
  const lines = output.split("\n");
  const shown = lines.slice(0, 120).join("\n");
  return lines.length > 120 ? `${shown}\n... truncated ${lines.length - 120} lines` : shown;
}

console.log("# Aggregation And Port Inspection Report\n");
console.log(`Working directory: ${process.cwd()}\n`);

const root = run("git rev-parse --show-toplevel");
section("Repository", root || "Not a git repository.");

const status = run("git status --short");
section("Git Status", status || "Clean or unavailable.");

const markers = [
  "package.json",
  "pnpm-lock.yaml",
  "yarn.lock",
  "vite.config.ts",
  "tsconfig.json",
  "go.mod",
  "Cargo.toml",
  "pyproject.toml",
  "pom.xml",
  "build.gradle",
].filter(existsSync);
section("Project Markers", markers.join("\n"));

section(
  "Generic Aggregation Names",
  rg("\\b(getDashboardData|getOverview|getSummary|getStats|getReport|queryAll|searchEverything|findAllWith|loadOverview|buildOverview|aggregate[A-Z][A-Za-z0-9_]*)\\b")
);

section(
  "Heavy SQL Or Query Joins",
  rg("\\b(JOIN|LEFT JOIN|RIGHT JOIN|INNER JOIN|FULL JOIN)\\b", "--glob '*.{sql,ts,tsx,js,jsx,go,java,kt,py,rs,cs}'")
);

section(
  "Broad ORM Graph Loading",
  rg("\\b(include|select|relations|preload|populate|withGraphFetched|leftJoinAndSelect|joinWith|Prefetch|select_related|joinedload)\\b")
);

section(
  "Many Optional Filters",
  rg("\\b(filters?|query|params|criteria|condition)\\b.*\\b(where|andWhere|orWhere|filter|if)\\b|\\bif\\s*\\([^)]*(filters?|query|params|criteria)")
);

section(
  "Mixed Result Assembly",
  rg("return\\s*\\{[^}]*\\b(users?|orders?|projects?|tasks?|stats|summary|metrics|notifications|settings|permissions|roles|teams|reports|charts|logs)\\b")
);

section(
  "Broad Repository Or Service Ports",
  rg("\\b(interface|type|class)\\s+[A-Za-z0-9_]*(Repository|Repo|Store|Service|Port|Gateway)\\b", "--glob '*.{go,java,kt,ts,tsx,js,jsx,cs,py}'")
);

section(
  "Mixed Command Query Responsibility Names",
  rg("\\b(create|insert|save|update|delete|remove|addMember|removeMember|join|leave|find|get|list|query|search|lookup|exists|count|unique|check[A-Z][A-Za-z0-9_]*)\\b", "--glob '*.{go,java,kt,ts,tsx,js,jsx,cs,py}'")
);

console.log("## Reading Guidance\n");
console.log("Treat every match as a candidate only. Confirm by reading the owning interface/function, implementations, callers, tests, data model, and API contract before recommending a split. For broad repositories or services, classify methods by command, member management, read model query, lookup, uniqueness/existence check, and policy check before proposing smaller ports.");
