#!/usr/bin/env node
// Docs-integrity gate: index coverage, relative-link integrity, symlink
// resolution — thresholds in .coverage-thresholds.json (std ADR-0001, pattern
// from frostyard/core ADR-0029; the never_relax guardrail follows core
// ADR-0019). Zero dependencies; Node >= 20. Run: node scripts/check-docs.mjs
import { readFileSync, readdirSync, lstatSync, existsSync, realpathSync } from "node:fs";
import { join, dirname, resolve, relative, sep } from "node:path";

const root = resolve(dirname(new URL(import.meta.url).pathname), "..");
const thresholds = JSON.parse(readFileSync(join(root, ".coverage-thresholds.json"), "utf8"));

if (thresholds.never_relax !== true) {
  console.error("FATAL: .coverage-thresholds.json never_relax must be true (ADR-0019: the loop may tighten, never loosen).");
  process.exit(1);
}

const failures = [];
const rate = (pass, total) => (total === 0 ? 1 : pass / total);

// ---- 1. Index coverage: every doc in the four categories is indexed. ----
const indexText = readFileSync(join(root, "docs/README.md"), "utf8");
const categories = ["adr", "design", "specs", "plans"];
let docsTotal = 0;
let docsIndexed = 0;
for (const cat of categories) {
  for (const name of readdirSync(join(root, "docs", cat))) {
    if (!name.endsWith(".md") || name === "TEMPLATE.md") continue;
    docsTotal++;
    if (indexText.includes(`${cat}/${name}`)) docsIndexed++;
    else failures.push(`index: docs/${cat}/${name} has no line in docs/README.md`);
  }
}

// ---- Tree walk shared by the link and symlink checks. ----
// dist/ is GoReleaser's output; vendor/ would be a Go module vendor tree.
// A symlinked directory (.claude/skills -> .agents/skills) is recorded as a
// symlink and not descended, so aliased trees are never double-counted.
const SKIP_DIRS = new Set([".git", "node_modules", "dist", "vendor"]);
const symlinks = [];
const mdFiles = [];
(function walk(dir) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    const stat = lstatSync(path);
    if (stat.isSymbolicLink()) {
      // Aliases are not docs: they carry no cross-link obligations (ADR-0001),
      // and their content is already link-checked at its canonical path — a
      // relative link cannot resolve from both the canonical path and the
      // alias path — so symlinked Markdown stays out of the link denominator.
      symlinks.push(path);
    } else if (stat.isDirectory()) {
      if (!SKIP_DIRS.has(name)) walk(path);
    } else if (name.endsWith(".md") && name !== "TEMPLATE.md") {
      // TEMPLATE.md files hold deliberate placeholder links (NNNN-….md).
      mdFiles.push(path);
    }
  }
})(root);
mdFiles.sort();

// ---- 2. Link integrity: relative md links resolve in every real Markdown file. ----
let linksTotal = 0;
let linksOk = 0;
for (const file of mdFiles) {
  // Strip fenced code blocks so example links aren't checked.
  const text = readFileSync(file, "utf8").replace(/```[\s\S]*?```/g, "");
  for (const m of text.matchAll(/\[[^\]]*\]\(([^)\s]+)\)/g)) {
    const target = m[1];
    if (/^[a-z][a-z+.-]*:/i.test(target) || target.startsWith("#")) continue; // external or anchor
    linksTotal++;
    const path = resolve(dirname(file), target.split("#")[0]);
    if (existsSync(path)) linksOk++;
    else failures.push(`link: ${relative(root, file)} -> ${target} does not resolve`);
  }
}

// ---- 3. Symlink resolution: every repo symlink resolves inside the repo. ----
let linksResolved = 0;
for (const path of symlinks) {
  try {
    const real = realpathSync(path);
    if (real === root || real.startsWith(root + sep)) linksResolved++;
    else failures.push(`symlink: ${relative(root, path)} escapes the repo (${real})`);
  } catch {
    failures.push(`symlink: ${relative(root, path)} is broken`);
  }
}

// ---- 4. Pinned headings: metric prose cannot drift silently (core ADR-0019). ----
const metric = readFileSync(join(root, "docs/specs/pr-acceptance-metric.md"), "utf8");
for (const heading of ["## Definition", "## Rules"]) {
  if (!metric.split("\n").includes(heading)) {
    failures.push(`pin: docs/specs/pr-acceptance-metric.md is missing the "${heading}" heading`);
  }
}

// ---- Report against thresholds. ----
const results = {
  docs_index_coverage: rate(docsIndexed, docsTotal),
  link_integrity: rate(linksOk, linksTotal),
  symlink_resolution: rate(linksResolved, symlinks.length),
};
for (const failure of failures) console.error(`FAIL ${failure}`);
let ok = failures.length === 0;
for (const [key, value] of Object.entries(results)) {
  const required = thresholds[key];
  const met = value >= required;
  if (!met) ok = false;
  console.log(`${met ? "ok  " : "FAIL"} ${key}: ${value.toFixed(3)} (required ${required})`);
}
console.log(`checked: ${docsTotal} docs, ${linksTotal} links, ${symlinks.length} symlinks`);
process.exit(ok ? 0 : 1);
