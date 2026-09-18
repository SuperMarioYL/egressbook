// Package analyze aggregates raw ingest events into the disclosure ledger
// and normalizes destination names.
package analyze

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SuperMarioYL/egressbook/internal/ingest"
	"github.com/SuperMarioYL/egressbook/internal/model"
)

// ProviderDisplay maps raw provider ids to honest display names. Anything
// unknown is surfaced verbatim rather than guessed.
func ProviderDisplay(raw string) string {
	if raw == "" {
		return "unknown (not recorded in log)"
	}
	l := strings.ToLower(raw)
	switch {
	case strings.Contains(l, "zhipu") || strings.HasPrefix(l, "glm"):
		return "Zhipu (GLM)"
	case strings.Contains(l, "openai") || strings.HasPrefix(l, "gpt"):
		return "OpenAI"
	case strings.Contains(l, "anthropic") || strings.HasPrefix(l, "claude"):
		return "Anthropic (Claude)"
	case strings.Contains(l, "deepseek"):
		return "DeepSeek"
	case strings.Contains(l, "qwen"):
		return "Qwen (Alibaba)"
	case strings.Contains(l, "kimi") || strings.Contains(l, "moonshot"):
		return "Moonshot (Kimi)"
	case strings.HasPrefix(l, "zcode:"):
		return "ZCode → " + ProviderDisplay(strings.TrimPrefix(l, "zcode:"))
	case strings.HasPrefix(l, "codex:"):
		return "Codex → " + ProviderDisplay(strings.TrimPrefix(l, "codex:"))
	case strings.HasPrefix(l, "claude-code"):
		return "Claude Code (provider not recorded per-session)"
	default:
		return raw
	}
}

// BuildLedger turns parsed events + coverage into the signed-able ledger.
func BuildLedger(source string, events []ingest.RawEvent, coverage []model.Coverage, gapsByHarness map[string][]string) *model.Ledger {
	items := make([]model.EgressItem, 0, len(events))
	for _, ev := range events {
		display := ProviderDisplay(ev.Provider)
		if display == ev.Provider && isOpaqueID(ev.Provider) && ev.Model != "" {
			// provider recorded as an opaque id (e.g. a UUID); the model id
			// carries the vendor — prefer it for the destination display.
			display = ProviderDisplay(ev.Model)
		}
		items = append(items, model.EgressItem{
			Timestamp:    ev.Timestamp,
			Harness:      ev.Harness,
			SessionID:    ev.SessionID,
			Kind:         ev.Kind,
			DestProvider: display,
			DestModel:    ev.Model,
			Bytes:        int64(len(ev.Text)),
			Evidence:     model.Locator{File: ev.Source, Line: ev.Line},
		})
	}
	cov := make([]model.Coverage, len(coverage))
	copy(cov, coverage)
	for i := range cov {
		if g, ok := gapsByHarness[cov[i].Harness]; ok {
			cov[i].Gaps = append(cov[i].Gaps, g...)
		}
	}
	return model.NewLedger(source, cov, items)
}

// SummaryLine renders one harness coverage row for terminal output.
func SummaryLine(c model.Coverage) string {
	gaps := ""
	if len(c.Gaps) > 0 {
		gaps = fmt.Sprintf("  (gaps: %d)", len(c.Gaps))
	}
	return fmt.Sprintf("%-12s sessions: found=%d parsed=%d%s", c.Harness, c.SessionsFound, c.SessionsParsed, gaps)
}

// SortCoverage orders harnesses deterministically.
func SortCoverage(cov []model.Coverage) {
	sort.Slice(cov, func(i, j int) bool { return cov[i].Harness < cov[j].Harness })
}

// isOpaqueID reports whether s looks like an opaque machine id (UUID / hex
// blob) rather than a human-meaningful provider name.
func isOpaqueID(s string) bool {
	hasDash := false
	hexOrDash := true
	for _, r := range s {
		switch {
		case r == '-':
			hasDash = true
		case (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f'):
		default:
			hexOrDash = false
		}
	}
	return hexOrDash && hasDash && len(s) >= 32
}
