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

// zcodeContentBlock mirrors the content-block shape inside a ZCode model-io
// request body: {"type":"text","text":"...","cache_control":{...}}.
type zcodeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type zcodeMessage struct {
	Role    string             `json:"role"`
	Content []zcodeContentBlock `json:"content"`
}

type zcodeRequestBody struct {
	Model    string              `json:"model"`
	System   []zcodeContentBlock `json:"system"`
	Messages []zcodeMessage      `json:"messages"`
}

type zcodeRecord struct {
	CompletedAt string `json:"completedAt"`
	RequestID   string `json:"requestId"`
	Model       struct {
		ModelID    string `json:"modelId"`
		ProviderID string `json:"providerId"`
	} `json:"model"`
	Request struct {
		Body zcodeRequestBody `json:"body"`
	} `json:"request"`
}

// ParseZcode reads one model-io-sess_*.jsonl file. ZCode records the full
// outbound request body locally — including every system block and message —
// which is what makes retrospective disclosure possible for this harness.
func ParseZcode(path string) ([]RawEvent, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var events []RawEvent
	var gaps []string
	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	sessionID = strings.TrimPrefix(sessionID, "model-io-")

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		row := sc.Bytes()
		if len(strings.TrimSpace(string(row))) == 0 {
			continue
		}
		var rec zcodeRecord
		if err := json.Unmarshal(row, &rec); err != nil {
			gaps = append(gaps, fmt.Sprintf("%s:%d unparseable line: %v", filepath.Base(path), line, err))
			continue
		}
		ts := time.Now().UTC()
		if rec.CompletedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, rec.CompletedAt); err == nil {
				ts = t
			}
		}
		dest := rec.Model.ProviderID
		if dest == "" {
			dest = "zcode:" + rec.Model.ModelID
		}
		for _, sys := range rec.Request.Body.System {
			if sys.Text == "" {
				continue
			}
			events = append(events, RawEvent{
				Timestamp: ts, Harness: "zcode", SessionID: sessionID,
				Kind: model.KindSystemPrompt, Text: sys.Text,
				Provider: dest, Model: rec.Model.ModelID,
				Source: path, Line: line,
			})
		}
		for _, msg := range rec.Request.Body.Messages {
			for _, blk := range msg.Content {
				if blk.Text == "" {
					continue
				}
				kind := ClassifyText(blk.Text)
				if msg.Role != "user" && msg.Role != "tool" {
					// assistant blocks are model output arriving, not leaving
					continue
				}
				if msg.Role == "tool" {
					kind = model.KindToolResult
				}
				events = append(events, RawEvent{
					Timestamp: ts, Harness: "zcode", SessionID: sessionID,
					Kind: kind, Text: blk.Text,
					Provider: dest, Model: rec.Model.ModelID,
					Source: path, Line: line,
				})
			}
		}
	}
	if err := sc.Err(); err != nil {
		gaps = append(gaps, fmt.Sprintf("%s: scan error: %v", filepath.Base(path), err))
	}
	return events, gaps, nil
}
