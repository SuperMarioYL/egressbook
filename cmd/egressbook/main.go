// egressbook reconstructs a signed, destination-mapped disclosure ledger of
// what your coding agents already transmitted, from the session logs already
// on your disk. It performs zero network calls.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/SuperMarioYL/egressbook/internal/analyze"
	"github.com/SuperMarioYL/egressbook/internal/ingest"
	"github.com/SuperMarioYL/egressbook/internal/model"
	"github.com/SuperMarioYL/egressbook/internal/report"
)

var version = "0.2.0"

func main() {
	root := &cobra.Command{
		Use:     "egressbook",
		Short:   "Retrospective egress disclosure ledger for coding agent sessions",
		Version: version,
	}
	var home string
	var dataDir string
	root.PersistentFlags().StringVar(&home, "home", os.Getenv("HOME"), "home directory to scan for agent logs")
	root.PersistentFlags().StringVar(&dataDir, "data-dir", "", "directory holding saved ledgers + signing key (default ~/.egressbook)")

	dataDirOf := func() string {
		if dataDir != "" {
			return dataDir
		}
		return filepath.Join(home, ".egressbook")
	}

	scan := &cobra.Command{
		Use:   "scan",
		Short: "detect harness session logs on this machine and report coverage",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cov, _, gaps := collect(home)
			analyze.SortCoverage(cov)
			for _, c := range cov {
				fmt.Println(analyze.SummaryLine(c))
				for _, g := range gaps[c.Harness] {
					fmt.Printf("      gap: %s\n", g)
				}
			}
			return nil
		},
	}

	var ledgerOut string
	ledger := &cobra.Command{
		Use:   "ledger",
		Short: "build the signed disclosure ledger and print the destination map",
		RunE: func(cmd *cobra.Command, args []string) error {
			led := buildSigned(home, dataDirOf(), nil)
			report.Terminal(os.Stdout, led)
			if ledgerOut != "" {
				if err := saveLedger(led, ledgerOut); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "ledger saved: %s\n", ledgerOut)
			}
			return nil
		},
	}
	ledger.Flags().StringVar(&ledgerOut, "out", "", "save the signed ledger to this path (.json or .yaml)")

	var htmlOut string
	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "render the self-contained HTML disclosure report",
		RunE: func(cmd *cobra.Command, args []string) error {
			led := buildSigned(home, dataDirOf(), nil)
			f, err := os.Create(htmlOut)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := report.HTML(f, led); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "report written: %s (%d items)\n", htmlOut, len(led.Items))
			return nil
		},
	}
	reportCmd.Flags().StringVar(&htmlOut, "html", "egressbook-report.html", "output path for the HTML report")

	var since string
	var oldLedger, newLedger string
	diff := &cobra.Command{
		Use:   "diff",
		Short: "what changed in what leaves (two saved ledgers, or --since split)",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case oldLedger != "" && newLedger != "":
				from, err := loadLedger(oldLedger)
				if err != nil {
					return err
				}
				to, err := loadLedger(newLedger)
				if err != nil {
					return err
				}
				report.PrintDiff(os.Stdout, report.ComputeDiff(from, to))
				return nil
			case since != "":
				dur, err := parseSince(since)
				if err != nil {
					return err
				}
				cut := time.Now().Add(-dur)
				recent := buildSigned(home, dataDirOf(), &cut)
				older := buildBefore(home, cut)
				report.PrintDiff(os.Stdout, report.ComputeDiff(older, recent))
				return nil
			default:
				return fmt.Errorf("pass --old and --new, or --since (e.g. --since 2w)")
			}
		},
	}
	diff.Flags().StringVar(&since, "since", "", "split this machine's events at now-<dur> (e.g. 2w, 72h)")
	diff.Flags().StringVar(&oldLedger, "old", "", "older saved ledger file")
	diff.Flags().StringVar(&newLedger, "new", "", "newer saved ledger file")

	verify := &cobra.Command{
		Use:   "verify",
		Short: "verify a saved ledger's signature",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			led, err := loadLedger(args[0])
			if err != nil {
				return err
			}
			if err := led.Verify(); err != nil {
				return fmt.Errorf("VERIFY FAILED: %w", err)
			}
			fmt.Printf("signature OK (%s, content_sha256=%s, %d items)\n", led.Signature.Algorithm, led.Signature.ContentSHA, len(led.Items))
			return nil
		},
	}

	root.AddCommand(scan, ledger, reportCmd, diff, verify)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// collect discovers and parses every harness log under home.
func collect(home string) ([]ingest.RawEvent, []model.Coverage, map[string]int, map[string][]string) {
	var events []ingest.RawEvent
	cov := []model.Coverage{}
	gaps := map[string][]string{}
	sessions := map[string]int{}
	for _, root := range ingest.DefaultRoots(home) {
		d := ingest.Discover(root.Harness, root.Dir)
		c := model.Coverage{Harness: root.Harness, SessionsFound: len(d.Files)}
		sessions[root.Harness] = len(d.Files)
		// Discovery gaps (absent log root, root is not a directory) are as
		// load-bearing as parse gaps: without them a harness that is not
		// installed is indistinguishable from one that logged nothing.
		gaps[root.Harness] = append(gaps[root.Harness], d.Gaps...)
		for _, f := range d.Files {
			var evs []ingest.RawEvent
			var gg []string
			var err error
			switch root.Harness {
			case "zcode":
				evs, gg, err = ingest.ParseZcode(f)
			case "codex":
				evs, gg, err = ingest.ParseCodex(f)
			case "claude-code":
				evs, gg, err = ingest.ParseClaude(f)
			}
			if err != nil {
				gg = append(gg, f+": "+err.Error())
			} else {
				c.SessionsParsed++
				events = append(events, evs...)
			}
			gaps[root.Harness] = append(gaps[root.Harness], gg...)
		}
		cov = append(cov, c)
	}
	return events, cov, sessions, gaps
}

// filterSince keeps events at or after t (nil = keep all).
func filterSince(events []ingest.RawEvent, t *time.Time) []ingest.RawEvent {
	if t == nil {
		return events
	}
	out := events[:0]
	for _, e := range events {
		if !e.Timestamp.Before(*t) {
			out = append(out, e)
		}
	}
	return out
}

// filterBefore keeps events strictly before t.
func filterBefore(events []ingest.RawEvent, t time.Time) []ingest.RawEvent {
	out := make([]ingest.RawEvent, 0, len(events))
	for _, e := range events {
		if e.Timestamp.Before(t) {
			out = append(out, e)
		}
	}
	return out
}

func hostname() string {
	h, _ := os.Hostname()
	if h == "" {
		return "unknown"
	}
	return h
}

func buildSigned(home, keyDir string, since *time.Time) *model.Ledger {
	events, cov, _, gaps := collect(home)
	events = filterSince(events, since)
	led := analyze.BuildLedger(hostname(), events, cov, gaps)
	if key, err := model.LoadOrCreateKey(keyDir); err == nil {
		_ = led.Sign(key)
	}
	return led
}

func buildBefore(home string, cut time.Time) *model.Ledger {
	events, cov, _, gaps := collect(home)
	events = filterBefore(events, cut)
	return analyze.BuildLedger(hostname(), events, cov, gaps)
}

func saveLedger(l *model.Ledger, path string) error {
	var b []byte
	var err error
	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		b, err = yaml.Marshal(l)
	default:
		b, err = json.MarshalIndent(l, "", "  ")
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func loadLedger(path string) (*model.Ledger, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var l model.Ledger
	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(b, &l)
	default:
		err = json.Unmarshal(b, &l)
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func parseSince(s string) (time.Duration, error) {
	var n int64
	var unit string
	if _, err := fmt.Sscanf(s, "%d%s", &n, &unit); err != nil || n <= 0 {
		return 0, fmt.Errorf("bad --since %q (use a positive duration, e.g. 2w or 72h)", s)
	}
	var perUnit time.Duration
	switch unit {
	case "h":
		perUnit = time.Hour
	case "d":
		perUnit = 24 * time.Hour
	case "w":
		perUnit = 7 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("unknown --since unit %q (h/d/w)", unit)
	}
	if n > int64(math.MaxInt64)/int64(perUnit) {
		return 0, fmt.Errorf("--since %q is too large", s)
	}
	return time.Duration(n) * perUnit, nil
}
