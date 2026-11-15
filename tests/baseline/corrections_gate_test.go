// The corrections gate (core ADR-0018, ADR-0081): the record files in
// .memory/corrections/ are checked by core's scripts/check-corrections.mjs,
// copied here byte for byte, and `make ci` runs it twice — every record's
// format, and the append-only comparison with BASE. This file guards that
// wiring: the script, the 2 Makefile targets (which `ci` must reach), the
// refusal in corrections-base that turns an unresolvable BASE into a failure,
// the Node pin both steps need, and the Verify job of the CI workflow that
// supplies BASE and a clone deep enough to hold it.
//
// A gate nothing runs is not a gate: without these checks a Makefile edit or a
// workflow edit could retire the corrections comparison with every test green.
// The checks read the tracked tree the same way the baseline rows do, and
// skip when git is unusable.
package baseline_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	correctionsScript = "scripts/check-corrections.mjs"
	// ciWorkflow is the workflow file whose Verify job runs `make ci`.
	ciWorkflow = ".github/workflows/ci.yml"
	// correctionsBase is the commit the CI Verify job compares the records
	// with: the pull request's base.
	correctionsBase = "BASE: ${{ github.event.pull_request.base.sha }}"
)

// makeRuleBody returns the recipe lines of one Makefile rule: the consecutive
// lines after `target:` that begin with a tab, without the tab. It returns nil
// when the Makefile has no such rule.
func makeRuleBody(makefile, target string) []string {
	lines := strings.Split(makefile, "\n")
	for i, line := range lines {
		if _, ok := strings.CutPrefix(line, target+":"); !ok {
			continue
		}
		var body []string
		for _, recipe := range lines[i+1:] {
			if !strings.HasPrefix(recipe, "\t") {
				break
			}
			body = append(body, strings.TrimPrefix(recipe, "\t"))
		}
		return body
	}
	return nil
}

// ciReaches reports whether the `ci` rule runs target: as a prerequisite, or
// as a name in 1 recipe line (a recursive `$(MAKE) target` call).
func ciReaches(makefile, target string) bool {
	if slices.Contains(makeRulePrereqs(makefile, "ci"), target) {
		return true
	}
	for _, line := range makeRuleBody(makefile, "ci") {
		if slices.Contains(strings.Fields(line), target) {
			return true
		}
	}
	return false
}

// workflowJobs returns the text of each job under `jobs:` of a workflow file,
// keyed by job id. A job id is a line indented by exactly 2 spaces.
func workflowJobs(text string) map[string]string {
	jobs := map[string]string{}
	inJobs := false
	current := ""
	for _, line := range strings.Split(text, "\n") {
		if !inJobs {
			inJobs = line == "jobs:"
			continue
		}
		if rest, ok := strings.CutPrefix(line, "  "); ok && !strings.HasPrefix(rest, " ") && strings.HasSuffix(rest, ":") {
			current = strings.TrimSuffix(rest, ":")
			jobs[current] = ""
			continue
		}
		if current != "" {
			jobs[current] += line + "\n"
		}
	}
	return jobs
}

func TestMakeRuleBody(t *testing.T) {
	const makefile = ".PHONY: a b\n\na: b\n\t@echo a\n\tafter\n\nb:\n\t@echo b\n"
	if got := makeRuleBody(makefile, "a"); !slices.Equal(got, []string{"@echo a", "after"}) {
		t.Errorf("makeRuleBody(a) = %q, want the 2 recipe lines", got)
	}
	if got := makeRuleBody(makefile, "b"); !slices.Equal(got, []string{"@echo b"}) {
		t.Errorf("makeRuleBody(b) = %q, want 1 recipe line", got)
	}
	if got := makeRuleBody(makefile, "c"); got != nil {
		t.Errorf("makeRuleBody(c) = %q, want nil for a missing rule", got)
	}
	if got := makeRuleBody("a: b\n", "a"); len(got) != 0 {
		t.Errorf("makeRuleBody of a rule with no recipe = %q, want empty", got)
	}
}

func TestCiReaches(t *testing.T) {
	const makefile = "ci: verify\n\t@$(MAKE) --no-print-directory other\n\t$(GO) test ./...\n"
	if !ciReaches(makefile, "verify") {
		t.Error("ciReaches(verify) = false, want true for a prerequisite")
	}
	if !ciReaches(makefile, "other") {
		t.Error("ciReaches(other) = false, want true for a recursive make call")
	}
	for _, absent := range []string{"missing", "tes", "make", "other "} {
		if ciReaches(makefile, absent) {
			t.Errorf("ciReaches(%q) = true, want false: only whole names count", absent)
		}
	}
}

func TestWorkflowJobs(t *testing.T) {
	const workflow = "name: CI\non: push\npermissions: {}\n\njobs:\n  build:\n    name: Build\n    steps:\n      - run: make ci\n\n  other:\n    name: Other\n"
	jobs := workflowJobs(workflow)
	if len(jobs) != 2 {
		t.Fatalf("workflowJobs = %d jobs, want 2", len(jobs))
	}
	if !strings.Contains(jobs["build"], "run: make ci") {
		t.Errorf("jobs[build] = %q, want the make ci step", jobs["build"])
	}
	if strings.Contains(jobs["build"], "Other") {
		t.Error("jobs[build] holds the next job's lines")
	}
	if jobs["other"] == "" {
		t.Error("jobs[other] is empty")
	}
	if got := workflowJobs("name: CI\n"); len(got) != 0 {
		t.Errorf("workflowJobs without a jobs: block = %v, want none", got)
	}
}

func TestJobInstallsNode(t *testing.T) {
	tests := []struct {
		name string
		job  string
		want bool
	}{
		{"mise install_args names node last", "    steps:\n      - name: Install pinned tools\n        with:\n          install_args: golangci-lint rust node\n", true},
		{"mise install_args names node first", "        with:\n          install_args: node\n", true},
		{"no install_args", "    steps:\n      - run: make ci\n", false},
		{"node is part of another tool's name", "        with:\n          install_args: nodejs\n", false},
		{"empty job", "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := jobInstallsNode(test.job); got != test.want {
				t.Errorf("jobInstallsNode(%q) = %v, want %v", test.job, got, test.want)
			}
		})
	}
}

// correctionsMakefileProblems reports what is wrong with the corrections
// wiring of 1 Makefile text: the 2 rules exist, the validator runs in the
// corrections-check recipe itself, `ci` reaches both rules, and corrections-base
// runs the append-only comparison and refuses a BASE that does not resolve.
//
// The validator is looked for in that rule's own recipe, never in the whole
// Makefile: corrections-base runs the same script with --base, so a search over
// the file stays green when the format check loses its command.
func correctionsMakefileProblems(makefile string) []string {
	var findings []string
	for _, target := range []string{"corrections-check", "corrections-base"} {
		if !strings.Contains(makefile, "\n"+target+":") {
			findings = append(findings, "the Makefile has no "+target+" rule")
		}
	}
	check := strings.Join(makeRuleBody(makefile, "corrections-check"), "\n")
	if !strings.Contains(check, "node scripts/check-corrections.mjs") {
		findings = append(findings, "the corrections-check recipe does not run node scripts/check-corrections.mjs"+recipeDetail(check))
	}
	// `ci` must reach both, so the gate runs where CI runs `make ci` and in a
	// developer's local gate.
	for _, target := range []string{"corrections-check", "corrections-base"} {
		if !ciReaches(makefile, target) {
			findings = append(findings, "the ci rule no longer runs "+target+", so the corrections gate stops running in make ci and in CI")
		}
	}
	base := strings.Join(makeRuleBody(makefile, "corrections-base"), "\n")
	if !strings.Contains(base, `--base "$(BASE)"`) {
		findings = append(findings, "the corrections-base recipe no longer runs the check with --base \"$(BASE)\""+recipeDetail(base))
	}
	if !strings.Contains(base, "does not resolve to a commit") {
		findings = append(findings, "the corrections-base recipe lost the refusal of a BASE that does not resolve to a commit, so an unfetched ref would pass silently"+recipeDetail(base))
	}
	return findings
}

// recipeDetail names a recipe in a finding: its lines, or that it is empty.
func recipeDetail(recipe string) string {
	if recipe == "" {
		return " (its recipe is empty)"
	}
	return " (its recipe is:\n" + recipe + ")"
}

// correctionsMakefile is a Makefile with the corrections gate wired correctly:
// the validator in the corrections-check recipe, the append-only comparison in
// corrections-base, and both reached from `ci`.
const correctionsMakefile = `ci: verify
	@$(MAKE) --no-print-directory corrections-check
	@$(MAKE) --no-print-directory corrections-base

corrections-check:
	@command -v node >/dev/null 2>&1 || { echo "mise install"; exit 1; }
	node scripts/check-corrections.mjs

corrections-base:
	@if [ -z "$(BASE)" ]; then exit 0; fi; \
	if ! git rev-parse --verify --quiet "$(BASE)^{commit}" >/dev/null; then \
		echo "FAIL: BASE $(BASE) does not resolve to a commit (fetch it first)"; \
		exit 1; \
	fi; \
	node scripts/check-corrections.mjs --base "$(BASE)"
`

// correctionsValidator is the validator line of the corrections-check recipe in
// correctionsMakefile: the removal case takes out only this line.
const correctionsValidator = "\tnode scripts/check-corrections.mjs\n"

func TestCorrectionsMakefileProblems(t *testing.T) {
	tests := []struct {
		name     string
		makefile string
		want     string // "" means no finding
	}{
		{
			name:     "the gate is wired",
			makefile: correctionsMakefile,
		},
		{
			name:     "corrections-check no longer runs the validator",
			makefile: strings.Replace(correctionsMakefile, correctionsValidator, "", 1),
			want:     "the corrections-check recipe does not run node scripts/check-corrections.mjs",
		},
		{
			name:     "the corrections-check rule is gone",
			makefile: strings.Replace(correctionsMakefile, "corrections-check:\n", "", 1),
			want:     "the Makefile has no corrections-check rule",
		},
		{
			name:     "ci no longer runs corrections-check",
			makefile: strings.Replace(correctionsMakefile, "\t@$(MAKE) --no-print-directory corrections-check\n", "", 1),
			want:     "the ci rule no longer runs corrections-check",
		},
		{
			name:     "ci no longer runs corrections-base",
			makefile: strings.Replace(correctionsMakefile, "\t@$(MAKE) --no-print-directory corrections-base\n", "", 1),
			want:     "the ci rule no longer runs corrections-base",
		},
		{
			name:     "corrections-base compares no base",
			makefile: strings.Replace(correctionsMakefile, `--base "$(BASE)"`, "--base", 1),
			want:     `the corrections-base recipe no longer runs the check with --base`,
		},
		{
			name:     "corrections-base lost the refusal",
			makefile: strings.Replace(correctionsMakefile, "does not resolve to a commit", "is unreachable", 1),
			want:     "the corrections-base recipe lost the refusal",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := correctionsMakefileProblems(test.makefile)
			if test.want == "" {
				noFindings(t, "correctionsMakefileProblems", findings)
				return
			}
			for _, finding := range findings {
				if strings.Contains(finding, test.want) {
					return
				}
			}
			t.Errorf("correctionsMakefileProblems = %v, want a finding that names %q", findings, test.want)
		})
	}
}

// TestCorrectionsGate fails when the repository can no longer check its
// records: the script is gone, `make ci` stopped running the 2 corrections
// targets, corrections-base lost its refusal of an unresolvable BASE, the
// Node pin the steps need is gone, or the Verify job stopped supplying BASE
// and the history it names.
func TestCorrectionsGate(t *testing.T) {
	r := loadRepo(t)
	r.require(t, "Makefile", "mise.toml", "mise.lock", ciWorkflow)
	if _, err := os.Stat(filepath.Join(r.root, correctionsScript)); err != nil {
		t.Fatalf("%s is missing (core's script, copied here byte for byte): %v", correctionsScript, err)
	}
	noFindings(t, "Makefile (corrections gate)", correctionsMakefileProblems(r.read(t, "Makefile")))

	// Both steps run node, so the gate needs the pin (core ADR-0043).
	if !strings.Contains(r.read(t, "mise.toml"), `node = "24"`) {
		t.Error(`mise.toml does not pin node = "24", which the corrections gate runs on`)
	}
	if !strings.Contains(r.read(t, "mise.lock"), "[[tools.node]]") {
		t.Error("mise.lock has no [[tools.node]] entry, so mise install cannot verify the Node pin")
	}

	// The workflow job that runs `make ci`: exactly 1, and it supplies BASE and
	// a clone that holds the base commit.
	var gate, gateID string
	for id, job := range workflowJobs(r.read(t, ciWorkflow)) {
		if !strings.Contains(job, "run: make ci") {
			continue
		}
		if gate != "" {
			t.Fatalf("%s runs make ci in jobs %s and %s, want 1", ciWorkflow, gateID, id)
		}
		gate, gateID = job, id
	}
	if gate == "" {
		t.Fatalf("%s has no job that runs make ci", ciWorkflow)
	}
	if !strings.Contains(gate, correctionsBase) {
		t.Errorf("job %s of %s does not set `%s`, so the corrections comparison is skipped in CI", gateID, ciWorkflow, correctionsBase)
	}
	if !strings.Contains(gate, "fetch-depth: 0") {
		t.Errorf("job %s of %s checks out no full history (fetch-depth: 0 is missing), so the base commit of the comparison may be absent", gateID, ciWorkflow)
	}
	if !strings.Contains(gate, "persist-credentials: false") {
		t.Errorf("job %s of %s lost persist-credentials: false (baseline row B5)", gateID, ciWorkflow)
	}
	if !jobInstallsNode(gate) {
		t.Errorf("job %s of %s does not install node through mise (install_args), so the corrections steps cannot run there", gateID, ciWorkflow)
	}
}

// jobInstallsNode reports whether a workflow job's mise-action step names node
// in install_args, in any position.
func jobInstallsNode(job string) bool {
	for _, line := range strings.Split(job, "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "install_args:"); ok {
			if slices.Contains(strings.Fields(value), "node") {
				return true
			}
		}
	}
	return false
}
