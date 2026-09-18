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

// claudeRecord mirrors Claude Code session JSONL rows. The user rows carry
// what was sent back into the model context (user turns and tool results);
// the assistant rows are model output arriving, which we do not count.
type claudeRecord struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Timestamp string `json:"timestamp"`
	UUID      string `json:"uuid"`
	Message   struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type claudeContentBlock struct {
	Type    string          `json:"type"`
	Text    string          `json:"text"`
	Content json.RawMessage `json:"content"`
}

// ParseClaude reads one Claude Code session .jsonl file.
func ParseClaude(path string) ([]RawEvent, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var events []RawEvent
	var gaps []string
	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		row := sc.Bytes()
		if len(strings.TrimSpace(string(row))) == 0 {
			continue
		}
		var rec claudeRecord
		if err := json.Unmarshal(row, &rec); err != nil {
			gaps = append(gaps, fmt.Sprintf("%s:%d unparseable line: %v", filepath.Base(path), line, err))
			continue
		}
		if rec.Type != "user" || rec.Message.Role != "user" {
			continue // assistant output arrives; UI rows carry no payload
		}
		ts := time.Now().UTC()
		if rec.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
				ts = t
			}
		}
		var textBlocks []string
		var toolResults []string
		if len(rec.Message.Content) > 0 && rec.Message.Content[0] == '"' {
			var s string
			if err := json.Unmarshal(rec.Message.Content, &s); err == nil && s != "" {
				textBlocks = append(textBlocks, s)
			}
		} else {
			var blocks []claudeContentBlock
			if err := json.Unmarshal(rec.Message.Content, &blocks); err == nil {
				for _, b := range blocks {
					switch b.Type {
					case "text":
						if b.Text != "" {
							textBlocks = append(textBlocks, b.Text)
						}
					case "tool_result":
						if len(b.Content) > 0 && b.Content[0] == '"' {
							var s string
							if json.Unmarshal(b.Content, &s) == nil && s != "" {
								toolResults = append(toolResults, s)
							}
						} else if len(b.Content) > 0 {
							var inner []claudeContentBlock
							if json.Unmarshal(b.Content, &inner) == nil {
								for _, ib := range inner {
									if ib.Text != "" {
										toolResults = append(toolResults, ib.Text)
									}
								}
							}
						}
					}
				}
			}
		}
		for _, t := range textBlocks {
			events = append(events, RawEvent{
				Timestamp: ts, Harness: "claude-code", SessionID: sessionID,
				Kind: ClassifyText(t), Text: t,
				Provider: "claude-code", Source: path, Line: line,
			})
		}
		for _, t := range toolResults {
			events = append(events, RawEvent{
				Timestamp: ts, Harness: "claude-code", SessionID: sessionID,
				Kind: model.KindToolResult, Text: t,
				Provider: "claude-code", Source: path, Line: line,
			})
		}
	}
	if err := sc.Err(); err != nil {
		gaps = append(gaps, fmt.Sprintf("%s: scan error: %v", filepath.Base(path), err))
	}
	gaps = append(gaps,
		"claude-code: system prompt is harness-side and not written to session logs (bytes not provable from disk)",
		"claude-code: model provider is not recorded per-session in the log; destination shown as harness name")
	return events, gaps, nil
}
