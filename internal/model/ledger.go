// Package model defines the session-egress-disclosure ledger: a signed,
// canonical, diffable artifact that maps every reconstructed outbound item
// from a coding-agent session log to its destination.
package model

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ItemKind classifies what an egress item carried.
type ItemKind string

const (
	KindSystemPrompt ItemKind = "system_prompt"
	KindUserMessage  ItemKind = "user_message"
	KindFileContent  ItemKind = "file_content"
	KindCodeSnippet  ItemKind = "code_snippet"
	KindToolResult   ItemKind = "tool_result"
	KindGitArtifact  ItemKind = "git_artifact"
)

// AllItemKinds is the ordered enumeration used by reports and diffs.
var AllItemKinds = []ItemKind{KindSystemPrompt, KindUserMessage, KindFileContent, KindCodeSnippet, KindToolResult, KindGitArtifact}

// Locator points back into the raw session log so a reviewer can verify
// every ledger item against its source bytes.
type Locator struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Offset int64  `json:"offset"`
	Length int    `json:"length"`
}

// EgressItem is one reconstructed outbound payload unit.
type EgressItem struct {
	Timestamp    time.Time `json:"timestamp"`
	Harness      string    `json:"harness"`
	SessionID    string    `json:"session_id"`
	Kind         ItemKind  `json:"kind"`
	DestProvider string    `json:"dest_provider"`
	DestModel    string    `json:"dest_model,omitempty"`
	Bytes        int64     `json:"bytes"`
	Evidence     Locator   `json:"evidence"`
}

// Coverage records, per harness, what the local logs could and could not
// prove. Honesty is the product: "everything logged" is labeled, not implied.
type Coverage struct {
	Harness       string   `json:"harness"`
	SessionsFound int      `json:"sessions_found"`
	SessionsParsed int     `json:"sessions_parsed"`
	Gaps          []string `json:"gaps,omitempty"`
}

// Ledger is the disclosure artifact: itemized egress + coverage + signature.
type Ledger struct {
	GeneratedAt time.Time  `json:"generated_at"`
	Source      string     `json:"source"`
	Harnesses   []Coverage `json:"harnesses"`
	Items       []EgressItem `json:"items"`
	Signature   *Signature `json:"signature,omitempty"`
}

// Signature carries the ed25519 proof over the canonical serialization.
type Signature struct {
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	Value     string `json:"value"`
	ContentSHA string `json:"content_sha256"`
}

// NewLedger assembles an unsigned ledger.
func NewLedger(source string, cov []Coverage, items []EgressItem) *Ledger {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Timestamp.Equal(items[j].Timestamp) {
			return items[i].Evidence.File < items[j].Evidence.File
		}
		return items[i].Timestamp.Before(items[j].Timestamp)
	})
	return &Ledger{GeneratedAt: time.Now().UTC(), Source: source, Harnesses: cov, Items: items}
}

// canonical returns the deterministic serialization the signature covers:
// identical content must yield identical bytes regardless of map order.
func (l *Ledger) canonical() ([]byte, error) {
	clone := *l
	clone.Signature = nil
	b, err := json.MarshalIndent(&clone, "", "  ")
	if err != nil {
		return nil, err
	}
	return b, nil
}

// SignKey is a persistent ed25519 keypair used to sign ledgers.
type SignKey struct {
	Priv ed25519.PrivateKey
	Pub  ed25519.PublicKey
}

// LoadOrCreateKey reads the machine key at dir/egressbook.ed25519 or creates it.
func LoadOrCreateKey(dir string) (*SignKey, error) {
	path := filepath.Join(dir, "egressbook.ed25519")
	if b, err := os.ReadFile(path); err == nil {
		if len(b) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("key file %s has wrong size", path)
		}
		priv := ed25519.PrivateKey(b)
		return &SignKey{Priv: priv, Pub: priv.Public().(ed25519.PublicKey)}, nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, priv, 0o600); err != nil {
		return nil, err
	}
	return &SignKey{Priv: priv, Pub: pub}, nil
}

// Sign attaches an ed25519 signature over the canonical serialization.
func (l *Ledger) Sign(k *SignKey) error {
	canon, err := l.canonical()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(canon)
	sig := ed25519.Sign(k.Priv, canon)
	l.Signature = &Signature{
		Algorithm:  "ed25519",
		PublicKey:  base64.StdEncoding.EncodeToString(k.Pub),
		Value:      base64.StdEncoding.EncodeToString(sig),
		ContentSHA: fmt.Sprintf("%x", sum),
	}
	return nil
}

// Verify checks the embedded signature against the item content.
func (l *Ledger) Verify() error {
	if l.Signature == nil {
		return errors.New("ledger is unsigned")
	}
	canon, err := l.canonical()
	if err != nil {
		return err
	}
	pub, err := base64.StdEncoding.DecodeString(l.Signature.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("bad embedded public key")
	}
	sig, err := base64.StdEncoding.DecodeString(l.Signature.Value)
	if err != nil {
		return errors.New("bad signature encoding")
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), canon, sig) {
		return errors.New("signature mismatch: ledger content was modified after signing")
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(canon))
	if sum != l.Signature.ContentSHA {
		return errors.New("content hash mismatch")
	}
	return nil
}

// TotalsByProvider aggregates outbound bytes per destination.
func (l *Ledger) TotalsByProvider() map[string]int64 {
	m := map[string]int64{}
	for _, it := range l.Items {
		m[it.DestProvider] += it.Bytes
	}
	return m
}

// TotalsByKind aggregates outbound bytes per item kind.
func (l *Ledger) TotalsByKind() map[ItemKind]int64 {
	m := map[ItemKind]int64{}
	for _, it := range l.Items {
		m[it.Kind] += it.Bytes
	}
	return m
}
