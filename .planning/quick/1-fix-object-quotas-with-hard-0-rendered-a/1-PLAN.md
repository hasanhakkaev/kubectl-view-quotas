---
phase: quick-1
plan: 01
type: execute
wave: 1
depends_on: []
files_modified: [pkg/plugin/plugin.go, pkg/plugin/plugin_test.go]
autonomous: true
requirements: [QUICK-1]

must_haves:
  truths:
    - "A ResourceQuota whose hard values are all 0 (e.g. persistentvolumeclaims: 0, services.loadbalancers: 0, services.nodeports: 0) renders one table row per resource, not an empty table"
    - "A row with used=0 and hard=0 shows USED=0, HARD=0, a full progress bar, 100.0%, in the red (at-limit) tier"
    - "A row with used>0 and hard=0 renders as over-limit (colorOver tier) with no Inf, NaN, or panic in the output"
    - "Existing nonzero-hard rendering is unchanged (e.g. 5 of 10 still renders 50.0% green)"
  artifacts:
    - path: "pkg/plugin/plugin.go"
      provides: "hard=0 rows rendered instead of skipped; divide-by-zero-safe usage calculation"
      contains: "buildQuotaRows"
    - path: "pkg/plugin/plugin_test.go"
      provides: "Unit tests for hard=0 row building and usage rendering"
      min_lines: 40
  key_links:
    - from: "pkg/plugin/plugin.go printQuota"
      to: "buildQuotaRows"
      via: "row construction extracted into pure, testable function"
      pattern: "buildQuotaRows\\(quota"
    - from: "pkg/plugin/plugin.go resourceUsage"
      to: "chooseColor/progressBar"
      via: "guarded percentage calculation (no used/hard division when hard==0)"
      pattern: "hard == 0"
---

<objective>
Fix the regression from commit 70bfb42: object quotas whose spec.hard values are all 0 currently render as tables with headers but zero rows. hard=0 is a meaningful quota (the resource is forbidden in that namespace), so these rows must be shown with USED=0, HARD=0, and a full red bar at 100.0% — treated as at-limit — without dividing by zero or printing `<none>`.

Purpose: Users with the common "object quota" pattern (persistentvolumeclaims: 0, services.loadbalancers: 0, services.nodeports: 0) currently see empty tables and cannot tell those resources are blocked.
Output: Fixed pkg/plugin/plugin.go plus new pkg/plugin/plugin_test.go covering the hard=0 paths.
</objective>

<execution_context>
@/Users/hasanhakkaev/.claude/get-shit-done/workflows/execute-plan.md
@/Users/hasanhakkaev/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md
@pkg/plugin/plugin.go

Locked decision (STATE.md): hard=0 resources must be shown, not skipped — hard=0 means "forbidden", a real quota state users need to see.

<interfaces>
Current code in pkg/plugin/plugin.go relevant to the fix:

```go
type quotaRow struct {
	resource string
	used     string
	hard     string
	usage    string // raw visual string before colorization
	color    lipgloss.TerminalColor
}

// Inside printQuota(quota v1.ResourceQuota) — the buggy skip:
//   hardFloat := hard.AsApproximateFloat64()
//   if hardFloat == 0 { continue }   // <-- removes all-zero object quotas entirely

// resourceUsage returns the AdaptiveColor and a fixed-width usage string:
// "▉▉▉▉▉     50.0%". Percentage right-aligned in a 6-char field ("%5.1f%%").
func resourceUsage(used, hard float64) (lipgloss.TerminalColor, string)

// progressBar clamps pct to [0,100] and renders a 10-char eighth-block bar.
func progressBar(pct float64) string

// chooseColor tiers: pct > 100 -> colorOver, >= 90 -> colorRed,
// >= 75 -> colorYellow, else colorGreen.
func chooseColor(pct float64) lipgloss.TerminalColor
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Render hard=0 quota rows with safe usage calculation (test-first)</name>
  <files>pkg/plugin/plugin.go, pkg/plugin/plugin_test.go</files>
  <behavior>
    Write pkg/plugin/plugin_test.go FIRST (RED), covering:
    - Test 1 (buildQuotaRows, all-zero object quota): a v1.ResourceQuota with Status.Hard = {persistentvolumeclaims: "0", services.loadbalancers: "0", services.nodeports: "0"} and Status.Used all "0" produces 3 rows (sorted by resource name), each with used == "0" and hard == "0" — NOT zero rows.
    - Test 2 (resourceUsage(0, 0)): returns color == colorRed and a usage string containing "100.0%" and a full 10-rune "█" bar; string must not contain "Inf", "NaN", or "<none>".
    - Test 3 (resourceUsage(1, 0)): returns color == colorOver and a usage string with a full bar and an over-limit indicator (see action); must not contain "Inf" or "NaN".
    - Test 4 (regression, resourceUsage(5, 10)): returns colorGreen and a string containing "50.0%" (existing behavior unchanged).
    Use resource.MustParse from k8s.io/apimachinery/pkg/api/resource to build quantities.
  </behavior>
  <action>
    In pkg/plugin/plugin.go:

    1. Extract the row-building loop from printQuota into a new pure function
       `buildQuotaRows(quota v1.ResourceQuota) []quotaRow` (keeps printQuota's
       stdout side effects out of the testable path). printQuota calls it and
       renders the returned rows exactly as today.

    2. In buildQuotaRows, DELETE the `if hardFloat == 0 { continue }` skip
       introduced by commit 70bfb42. Every entry in Status.Hard gets a row.

    3. Make resourceUsage divide-by-zero safe:
       - hard == 0 && used == 0: pct = 100 → chooseColor(100) yields colorRed,
         progressBar(100) yields a full bar, text renders "100.0%". Semantics:
         0 used of 0 allowed = at-limit/blocked.
       - hard == 0 && used > 0: over-limit. Use colorOver (chooseColor with
         pct > 100) and a full bar. For the text, keep the fixed-width 6-char
         alignment of "%5.1f%%" — render ">100%" right-aligned in that field
         (e.g. fmt.Sprintf("%s %6s", progressBar(101), ">100%")). Do NOT
         format an Inf/NaN float. Do NOT print a raw fake number like 200.0%.
       - hard > 0: existing `used / hard * 100` path, unchanged.

    Keep the change minimal: no renaming of existing exported functions, no
    changes to table styling, colors, or the progressBar/chooseColor logic.
    Note gofumpt/goimports run via make bin — keep imports tidy (the test file
    will need k8s.io/apimachinery/pkg/api/resource and metav1/v1 as applicable).

    TDD flow: write plugin_test.go, run it, confirm Tests 1-3 FAIL against
    current code (Test 1 fails because rows are skipped); then implement the
    fix; then confirm all tests pass.
  </action>
  <verify>
    <automated>go test ./pkg/plugin/... -run 'TestBuildQuotaRows|TestResourceUsage' -v</automated>
  </verify>
  <done>All-zero object quota produces one row per resource; resourceUsage(0,0) = red full bar 100.0%; resourceUsage(1,0) = colorOver full bar with ">100%" style indicator; resourceUsage(5,10) unchanged at 50.0% green; no divide-by-zero, Inf, NaN, or `<none>` anywhere.</done>
</task>

<task type="auto">
  <name>Task 2: Full build, test, and vet verification</name>
  <files>pkg/plugin/plugin.go, pkg/plugin/plugin_test.go</files>
  <action>
    Run the project's standard verification suite from CLAUDE.md:
    - `make bin` (fmt + vet + build to bin/kubectl-view-quotas)
    - `make test` (go test ./pkg/... ./cmd/... with coverage)
    - `make vet`
    - If golangci-lint is installed locally, also run `golangci-lint run` and
      fix any findings in the changed files (gofumpt/goimports/gosec/revive
      are active per .golangci.yaml).
    Fix any formatting, vet, or lint issues surfaced. Do not touch files
    outside pkg/plugin/.
  </action>
  <verify>
    <automated>make bin && make test && make vet</automated>
  </verify>
  <done>make bin, make test, and make vet all exit 0 with the fix and new tests in place.</done>
</task>

</tasks>

<verification>
- `go test ./pkg/plugin/... -v` passes with new tests covering: all-zero quota row building, resourceUsage(0,0), resourceUsage(1,0), resourceUsage(5,10).
- `make bin && make test && make vet` all exit 0.
- Grep check: `grep -n "hardFloat == 0" pkg/plugin/plugin.go` returns nothing (skip removed), and `grep -n "buildQuotaRows" pkg/plugin/plugin.go` shows the extracted function wired into printQuota.
</verification>

<success_criteria>
- A ResourceQuota with all spec.hard values 0 renders a populated table (one row per resource) instead of headers with zero rows.
- hard=0/used=0 rows show USED=0, HARD=0, full red bar at 100.0%.
- hard=0/used>0 rows render in the over-limit (colorOver) tier with no Inf/NaN output.
- Nonzero-hard rendering byte-identical to current behavior.
- make bin, make test, make vet pass.
</success_criteria>

<output>
After completion, create `.planning/quick/1-fix-object-quotas-with-hard-0-rendered-a/1-SUMMARY.md`
</output>
