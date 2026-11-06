#!/usr/bin/env node
// Docs-integrity gate: index coverage, relative-link integrity, symlink
// resolution, and release-config workflow/docs consistency — thresholds in
// .coverage-thresholds.json (std ADR-0001, pattern from frostyard/core
// ADR-0029; the never_relax guardrail follows core ADR-0019). Zero
// dependencies; Node >= 24. Run: node scripts/check-docs.mjs
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

// Code-span rule copied from frostyard/core scripts/check-docs.mjs; keep the 2 copies the same.
// Inline code spans, as [start, end) ranges of the ORIGINAL text (never of a
// fence-stripped text: a removed chunk can join two backticks into a false
// span). CR, LF and CRLF end a line; only spaces and tabs make a blank line.
// Spans count only in a safe run of non-blank lines, so a link-shaped match is
// skipped only when CommonMark would render it as code. A run is unsafe, and
// all its links stay checked, when it has:
// - a `|` (a GFM table cell boundary splits inline contexts);
// - a backslash before a backtick (an escaped backtick is not a delimiter);
// - a `<` with a later `>` (raw HTML and autolinks bind before code spans);
// - a line with an unpaired backtick run (a span may wrap lines, or a heading
//   or other block may end it). Each line of a safe run pairs runs of equal
//   length left to right, which is CommonMark's own pairing;
// - a link, GFM autolink or inline-math marker outside a computed span, when
//   a backtick comes later in the run. The marker starts are `](`, `][`, `www.`,
//   `://`, or `$`. These constructs can consume a backtick as link destination,
//   title, reference label, autolink path or math text, so it is not a span
//   delimiter. A marker inside a real span does not trigger this rule; this keeps
//   code such as `AsType[T](err)` and `See `https://example.com/x`` safe.
function codeSpanRanges(text) {
  const lines = [];
  const lineEnding = /\r\n|\r|\n/g;
  let previousEnd = 0;
  let ending;
  while ((ending = lineEnding.exec(text)) !== null) {
    lines.push({ text: text.slice(previousEnd, ending.index), start: previousEnd });
    previousEnd = ending.index + ending[0].length;
  }
  lines.push({ text: text.slice(previousEnd), start: previousEnd });

  const ranges = [];
  let lineIndex = 0;
  while (lineIndex < lines.length) {
    if (/^[ \t]*$/.test(lines[lineIndex].text)) {
      lineIndex++;
      continue;
    }

    const runStartIndex = lineIndex;
    while (lineIndex < lines.length && !/^[ \t]*$/.test(lines[lineIndex].text)) lineIndex++;
    const runLines = lines.slice(runStartIndex, lineIndex);
    const runStart = runLines[0].start;
    const lastLine = runLines[runLines.length - 1];
    const runEnd = lastLine.start + lastLine.text.length;
    const runText = text.slice(runStart, runEnd);

    let unsafe = false;
    if (runLines.some((line) => line.text.includes("|"))) unsafe = true;
    if (!unsafe && runText.includes("\\`")) unsafe = true;

    if (!unsafe) {
      for (let i = runText.indexOf("<"); i !== -1; i = runText.indexOf("<", i + 1)) {
        if (runText.indexOf(">", i + 1) !== -1) {
          unsafe = true;
          break;
        }
      }
    }

    const runRanges = [];
    if (!unsafe) {
      for (const line of runLines) {
        let pos = 0;
        let lineUnpaired = false;
        while (pos < line.text.length) {
          if (line.text[pos] !== "`") {
            pos++;
            continue;
          }

          const opener = pos;
          while (pos < line.text.length && line.text[pos] === "`") pos++;
          const length = pos - opener;

          let search = pos;
          let closerStart = -1;
          let closerEnd = -1;
          while (search < line.text.length) {
            if (line.text[search] !== "`") {
              search++;
              continue;
            }
            const candidate = search;
            while (search < line.text.length && line.text[search] === "`") search++;
            if (search - candidate === length) {
              closerStart = candidate;
              closerEnd = search;
              break;
            }
          }

          if (closerStart === -1) {
            lineUnpaired = true;
            break;
          }

          runRanges.push([line.start + opener, line.start + closerEnd]);
          pos = closerEnd;
        }

        if (lineUnpaired) {
          unsafe = true;
          break;
        }
      }
    }

    if (!unsafe) {
      for (let i = 0; i < runText.length; i++) {
        const markerEnd = consumedBacktickMarkerEnd(runText, i);
        if (markerEnd === -1) continue;
        const position = runStart + i;
        if (isInCodeSpan(runRanges, position, position + 1)) continue;
        if (runText.indexOf("`", markerEnd) !== -1) {
          unsafe = true;
          break;
        }
      }
    }

    if (!unsafe) ranges.push(...runRanges);
  }

  return ranges;
}

function consumedBacktickMarkerEnd(text, index) {
  if (text.startsWith("](", index) || text.startsWith("][", index)) return index + 2;
  if (text.startsWith("www.", index)) return index + 4;
  if (text.startsWith("://", index)) return index + 3;
  if (text[index] === "$") return index + 1;
  return -1;
}

function isInCodeSpan(ranges, start, end) {
  return ranges.some(([rangeStart, rangeEnd]) => rangeStart <= start && end <= rangeEnd);
}

function stripWithMap(text, regex) {
  const removedIntervals = [];
  let nextOriginal = 0;
  for (const match of text.matchAll(regex)) {
    const start = match.index;
    const end = start + match[0].length;
    if (start < nextOriginal) continue;
    removedIntervals.push([start, end]);
    nextOriginal = end;
  }

  const boundaries = [];
  let stripped = "";
  let originalCursor = 0;
  for (const [start, end] of removedIntervals) {
    const kept = text.slice(originalCursor, start);
    boundaries[stripped.length] = originalCursor;
    for (let i = 0; i < kept.length; i++) {
      stripped += kept[i];
      boundaries[stripped.length] = originalCursor + i + 1;
    }
    originalCursor = end;
  }
  const kept = text.slice(originalCursor);
  boundaries[stripped.length] = originalCursor;
  for (let i = 0; i < kept.length; i++) {
    stripped += kept[i];
    boundaries[stripped.length] = originalCursor + i + 1;
  }

  return {
    text: stripped,
    toOriginal(index) {
      return boundaries[index] ?? text.length;
    },
  };
}

// ---- 2. Link integrity: relative md links resolve in every real Markdown file. ----
let linksTotal = 0;
let linksOk = 0;
for (const file of mdFiles) {
  // Strip fenced code blocks so example links aren't checked; a match is
  // skipped as inline code only when its original range is inside 1 span.
  const originalText = readFileSync(file, "utf8");
  const fenceStripped = stripWithMap(originalText, /```[\s\S]*?```/g);
  const text = fenceStripped.text;
  const codeSpanRangesInText = codeSpanRanges(originalText);
  for (const m of text.matchAll(/\[[^\]]*\]\(([^)\s]+)\)/g)) {
    const originalStart = fenceStripped.toOriginal(m.index);
    const originalEnd = fenceStripped.toOriginal(m.index + m[0].length);
    if (
      originalEnd - originalStart === m[0].length &&
      isInCodeSpan(codeSpanRangesInText, originalStart, originalEnd)
    ) continue;
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

// ---- 5. Release-config docs match the workflow's secret guards. ----
const workflow = readFileSync(join(root, ".github/workflows/ci.yml"), "utf8");
const qualityLoop = readFileSync(join(root, "docs/design/quality-loop.md"), "utf8");
for (const guard of [
  "- name: Validate .goreleaser.yaml\n        if: env.GORELEASER_KEY != ''",
  "- name: Skip goreleaser check (missing GORELEASER_KEY)\n        if: env.GORELEASER_KEY == ''",
]) {
  if (workflow.split(guard).length - 1 !== 1) {
    failures.push(`release-config: workflow must contain exactly one guarded branch "${guard}"`);
  }
}
const skipMessage = "GORELEASER_KEY unavailable for this run; skipping goreleaser check.";
if (!workflow.includes(skipMessage)) {
  failures.push(`release-config: workflow is missing the explicit skip message "${skipMessage}"`);
}
for (const wording of ["job reports on every trigger", "job succeeds without validation", "Secretless runs do not validate"]) {
  if (!qualityLoop.includes(wording)) {
    failures.push(`release-config: docs/design/quality-loop.md is missing conditional wording "${wording}"`);
  }
}

// ---- 6. Cross-arch example build: the workflow job and the prose agree. ----
// `make ci` and the workflow's Examples Cross-Arch Build job (`examples-build`)
// build the same _examples/ programs, off the same scripts/example-dirs.sh
// enumeration, for the same linux/amd64 and linux/arm64 pair. The job was added
// after the prose that called that build local-only, and nothing caught the
// contradiction for two days — so pin it here, in the style of section 5 above:
// while the job exists, no operator-facing text may describe the build as
// running in only one of the two places, and the entry-point design doc must
// name the job.
if (workflow.includes("\n  examples-build:")) {
  const localOnly = ["runs only here", "runs only locally", "`make ci` only"];
  for (const file of ["Makefile", "AGENTS.md", "docs/org-adrs.md"]) {
    const text = readFileSync(join(root, file), "utf8");
    for (const phrase of localOnly) {
      if (text.includes(phrase)) {
        failures.push(`cross-arch: ${file} calls the cross-arch _examples/ build local-only ("${phrase}") while .github/workflows/ci.yml runs it in the examples-build job`);
      }
    }
  }
  if (!readFileSync(join(root, "docs/design/overview.md"), "utf8").includes("Examples Cross-Arch Build")) {
    failures.push("cross-arch: docs/design/overview.md does not name the Examples Cross-Arch Build job that .github/workflows/ci.yml defines");
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
