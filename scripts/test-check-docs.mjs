#!/usr/bin/env node
// Fixture-based self-test for scripts/check-docs.mjs. It proves the
// code-span rule, plus the repository's baseline valid and dead-link cases.
// Zero dependencies; Node >= 24.
import {
  cpSync,
  mkdirSync,
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const scriptsDir = dirname(fileURLToPath(import.meta.url));
const realChecker = join(scriptsDir, "check-docs.mjs");

const THRESHOLDS = JSON.stringify({
  never_relax: true,
  docs_index_coverage: 1.0,
  link_integrity: 1.0,
  symlink_resolution: 1.0,
});

const workDir = mkdtempSync(join(tmpdir(), "check-docs-test-"));
let failures = 0;

function buildFixture(name) {
  const root = join(workDir, name);
  mkdirSync(join(root, "scripts"), { recursive: true });
  cpSync(realChecker, join(root, "scripts", "check-docs.mjs"));
  writeFileSync(join(root, ".coverage-thresholds.json"), THRESHOLDS);

  for (const cat of ["adr", "design", "specs", "plans"]) {
    mkdirSync(join(root, "docs", cat), { recursive: true });
  }
  mkdirSync(join(root, ".agents", "skills", "example"), { recursive: true });

  writeFileSync(join(root, "docs", "adr", "0001-example.md"), "# ADR 0001\n\nExample decision.\n");
  writeFileSync(join(root, "docs", "design", "example.md"), "# Design\n\nExample design doc.\n");
  writeFileSync(
    join(root, "docs", "specs", "pr-acceptance-metric.md"),
    "# PR Acceptance Metric\n\n## Definition\n\nExample.\n\n## Rules\n\nExample.\n",
  );
  writeFileSync(join(root, "docs", "plans", "example.md"), "# Plan\n\nExample plan.\n");
  writeFileSync(
    join(root, "docs", "README.md"),
    [
      "# Docs index",
      "",
      "- adr/0001-example.md",
      "- design/example.md",
      "- design/quality-loop.md",
      "- specs/pr-acceptance-metric.md",
      "- plans/example.md",
      "",
    ].join("\n"),
  );
  mkdirSync(join(root, ".github", "workflows"), { recursive: true });
  writeFileSync(
    join(root, ".github", "workflows", "ci.yml"),
    "- name: Validate .goreleaser.yaml\n        if: env.GORELEASER_KEY != ''\n" +
      "- name: Skip goreleaser check (missing GORELEASER_KEY)\n        if: env.GORELEASER_KEY == ''\n" +
      "GORELEASER_KEY unavailable for this run; skipping goreleaser check.\n",
  );
  mkdirSync(join(root, "docs", "design"), { recursive: true });
  writeFileSync(
    join(root, "docs", "design", "quality-loop.md"),
    "job reports on every trigger\njob succeeds without validation\nSecretless runs do not validate\n",
  );
  writeFileSync(join(root, ".agents", "skills", "example", "SKILL.md"), "# Example\n");

  return root;
}

function run(root) {
  return spawnSync(process.execPath, [join(root, "scripts", "check-docs.mjs")], {
    cwd: root,
    encoding: "utf8",
  });
}

function assertCase(name, text, wantCode) {
  const root = buildFixture(name);
  writeFileSync(join(root, ".agents", "skills", "example", "SKILL.md"), text);
  const result = run(root);
  const output = `${result.stdout}\n${result.stderr}`;
  const codeOk = result.status === wantCode;
  const diagnosticOk = wantCode === 0 || output.includes("-> missing.md does not resolve");
  if (codeOk && diagnosticOk) {
    console.log(`PASS: ${name}`);
  } else {
    failures++;
    console.log(`FAIL: ${name} (exit ${result.status}, want ${wantCode}; diagnostic ${diagnosticOk ? "found" : "missing"})`);
    console.log(output);
  }
}

assertCase("valid-fixture-passes", "# Example\n", 0);
assertCase("plain-dead-link-fails", "[x](missing.md)\n", 1);

assertCase("escaped-backticks-guard-a-real-link", "\\` [x](missing.md) \\`\n", 1);
assertCase("spans-match-by-backtick-run-length", "`` a ` `` [x](missing.md)\n", 1);
assertCase("unclosed-double-backtick-run-guards-a-real-link", "`` [x](missing.md) `\n", 1);
assertCase("adjacent-headings-with-unpaired-backticks", "# First `\n# [x](missing.md) `\n", 1);
assertCase("span-that-wraps-lines-guards-the-outside-link", "`code\ncontinued` [x](missing.md) `tail`\n", 1);
assertCase("standalone-gfm-table-row", "| ` | [x](missing.md) ` |\n", 1);
assertCase("gfm-table-with-header-and-delimiter", "| Code | Link |\n| --- | --- |\n| ` | [x](missing.md) ` |\n", 1);
assertCase("raw-html-with-a-backtick", "<span title=\"`\">[x](missing.md)`</span>\n", 1);
assertCase("email-autolink-with-a-backtick", "<1`@example.com> [x](missing.md) `\n", 1);
assertCase("code-span-inside-link-text", "[`code`](missing.md)\n", 1);
assertCase(
  "go-generic-calls-in-inline-code-spans",
  "- `errors.AsType[T](err)`\n\n`reflect.TypeAssert[T](v)`\n",
  0,
);
assertCase("double-backtick-span-exact-run-length", "`` a ` [x](missing.md) ``\n", 0);
assertCase("less-than-without-greater-than", "Go <1.26 and `errors.AsType[T](err)`\n", 0);
assertCase("naive-fence-removal-keeps-real-link", "` ```x` y``` [l](missing.md) `\n", 1);
assertCase("list-item-fence-keeps-later-real-link", "- ```\n  code\n  ```\n[x](missing.md)\n\n```\nz\n```\n", 1);
assertCase("backtick-in-link-destination", "[a](https://example.com/a`b) [x](missing.md) `tail\n", 1);
assertCase("backtick-in-link-title", "[a](u \"t`\") [x](missing.md) `tail\n", 1);
assertCase("backtick-in-full-reference-label", "[a][b`c] [x](missing.md) `tail\n\n[b`c]: https://example.com\n", 1);
assertCase("unicode-blank-line-keeps-real-link", "`head\n\u00a0\nend` [x](missing.md) `tail\n", 1);
assertCase("lone-cr-line-ending-keeps-real-link", "`head\r\r[x](missing.md) `tail\r", 1);
assertCase("crlf-line-ending-with-paired-span", "Use `errors.AsType[T](err)` here.\r\n", 0);
assertCase("gfm-www-autolink-consumes-backtick", "www.example.com/a`b [x](missing.md) `tail\n", 1);
assertCase("gfm-https-autolink-consumes-backtick", "https://example.com/a`b [x](missing.md) `tail\n", 1);
assertCase("inline-math-consumes-backtick", "$a`$ [x](missing.md) `tail\n", 1);
assertCase(
  "autolink-and-go-generic-markers-in-code-spans",
  "See `https://example.com/x` and `errors.AsType[T](err)`.\n",
  0,
);

rmSync(workDir, { recursive: true, force: true });

if (failures > 0) {
  console.error(`test-check-docs: ${failures} assertion(s) failed`);
  process.exit(1);
}
console.log("test-check-docs: all assertions passed");
