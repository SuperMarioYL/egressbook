package report

import (
	"fmt"
	"io"

	"github.com/SuperMarioYL/egressbook/internal/model"
)

// Diff is the cross-ledger comparison: what changed in what leaves.
type Diff struct {
	NewDestinations []string
	NewKinds        []string
	ProviderDeltas  []string
	ItemCountFrom   int
	ItemCountTo     int
}

// ComputeDiff compares an older ledger (from) against a newer one (to).
func ComputeDiff(from, to *model.Ledger) Diff {
	d := Diff{ItemCountFrom: len(from.Items), ItemCountTo: len(to.Items)}

	fromProv := from.TotalsByProvider()
	toProv := to.TotalsByProvider()
	for p, v := range toProv {
		old, ok := fromProv[p]
		if !ok {
			d.NewDestinations = append(d.NewDestinations, fmt.Sprintf("%s (+%d bytes, new destination)", p, v))
		} else if v != old {
			d.ProviderDeltas = append(d.ProviderDeltas, fmt.Sprintf("%s %d → %d bytes (%+d)", p, old, v, v-old))
		}
	}
	for p, old := range fromProv {
		if _, ok := toProv[p]; !ok {
			d.ProviderDeltas = append(d.ProviderDeltas, fmt.Sprintf("%s %d → 0 bytes (destination gone)", p, old))
		}
	}
	fromKind := from.TotalsByKind()
	toKind := to.TotalsByKind()
	for k := range toKind {
		if _, ok := fromKind[k]; !ok {
			d.NewKinds = append(d.NewKinds, string(k))
		}
	}
	return d
}

// PrintDiff renders the diff to w.
func PrintDiff(w io.Writer, d Diff) {
	fmt.Fprintf(w, "egressbook diff — what changed in what leaves\n")
	fmt.Fprintf(w, "items: %d → %d\n", d.ItemCountFrom, d.ItemCountTo)
	if len(d.NewDestinations) > 0 {
		fmt.Fprintf(w, "\nNEW DESTINATIONS\n")
		for _, s := range d.NewDestinations {
			fmt.Fprintf(w, "  + %s\n", s)
		}
	}
	if len(d.NewKinds) > 0 {
		fmt.Fprintf(w, "\nNEW PAYLOAD KINDS\n")
		for _, s := range d.NewKinds {
			fmt.Fprintf(w, "  + %s\n", s)
		}
	}
	if len(d.ProviderDeltas) > 0 {
		fmt.Fprintf(w, "\nDESTINATION DELTAS\n")
		for _, s := range d.ProviderDeltas {
			fmt.Fprintf(w, "  ~ %s\n", s)
		}
	}
	if len(d.NewDestinations) == 0 && len(d.NewKinds) == 0 && len(d.ProviderDeltas) == 0 {
		fmt.Fprintf(w, "\nno changes in destinations or payload kinds\n")
	}
}
