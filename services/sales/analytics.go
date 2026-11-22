package sales

import (
	"cmp"
	"math"
	"slices"
)

// summarize computes every dashboard widget for one filter. Status
// classification happens here, at query time, so mapping edits apply at once.
func summarize(ds *Dataset, cfg *Config, f Filter) *Summary {
	s := &Summary{ByProduct: []ProductRow{}, Top10: []ProductRow{}}

	allowed := map[Channel]bool{}
	for _, c := range f.Channels {
		allowed[c] = true
	}
	from, to := DateOnly(f.From), DateOnly(f.To)

	type chanAgg struct {
		orders map[string]bool
		units  int
		rev    int64
	}
	chans := map[Channel]*chanAgg{}
	for _, c := range AllChannels {
		chans[c] = &chanAgg{orders: map[string]bool{}}
	}
	type prodAgg struct {
		units int
		rev   int64
	}
	prods := map[string]*prodAgg{}
	daily := map[string]int64{}
	orders := map[string]bool{}

	for _, l := range ds.Lines {
		d := DateOnly(l.OrderDate)
		if d.Before(from) || d.After(to) || !allowed[l.Channel] {
			continue
		}
		if c, ok := cfg.Classify(l.Status); !ok || c != StatusIncluded {
			continue
		}
		s.NetRevenue += l.TotalPrice
		s.Units += l.Qty
		orders[l.OrderID] = true

		ca := chans[l.Channel]
		if ca == nil {
			ca = &chanAgg{orders: map[string]bool{}}
			chans[l.Channel] = ca
		}
		ca.orders[l.OrderID] = true
		ca.units += l.Qty
		ca.rev += l.TotalPrice

		pa := prods[l.SKU]
		if pa == nil {
			pa = &prodAgg{}
			prods[l.SKU] = pa
		}
		pa.units += l.Qty
		pa.rev += l.TotalPrice

		daily[d.Format("2006-01-02")] += l.TotalPrice
	}

	s.Orders = len(orders)
	if s.Orders > 0 {
		s.AOV = int64(math.Round(float64(s.NetRevenue) / float64(s.Orders)))
	}
	share := func(v int64) float64 {
		if s.NetRevenue == 0 {
			return 0
		}
		return float64(v) / float64(s.NetRevenue) * 100
	}

	for _, c := range AllChannels {
		if !allowed[c] {
			continue
		}
		ca := chans[c]
		s.ByChannel = append(s.ByChannel, ChannelRow{
			Channel: c, Orders: len(ca.orders), Units: ca.units, NetRevenue: ca.rev, SharePct: share(ca.rev),
		})
	}

	defaults := defaultDisplayNames(ds)
	for sku, pa := range prods {
		s.ByProduct = append(s.ByProduct, ProductRow{
			SKU: sku, DisplayName: resolveDisplayName(sku, cfg, defaults), Units: pa.units, NetRevenue: pa.rev, SharePct: share(pa.rev),
		})
	}
	slices.SortFunc(s.ByProduct, func(a, b ProductRow) int {
		if c := cmp.Compare(b.NetRevenue, a.NetRevenue); c != 0 {
			return c
		}
		return cmp.Compare(a.SKU, b.SKU)
	})
	for i := range s.ByProduct {
		s.ByProduct[i].Rank = i + 1
	}
	s.Top10 = s.ByProduct[:min(10, len(s.ByProduct))]

	if !from.After(to) {
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			s.Daily = append(s.Daily, DailyPoint{Date: d, NetRevenue: daily[d.Format("2006-01-02")]})
		}
	}
	return s
}

// defaultDisplayNames picks the most frequent product name per SKU, with
// ties broken by the alphabetically first name.
func defaultDisplayNames(ds *Dataset) map[string]string {
	counts := map[string]map[string]int{}
	if ds != nil {
		for _, l := range ds.Lines {
			if counts[l.SKU] == nil {
				counts[l.SKU] = map[string]int{}
			}
			counts[l.SKU][l.ProductName]++
		}
	}
	out := make(map[string]string, len(counts))
	for sku, names := range counts {
		best, bestN := "", -1
		for name, n := range names {
			if n > bestN || (n == bestN && name < best) {
				best, bestN = name, n
			}
		}
		out[sku] = best
	}
	return out
}

// resolveDisplayName returns the user override, then the default, then the SKU.
func resolveDisplayName(sku string, cfg *Config, defaults map[string]string) string {
	if cfg != nil {
		if n := cfg.DisplayNames[sku]; n != "" {
			return n
		}
	}
	if n := defaults[sku]; n != "" {
		return n
	}
	return sku
}
