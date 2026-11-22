// Package sales holds the chi HTTP handlers for the sales domain.
package sales

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/azisuazusa/biota/services/sales"
)

const msgInvalidFilter = "Rentang tanggal tidak valid"

// DashboardQuery is the raw dashboard filter query string.
type DashboardQuery struct {
	From     string
	To       string
	Channels []string
}

// parseDashboardQuery turns query params into a filter. Empty dates default to
// the dataset's range; without filtered=1 every channel is selected. A
// non-empty errMsg means the range is invalid; the filter still carries
// whatever could be parsed so the form can show it.
func parseDashboardQuery(r *http.Request, ds *sales.Dataset) (sales.Filter, string) {
	q := r.URL.Query()
	dq := DashboardQuery{
		From:     strings.TrimSpace(q.Get("from")),
		To:       strings.TrimSpace(q.Get("to")),
		Channels: q["channel"],
	}

	f := sales.Filter{From: ds.MinDate, To: ds.MaxDate}
	if q.Get("filtered") == "" && len(dq.Channels) == 0 {
		f.Channels = slices.Clone(sales.AllChannels)
	} else {
		for _, c := range sales.AllChannels {
			if slices.Contains(dq.Channels, string(c)) {
				f.Channels = append(f.Channels, c)
			}
		}
	}

	errMsg := ""
	parse := func(raw string, dst *time.Time) {
		if raw == "" {
			return
		}
		t, err := time.ParseInLocation("2006-01-02", raw, time.UTC)
		if err != nil {
			errMsg = msgInvalidFilter
			return
		}
		*dst = t
	}
	parse(dq.From, &f.From)
	parse(dq.To, &f.To)
	if errMsg == "" && (f.From.After(f.To) || f.From.Before(sales.DateOnly(ds.MinDate)) || f.To.After(sales.DateOnly(ds.MaxDate))) {
		errMsg = msgInvalidFilter
	}
	return f, errMsg
}

// bracketKey returns the inner part of "prefix[inner]" form keys.
func bracketKey(key, prefix string) (string, bool) {
	if !strings.HasPrefix(key, prefix+"[") || !strings.HasSuffix(key, "]") {
		return "", false
	}
	return key[len(prefix)+1 : len(key)-1], true
}

// parseReviewDecision reads token, keep_dup[<key>], sku[<name>] and
// status[<status>] from the posted form. r.ParseForm must have been called.
func parseReviewDecision(r *http.Request) sales.ReviewDecision {
	d := sales.ReviewDecision{
		Token:          strings.TrimSpace(r.PostForm.Get("token")),
		KeepDuplicates: map[string]bool{},
		SKUChoices:     map[string]string{},
		StatusChoices:  map[string]sales.StatusClass{},
	}
	for key, vals := range r.PostForm {
		if len(vals) == 0 {
			continue
		}
		v := vals[0]
		if k, ok := bracketKey(key, "keep_dup"); ok {
			d.KeepDuplicates[k] = v == "on" || v == "true" || v == "1"
		} else if k, ok := bracketKey(key, "sku"); ok {
			d.SKUChoices[k] = strings.TrimSpace(v)
		} else if k, ok := bracketKey(key, "status"); ok {
			d.StatusChoices[k] = sales.StatusClass(v)
		}
	}
	return d
}
