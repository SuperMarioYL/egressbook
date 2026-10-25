package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCollectReportsMissingHarnessGap guards the product's central honesty
// claim: a harness whose log root does not exist must surface as an explicit
// gap, not silently as "0 sessions found".
func TestCollectReportsMissingHarnessGap(t *testing.T) {
	home := t.TempDir()
	zcodeDir := filepath.Join(home, ".zcode", "cli", "rollout")
	if err := os.MkdirAll(zcodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sample, err := os.ReadFile(filepath.Join("..", "..", "testdata", "samples", "zcode_sample.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(zcodeDir, "model-io-sess-test.jsonl"), sample, 0o644); err != nil {
		t.Fatal(err)
	}

	events, _, _, gaps := collect(home)
	if len(events) == 0 {
		t.Fatal("expected zcode events from the copied sample")
	}
	for _, harness := range []string{"codex", "claude-code"} {
		if joined := strings.Join(gaps[harness], " | "); !strings.Contains(joined, "no log dir") {
			t.Errorf("gaps[%s] = %q, want a missing-log-dir gap", harness, joined)
		}
	}
	if len(gaps["zcode"]) != 0 {
		t.Errorf("gaps[zcode] = %q, want none", gaps["zcode"])
	}

	// The same gaps must reach the ledger's coverage rows.
	led := buildSigned(home, filepath.Join(home, ".egressbook"), nil)
	for _, c := range led.Harnesses {
		if c.Harness == "codex" && len(c.Gaps) == 0 {
			t.Errorf("ledger coverage for codex carries no gaps: %+v", c)
		}
	}
}

func TestParseSinceRejectsInvalid(t *testing.T) {
	for _, s := range []string{"0h", "-1h", "0d", "-3w", "999999999999999d"} {
		if d, err := parseSince(s); err == nil {
			t.Errorf("parseSince(%q) = %v, want error", s, d)
		}
	}
	for s, want := range map[string]time.Duration{
		"72h": 72 * time.Hour,
		"2d":  2 * 24 * time.Hour,
		"2w":  2 * 7 * 24 * time.Hour,
	} {
		got, err := parseSince(s)
		if err != nil || got != want {
			t.Errorf("parseSince(%q) = %v, %v; want %v, nil", s, got, err, want)
		}
	}
}

// TestVersionSurfacesAgreeWithVERSION is the lockstep guard: VERSION is the
// source of truth and every user-visible surface must agree with it.
func TestVersionSurfacesAgreeWithVERSION(t *testing.T) {
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	v := strings.TrimSpace(string(raw))
	if v == "" {
		t.Fatal("VERSION is empty")
	}
	if version != v {
		t.Errorf("main.version = %q, VERSION = %q", version, v)
	}
	for _, f := range []string{"README.md", "README.zh-CN.md"} {
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "badge/version-"+v+"-") {
			t.Errorf("%s carries no version-%s badge", f, v)
		}
	}
	ch, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ch), "\n## "+v+" ") {
		t.Errorf("CHANGELOG.md has no '## %s' section", v)
	}
	db, err := os.ReadFile(filepath.Join(root, "docs", "demo-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var demo struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(db, &demo); err != nil {
		t.Fatal(err)
	}
	if demo.Version != v {
		t.Errorf("docs/demo-results.json version = %q, VERSION = %q", demo.Version, v)
	}
}
