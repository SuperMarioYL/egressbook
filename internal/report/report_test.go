package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/SuperMarioYL/egressbook/internal/model"
)

func TestShortFileKeepsFileName(t *testing.T) {
	cases := map[string]string{
		"/tmp/demo-home/.zcode/cli/rollout/model-io-sess-test.jsonl":   "…/rollout/model-io-sess-test.jsonl",
		"/tmp/demo-home/.codex/sessions/2026/09/18/rollout-test.jsonl": "…/18/rollout-test.jsonl",
		"/tmp/demo-home/.claude/projects/-proj/abc.jsonl":              "…/-proj/abc.jsonl",
		"abc.jsonl":     "abc.jsonl",
		"dir/abc.jsonl": "dir/abc.jsonl",
	}
	for in, want := range cases {
		if got := shortFile(in); got != want {
			t.Errorf("shortFile(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestHTMLReportEvidenceNamesTheFile checks the rendered artifact, not just the
// helper: the shareable report's evidence column must identify the log file.
func TestHTMLReportEvidenceNamesTheFile(t *testing.T) {
	led := &model.Ledger{
		GeneratedAt: time.Now().UTC(),
		Source:      "host",
		Harnesses:   []model.Coverage{{Harness: "claude-code", SessionsFound: 1, SessionsParsed: 1}},
		Items: []model.EgressItem{{
			Timestamp:    time.Now().UTC(),
			Harness:      "claude-code",
			SessionID:    "abc",
			Kind:         model.KindUserMessage,
			DestProvider: "Anthropic (Claude)",
			Bytes:        12,
			Evidence:     model.Locator{File: "/home/u/.claude/projects/-proj/abc.jsonl", Line: 7},
		}},
	}
	var buf bytes.Buffer
	if err := HTML(&buf, led); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "abc.jsonl:7") {
		t.Errorf("HTML evidence column does not name the log file:\n%s", buf.String())
	}
}
