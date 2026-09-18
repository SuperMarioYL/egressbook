package model

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestSignVerifyRoundtrip(t *testing.T) {
	dir := t.TempDir()
	key, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	l := NewLedger("host", []Coverage{{Harness: "zcode", SessionsFound: 1, SessionsParsed: 1}},
		[]EgressItem{{Timestamp: mustTime(t, "2026-09-18T00:00:28Z"), Harness: "zcode", SessionID: "s1", Kind: KindSystemPrompt, DestProvider: "Zhipu (GLM)", Bytes: 42}})
	if err := l.Sign(key); err != nil {
		t.Fatal(err)
	}
	if err := l.Verify(); err != nil {
		t.Fatalf("fresh signature should verify: %v", err)
	}
	// Persist → load → verify (canonical JSON path).
	b, _ := json.Marshal(l)
	var back Ledger
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if err := back.Verify(); err != nil {
		t.Fatalf("roundtrip signature should verify: %v", err)
	}
	// Tamper → verify must fail.
	back.Items[0].Bytes = 999
	if err := back.Verify(); err == nil {
		t.Fatal("tampered ledger must fail verification")
	}
}

func TestLoadOrCreateKeyStable(t *testing.T) {
	dir := t.TempDir()
	k1, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := LoadOrCreateKey(filepath.Join(dir)) // same dir
	if err != nil {
		t.Fatal(err)
	}
	if string(k1.Pub) != string(k2.Pub) {
		t.Fatal("key must be persistent across loads")
	}
}

func TestTotals(t *testing.T) {
	l := NewLedger("h", nil, []EgressItem{
		{Kind: KindUserMessage, DestProvider: "A", Bytes: 10},
		{Kind: KindUserMessage, DestProvider: "A", Bytes: 5},
		{Kind: KindFileContent, DestProvider: "B", Bytes: 100},
	})
	if l.TotalsByProvider()["A"] != 15 || l.TotalsByProvider()["B"] != 100 {
		t.Errorf("provider totals wrong: %v", l.TotalsByProvider())
	}
	if l.TotalsByKind()[KindUserMessage] != 15 {
		t.Errorf("kind totals wrong: %v", l.TotalsByKind())
	}
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}
