package sales

import (
	"cmp"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // used as a content fingerprint, not for security
	"encoding/hex"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/azisuazusa/biota/pkg/util"
)

const maxSKULen = 64

func newToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return hex.EncodeToString(b), nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// buildPendingImport parses, normalizes and analyses an upload for review.
func buildPendingImport(fileName string, r io.Reader, cfg *Config) (*PendingImport, error) {
	rows, idx, err := parseCSV(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	token, err := newToken()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}

	p := &PendingImport{
		Token:          token,
		FileName:       fileName,
		CreatedAt:      time.Now(),
		TotalRows:      len(rows),
		LearnedMapping: map[string]string{},
	}

	dupIdx := map[string]int{} // key → index in groups
	var groups []DuplicateGroup
	for _, row := range rows {
		l, errs := normalizeRow(row.Line, row.Cells, idx)
		if len(errs) > 0 {
			p.ParseErrors = append(p.ParseErrors, errs...)
			continue
		}
		p.Lines = append(p.Lines, l)
		p.Warnings = append(p.Warnings, checkRowWarnings(l)...)

		sum := sha1.Sum([]byte(strings.Join(trimmedCells(row.Cells, idx), "\x1f"))) //nolint:gosec // fingerprint only
		key := hex.EncodeToString(sum[:])
		if i, ok := dupIdx[key]; ok {
			groups[i].Lines = append(groups[i].Lines, l.SourceLine)
		} else {
			dupIdx[key] = len(groups)
			groups = append(groups, DuplicateGroup{Key: key, Lines: []int{l.SourceLine}, Line: l})
		}
	}
	slices.SortFunc(p.Warnings, func(a, b RowWarning) int { return cmp.Compare(a.Line, b.Line) })
	for _, g := range groups {
		if len(g.Lines) >= 2 {
			slices.Sort(g.Lines)
			p.Duplicates = append(p.Duplicates, g)
		}
	}

	// Learned name → SKU pairs from rows that carry a SKU.
	nameSKUs := map[string]map[string]bool{}
	for _, l := range p.Lines {
		if l.SKU == "" {
			continue
		}
		if nameSKUs[l.ProductName] == nil {
			nameSKUs[l.ProductName] = map[string]bool{}
		}
		nameSKUs[l.ProductName][l.SKU] = true
	}
	for name, set := range nameSKUs {
		if len(set) == 1 {
			for sku := range set {
				p.LearnedMapping[name] = sku
			}
		}
	}

	// SKU inference for empty-SKU lines.
	unresolved := map[string]*UnresolvedSKU{}
	for i := range p.Lines {
		l := &p.Lines[i]
		if l.SKU != "" {
			continue
		}
		set := nameSKUs[l.ProductName]
		if len(set) <= 1 {
			if sku, ok := p.LearnedMapping[l.ProductName]; ok {
				l.SKU, l.SKUInferred = sku, true
				continue
			}
			if sku, ok := cfg.NameToSKU[l.ProductName]; ok && sku != "" {
				l.SKU, l.SKUInferred = sku, true
				continue
			}
		}
		u := unresolved[l.ProductName]
		if u == nil {
			u = &UnresolvedSKU{ProductName: l.ProductName}
			if len(set) > 1 {
				u.CandidateSKUs = sortedKeys(set)
			}
			unresolved[l.ProductName] = u
		}
		u.Lines = append(u.Lines, l.SourceLine)
	}
	for _, name := range sortedKeys(unresolved) {
		p.Unresolved = append(p.Unresolved, *unresolved[name])
	}

	// Unknown statuses, grouped case-insensitively.
	unknown := map[string]*UnknownStatus{}
	var order []string
	for _, l := range p.Lines {
		if _, ok := cfg.Classify(l.Status); ok {
			continue
		}
		k := statusKey(l.Status)
		if unknown[k] == nil {
			unknown[k] = &UnknownStatus{Status: l.Status}
			order = append(order, k)
		}
		unknown[k].Count++
	}
	for _, k := range order {
		p.UnknownStatus = append(p.UnknownStatus, *unknown[k])
	}
	slices.SortStableFunc(p.UnknownStatus, func(a, b UnknownStatus) int {
		if c := cmp.Compare(b.Count, a.Count); c != 0 {
			return c
		}
		return cmp.Compare(a.Status, b.Status)
	})

	// Known SKUs for the picker.
	known := map[string]bool{}
	for _, l := range p.Lines {
		if l.SKU != "" {
			known[l.SKU] = true
		}
	}
	for _, sku := range cfg.NameToSKU {
		if sku != "" {
			known[sku] = true
		}
	}
	for sku := range cfg.DisplayNames {
		known[sku] = true
	}
	p.KnownSKUs = sortedKeys(known)
	return p, nil
}

// applyDecision validates the review answers and produces the new dataset
// and config. Neither p nor cfg is modified.
func applyDecision(p *PendingImport, d ReviewDecision, cfg *Config) (*Dataset, *Config, error) {
	skuChoice := map[string]string{}
	for _, u := range p.Unresolved {
		sku := strings.TrimSpace(d.SKUChoices[u.ProductName])
		if sku == "" || utf8.RuneCountInString(sku) > maxSKULen {
			return nil, nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrUnresolvedSKU)
		}
		skuChoice[u.ProductName] = sku
	}
	statusChoice := map[string]StatusClass{}
	for _, u := range p.UnknownStatus {
		c := d.StatusChoices[u.Status]
		if c != StatusIncluded && c != StatusExcluded {
			return nil, nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrUnknownStatus)
		}
		statusChoice[u.Status] = c
	}

	drop := map[int]bool{}
	for _, g := range p.Duplicates {
		if d.KeepDuplicates[g.Key] {
			continue
		}
		for _, ln := range g.Lines[1:] {
			drop[ln] = true
		}
	}

	lines := make([]OrderLine, 0, len(p.Lines))
	for _, l := range p.Lines {
		if drop[l.SourceLine] {
			continue
		}
		if l.SKU == "" {
			l.SKU, l.SKUInferred = skuChoice[l.ProductName], true
		}
		lines = append(lines, l)
	}
	if len(lines) == 0 {
		return nil, nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrNoValidLines)
	}

	newCfg := cfg.Clone()
	for name, sku := range p.LearnedMapping {
		newCfg.NameToSKU[name] = sku
	}
	for name, sku := range skuChoice {
		newCfg.NameToSKU[name] = sku
	}
	for status, c := range statusChoice {
		newCfg.SetStatus(status, c)
	}

	slices.SortFunc(lines, func(a, b OrderLine) int {
		if c := a.OrderDate.Compare(b.OrderDate); c != 0 {
			return c
		}
		return cmp.Compare(a.SourceLine, b.SourceLine)
	})
	ds := &Dataset{
		FileName:   p.FileName,
		UploadedAt: time.Now(),
		MinDate:    DateOnly(lines[0].OrderDate),
		MaxDate:    DateOnly(lines[0].OrderDate),
		Lines:      lines,
	}
	for _, l := range lines {
		dd := DateOnly(l.OrderDate)
		if dd.Before(ds.MinDate) {
			ds.MinDate = dd
		}
		if dd.After(ds.MaxDate) {
			ds.MaxDate = dd
		}
	}
	return ds, newCfg, nil
}
