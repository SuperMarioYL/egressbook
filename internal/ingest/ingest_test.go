package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SuperMarioYL/egressbook/internal/model"
)

func TestParseZcodeSample(t *testing.T) {
	evs, gaps, err := ParseZcode(filepath.Join("..", "..", "testdata", "samples", "zcode_sample.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) == 0 {
		t.Fatal("expected events from zcode sample")
	}
	var sys, tool, git int
	for _, e := range evs {
		if e.Harness != "zcode" {
			t.Errorf("harness = %q", e.Harness)
		}
		switch e.Kind {
		case model.KindSystemPrompt:
			sys++
		case model.KindToolResult:
			tool++
		case model.KindGitArtifact:
			git++
		}
	}
	if sys != 2 {
		t.Errorf("system blocks = %d, want 2", sys)
	}
	if tool != 1 {
		t.Errorf("tool results = %d, want 1", tool)
	}
	if git != 1 {
		t.Errorf("git artifacts = %d, want 1", git)
	}
	if len(gaps) != 0 {
		t.Logf("gaps (informational): %v", gaps)
	}
}

func TestParseCodexSample(t *testing.T) {
	evs, _, err := ParseCodex(filepath.Join("..", "..", "testdata", "samples", "codex_sample.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 3 {
		t.Fatalf("events = %d, want 3 (system + user + tool output)", len(evs))
	}
	if evs[0].Kind != model.KindSystemPrompt {
		t.Errorf("first kind = %s, want system_prompt", evs[0].Kind)
	}
	if evs[2].Kind != model.KindToolResult {
		t.Errorf("third kind = %s, want tool_result", evs[2].Kind)
	}
}

func TestParseClaudeSample(t *testing.T) {
	evs, gaps, err := ParseClaude(filepath.Join("..", "..", "testdata", "samples", "claude_sample.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("events = %d, want 2 (user text + tool result)", len(evs))
	}
	if evs[0].Kind != model.KindUserMessage {
		t.Errorf("kind = %s, want user_message", evs[0].Kind)
	}
	if evs[1].Kind != model.KindToolResult {
		t.Errorf("kind = %s, want tool_result", evs[1].Kind)
	}
	if len(gaps) == 0 {
		t.Error("expected honest gap notes for claude-code (system prompt + provider not in logs)")
	}
}

func TestClassifyText(t *testing.T) {
	cases := map[string]model.ItemKind{
		"hello please help":                  model.KindUserMessage,
		"main.go:\n```go\nx\n```":            model.KindFileContent,
		"```\nSELECT 1\n```":                 model.KindCodeSnippet,
		"diff --git a/x b/x":                 model.KindGitArtifact,
	}
	for in, want := range cases {
		if got := ClassifyText(in); got != want {
			t.Errorf("ClassifyText(%q) = %s, want %s", in, got, want)
		}
	}
}

// TestClassifyRealGitDiff: a real diff carries hunk-body lines that look like
// path headers (" src/server.go:"), and the git_artifact marker must win.
func TestClassifyRealGitDiff(t *testing.T) {
	diff := "diff --git a/src/server.go b/src/server.go\n" +
		"index 1a2b3c4..5d6e7f8 100644\n" +
		"--- a/src/server.go\n" +
		"+++ b/src/server.go\n" +
		"@@ -1,4 +1,5 @@\n" +
		" package main\n" +
		" \n" +
		" src/server.go:\n" +
		"+// added\n"
	if got := ClassifyText(diff); got != model.KindGitArtifact {
		t.Errorf("ClassifyText(real diff) = %s, want %s", got, model.KindGitArtifact)
	}
}

func TestDiscoverMissingDir(t *testing.T) {
	dir := t.TempDir()
	res := Discover("zcode", filepath.Join(dir, "nope"))
	if len(res.Files) != 0 || len(res.Gaps) == 0 {
		t.Errorf("expected 0 files + gap note for missing dir, got %+v", res)
	}
}

func TestMain(m *testing.M) {
	// tests run from package dir; nothing global needed
	os.Exit(m.Run())
}
