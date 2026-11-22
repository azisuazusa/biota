package views

import (
	"encoding/json"
	"net/url"
	"sort"

	"github.com/azisuazusa/biota/services/sales"
)

// StatusRow is one row on the status settings page.
type StatusRow struct {
	Label string
	Class sales.StatusClass
}

// StatusRows lists every mapped status, included first, then by label.
func StatusRows(cfg *sales.Config) []StatusRow {
	out := make([]StatusRow, 0, len(cfg.StatusMapping))
	for k, c := range cfg.StatusMapping {
		label := cfg.StatusLabels[k]
		if label == "" {
			label = k
		}
		out = append(out, StatusRow{Label: label, Class: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Class != out[j].Class {
			return out[i].Class == sales.StatusIncluded
		}
		return out[i].Label < out[j].Label
	})
	return out
}

// MappingRow is one product_name → SKU pair.
type MappingRow struct {
	ProductName string
	SKU         string
}

// MappingRows lists the config's name → SKU pairs sorted by SKU then name.
func MappingRows(cfg *sales.Config) []MappingRow {
	out := make([]MappingRow, 0, len(cfg.NameToSKU))
	for n, s := range cfg.NameToSKU {
		out = append(out, MappingRow{ProductName: n, SKU: s})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SKU != out[j].SKU {
			return out[i].SKU < out[j].SKU
		}
		return out[i].ProductName < out[j].ProductName
	})
	return out
}

func statusDeleteURL(label string) string {
	return "/settings/status/" + url.PathEscape(label)
}

func kv(k, v string) string {
	return templJSON(map[string]string{k: v})
}

func templJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
