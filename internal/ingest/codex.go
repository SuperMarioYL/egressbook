package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SuperMarioYL/egressbook/internal/model"
)

// codexPayload uses json.RawMessage because Codex rollout records vary by
// type; we decode the shapes we know and record gaps for the rest.
type codexRecord struct {
	Timestamp string          `json:"timestamp"`
	Ordinal   int             `json:"ordinal"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexSessionMeta struct {
	SessionID     string `json:"session_id"`
	Cwd           string `json:"cwd"`
	ModelProvider string `json:"model_provider"`
	BaseInstructions struct {
		Text string `json:"text"`
	} `json:"base_instructions"`
}

type codexContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexResponseItem struct {
	Type    string             `json:"type"` // "message" | "function_call" | "function_call_output" | ...
	Role    string             `json:"role"`
	Content []codexContentPart `json:"content"`
	Output  string             `json:"output"`
}

// ParseCodex reads one Codex rollout-*.jsonl session file.
func ParseCodex(path string) ([]RawEvent, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var events []RawEvent
	var gaps []string
	sessionID := ""
	provider := ""

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		row := sc.Bytes()
		if len(strings.TrimSpace(string(row))) == 0 {
			continue
		}
		var rec codexRecord
		if err := json.Unmarshal(row, &rec); err != nil {
			gaps = append(gaps, fmt.Sprintf("%s:%d unparseable line: %v", filepath.Base(path), line, err))
			continue
		}
		ts := time.Now().UTC()
		if rec.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
				ts = t
			}
		}
		switch rec.Type {
		case "session_meta":
			var meta codexSessionMeta
			if err := json.Unmarshal(rec.Payload, &meta); err != nil {
				gaps = append(gaps, fmt.Sprintf("%s:%d session_meta decode: %v", filepath.Base(path), line, err))
				continue
			}
			sessionID = meta.SessionID
			provider = meta.ModelProvider
			if meta.BaseInstructions.Text != "" {
				events = append(events, RawEvent{
					Timestamp: ts, Harness: "codex", SessionID: sessionID,
					Kind: model.KindSystemPrompt, Text: meta.BaseInstructions.Text,
					Provider: "codex:" + provider, Source: path, Line: line,
				})
			}
		case "response_item":
			var item codexResponseItem
			if err := json.Unmarshal(rec.Payload, &item); err != nil {
				gaps = append(gaps, fmt.Sprintf("%s:%d response_item decode: %v", filepath.Base(path), line, err))
				continue
			}
			switch {
			case item.Type == "message" && item.Role == "user":
				for _, part := range item.Content {
					if part.Text == "" {
						continue
					}
					events = append(events, RawEvent{
						Timestamp: ts, Harness: "codex", SessionID: sessionID,
						Kind: ClassifyText(part.Text), Text: part.Text,
						Provider: "codex:" + provider, Source: path, Line: line,
					})
				}
			case item.Type == "function_call_output":
				if item.Output != "" {
					events = append(events, RawEvent{
						Timestamp: ts, Harness: "codex", SessionID: sessionID,
						Kind: model.KindToolResult, Text: item.Output,
						Provider: "codex:" + provider, Source: path, Line: line,
					})
				}
			}
		default:
			// turn_context / event_msg / compacted etc.: no outbound payload
			// content we can attribute; counted as coverage, not egress.
		}
	}
	if err := sc.Err(); err != nil {
		gaps = append(gaps, fmt.Sprintf("%s: scan error: %v", filepath.Base(path), err))
	}
	if provider == "" {
		gaps = append(gaps, filepath.Base(path)+": model provider not recorded in session log")
	}
	return events, gaps, nil
}
