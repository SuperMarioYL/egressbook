// Package report renders the disclosure ledger: terminal table, HTML
// export, and cross-ledger diff.
package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/SuperMarioYL/egressbook/internal/model"
)

// Terminal renders the destination-mapped inventory to w.
func Terminal(w io.Writer, l *model.Ledger) {
	fmt.Fprintf(w, "egressbook — session egress disclosure ledger\n")
	fmt.Fprintf(w, "generated: %s    source: %s    items: %d\n\n", l.GeneratedAt.Format("2006-01-02 15:04 MST"), l.Source, len(l.Items))

	fmt.Fprintf(w, "COVERAGE (what the local logs could and could not prove)\n")
	for _, c := range l.Harnesses {
		fmt.Fprintf(w, "  %-12s sessions found=%d parsed=%d\n", c.Harness, c.SessionsFound, c.SessionsParsed)
		for _, g := range c.Gaps {
			fmt.Fprintf(w, "      gap: %s\n", g)
		}
	}

	byProvider := l.TotalsByProvider()
	fmt.Fprintf(w, "\nDESTINATION MAP (outbound bytes by provider)\n")
	provKeys := sortedKeysLen(byProvider)
	for _, p := range provKeys {
		fmt.Fprintf(w, "  %-46s %12d bytes\n", p, byProvider[p])
	}

	byKind := l.TotalsByKind()
	fmt.Fprintf(w, "\nPAYLOAD KINDS (outbound bytes)\n")
	for _, k := range model.AllItemKinds {
		if n, ok := byKind[k]; ok {
			fmt.Fprintf(w, "  %-16s %12d bytes\n", string(k), n)
		}
	}

	fmt.Fprintf(w, "\nITEMS (most recent first)\n")
	fmt.Fprintf(w, "  %-19s %-11s %-14s %-14s %10s  %s\n", "timestamp", "harness", "kind", "destination", "bytes", "evidence")
	items := append([]model.EgressItem{}, l.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].Timestamp.After(items[j].Timestamp) })
	limit := len(items)
	if limit > 40 {
		limit = 40
		fmt.Fprintf(w, "  (showing latest 40 of %d — full list in --html report)\n", len(items))
	}
	for i := 0; i < limit; i++ {
		it := items[i]
		fmt.Fprintf(w, "  %-19s %-11s %-14s %-14s %10d  %s:%d\n",
			it.Timestamp.Format("2006-01-02 15:04"), it.Harness, string(it.Kind), trunc(it.DestProvider, 14), it.Bytes,
			trunc(baseName(it.Evidence.File), 30), it.Evidence.Line)
	}
	if l.Signature != nil {
		fmt.Fprintf(w, "\nsignature: %s content_sha256=%s\n", l.Signature.Algorithm, l.Signature.ContentSHA)
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func baseName(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

func sortedKeysLen(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] > m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	return keys
}
