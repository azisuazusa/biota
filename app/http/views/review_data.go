package views

import (
	"slices"
	"strconv"
	"strings"

	"github.com/azisuazusa/biota/services/sales"
)

func failedRows(p *sales.PendingImport) int {
	seen := map[int]bool{}
	for _, e := range p.ParseErrors {
		seen[e.Line] = true
	}
	return len(seen)
}

func joinLines(lines []int) string {
	parts := make([]string, len(lines))
	for i, l := range lines {
		parts[i] = strconv.Itoa(l)
	}
	return strings.Join(parts, ", ")
}

// skuOptions lists candidate SKUs first, then every other known SKU.
func skuOptions(p *sales.PendingImport, u sales.UnresolvedSKU) []string {
	out := slices.Clone(u.CandidateSKUs)
	for _, s := range p.KnownSKUs {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func skuChoice(d *sales.ReviewDecision, name string) string {
	if d == nil {
		return ""
	}
	return d.SKUChoices[name]
}

func statusChoice(d *sales.ReviewDecision, status string, c sales.StatusClass) bool {
	return d != nil && d.StatusChoices[status] == c
}

func keepDup(d *sales.ReviewDecision, key string) bool {
	return d != nil && d.KeepDuplicates[key]
}

func listID(i int) string { return "known-skus-" + strconv.Itoa(i) }

type countItem struct {
	label string
	value int
	warn  bool
}

func reviewCounts(p *sales.PendingImport) []countItem {
	return []countItem{
		{"Total baris", p.TotalRows, false},
		{"Baris valid", len(p.Lines), false},
		{"Baris gagal dibaca", failedRows(p), failedRows(p) > 0},
		{"Peringatan", len(p.Warnings), len(p.Warnings) > 0},
		{"Grup duplikat", len(p.Duplicates), len(p.Duplicates) > 0},
		{"SKU belum dikenali", len(p.Unresolved), len(p.Unresolved) > 0},
		{"Status baru", len(p.UnknownStatus), len(p.UnknownStatus) > 0},
	}
}
