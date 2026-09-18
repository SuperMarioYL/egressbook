// Package ingest parses local coding-agent session logs into normalized
// outbound events. It reads only files already on disk; it never opens
// network connections.
package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/SuperMarioYL/egressbook/internal/model"
)

// RawEvent is one outbound payload unit found in a log, pre-aggregation.
type RawEvent struct {
	Timestamp time.Time
	Harness   string
	SessionID string
	Kind      model.ItemKind
	Text      string // the payload content that left (or left-ward direction)
	Provider  string // raw provider id, "" when the log does not record it
	Model     string
	Source    string // absolute path of the log line
	Line      int
}

// DiscoverResult reports what exists on this machine.
type DiscoverResult struct {
	Dir        string
	Files      []string
	Harness    string
	Gaps       []string
}

// DefaultRoots lists the default session-log roots per harness.
func DefaultRoots(home string) []DiscoverResult {
	return []DiscoverResult{
		{Harness: "zcode", Dir: filepath.Join(home, ".zcode", "cli", "rollout"), Gaps: nil},
		{Harness: "codex", Dir: filepath.Join(home, ".codex", "sessions")},
		{Harness: "claude-code", Dir: filepath.Join(home, ".claude", "projects")},
	}
}

// Discover finds session-log files for one harness root.
func Discover(harness, dir string) DiscoverResult {
	res := DiscoverResult{Harness: harness, Dir: dir}
	var ext string
	switch harness {
	case "zcode":
		ext = ".jsonl"
	case "codex":
		ext = ".jsonl"
	case "claude-code":
		ext = ".jsonl"
	default:
		res.Gaps = append(res.Gaps, fmt.Sprintf("unknown harness %q", harness))
		return res
	}
	st, err := os.Stat(dir)
	if err != nil {
		res.Gaps = append(res.Gaps, "not installed / no log dir")
		return res
	}
	if !st.IsDir() {
		res.Gaps = append(res.Gaps, "log root is not a directory")
		return res
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ext) && !strings.HasSuffix(path, ".probe.yaml") {
			res.Files = append(res.Files, path)
		}
		return nil
	})
	return res
}

var (
	fenceRe   = regexp.MustCompile("(?s)```[a-zA-Z0-9+-]*\\n.*?```")
	pathHeadRe = regexp.MustCompile("^[A-Za-z0-9_./~-]+\\.[A-Za-z0-9]+:?[ \t]*$")
)

// ClassifyText maps a payload block to an item kind using honest, conservative
// heuristics: a block is file_content when it carries a path header or a
// path-prefenced fenced block, code_snippet for bare fences, else user text.
func ClassifyText(text string) model.ItemKind {
	lines := strings.Split(text, "\n")
	for _, ln := range lines {
		trimmed := strings.TrimRight(ln, "\r")
		if trimmed != "" && pathHeadRe.MatchString(strings.TrimSpace(trimmed)) {
			return model.KindFileContent
		}
	}
	if strings.Contains(text, "diff --git") || strings.HasPrefix(strings.TrimSpace(text), "@@ ") {
		return model.KindGitArtifact
	}
	if fenceRe.MatchString(text) {
		return model.KindCodeSnippet
	}
	return model.KindUserMessage
}
