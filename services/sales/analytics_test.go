package sales

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullFilter(ds *Dataset) Filter {
	return Filter{From: ds.MinDate, To: ds.MaxDate, Channels: AllChannels}
}

func TestSummarizeSampleAcceptance(t *testing.T) {
	ds, cfg := sampleDataset(t, false)
	s := summarize(ds, cfg, fullFilter(ds))

	assert.Equal(t, int64(360363000), s.NetRevenue)
	assert.Equal(t, 353, s.Orders)
	assert.Equal(t, 517, s.Units)
	assert.Equal(t, int64(1020858), s.AOV)

	require.Len(t, s.ByChannel, 3)
	assert.Equal(t, ChannelShopee, s.ByChannel[0].Channel)
	assert.Equal(t, int64(228176000), s.ByChannel[0].NetRevenue)
	assert.Equal(t, ChannelTikTok, s.ByChannel[1].Channel)
	assert.Equal(t, int64(87666000), s.ByChannel[1].NetRevenue)
	assert.Equal(t, ChannelTokopedia, s.ByChannel[2].Channel)
	assert.Equal(t, int64(44521000), s.ByChannel[2].NetRevenue)
	var pct float64
	orders := 0
	for _, c := range s.ByChannel {
		pct += c.SharePct
		orders += c.Orders
	}
	assert.InDelta(t, 100, pct, 1e-9)
	assert.Equal(t, 353, orders)

	want := []struct {
		sku   string
		rev   int64
		units int
	}{
		{"BIO-CAM-BBY", 67548000, 52},
		{"BIO-CAM-BF", 57148000, 52},
		{"BIO-PET-FD", 48546000, 54},
		{"BIO-CAM-E4G", 43980000, 20},
		{"BIO-CAM-OPZ", 32154000, 46},
		{"BIO-CAM-DB", 29563000, 37},
		{"BIO-CAM-IPZ", 20205000, 45},
		{"BIO-CAM-IPL", 17799000, 51},
		{"BIO-PET-WF", 13566000, 34},
		{"BIO-LGT-HEX", 13491000, 9},
		{"BIO-SD-V128", 9699000, 61},
		{"BIO-SD-S64", 6664000, 56},
	}
	require.Len(t, s.ByProduct, 12)
	for i, w := range want {
		p := s.ByProduct[i]
		assert.Equal(t, i+1, p.Rank)
		assert.Equal(t, w.sku, p.SKU, "rank %d", i+1)
		assert.Equal(t, w.rev, p.NetRevenue, w.sku)
		assert.Equal(t, w.units, p.Units, w.sku)
		assert.NotEmpty(t, p.DisplayName)
	}
	require.Len(t, s.Top10, 10)
	assert.Equal(t, "BIO-CAM-BBY", s.Top10[0].SKU)
	assert.Equal(t, int64(67548000), s.Top10[0].NetRevenue)
	assert.Equal(t, 52, s.Top10[0].Units)

	require.Len(t, s.Daily, 31)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), s.Daily[0].Date)
	assert.Equal(t, time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), s.Daily[30].Date)
	var sum int64
	for _, d := range s.Daily {
		sum += d.NetRevenue
	}
	assert.Equal(t, s.NetRevenue, sum)
}

func TestSummarizeSampleKeepAllDuplicates(t *testing.T) {
	ds, cfg := sampleDataset(t, true)
	s := summarize(ds, cfg, fullFilter(ds))
	assert.Equal(t, int64(375865000), s.NetRevenue)
	assert.Equal(t, 353, s.Orders)
	assert.Equal(t, 545, s.Units)
}

func TestSummarizeChannelFilter(t *testing.T) {
	ds, cfg := sampleDataset(t, false)
	f := fullFilter(ds)
	f.Channels = []Channel{ChannelTokopedia}
	s := summarize(ds, cfg, f)
	assert.Equal(t, int64(44521000), s.NetRevenue)
	require.Len(t, s.ByChannel, 1)
	assert.Equal(t, ChannelTokopedia, s.ByChannel[0].Channel)
	assert.InDelta(t, 100, s.ByChannel[0].SharePct, 1e-9)

	f.Channels = []Channel{ChannelShopee}
	assert.Equal(t, int64(228176000), summarize(ds, cfg, f).NetRevenue)
}

func TestSummarizeNoChannels(t *testing.T) {
	ds, cfg := sampleDataset(t, false)
	f := fullFilter(ds)
	f.Channels = nil
	s := summarize(ds, cfg, f)
	assert.Zero(t, s.NetRevenue)
	assert.Zero(t, s.Orders)
	assert.Zero(t, s.Units)
	assert.Zero(t, s.AOV)
	assert.Empty(t, s.ByChannel)
	assert.Empty(t, s.ByProduct)
	assert.Empty(t, s.Top10)
	assert.Len(t, s.Daily, 31)
}

func TestSummarizeSingleDayAndDateOnlyRows(t *testing.T) {
	ds, cfg := sampleDataset(t, false)
	day := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)

	// Expected values computed directly from the lines.
	var want, tkp int64
	for _, l := range ds.Lines {
		if c, _ := cfg.Classify(l.Status); c != StatusIncluded || !DateOnly(l.OrderDate).Equal(day) {
			continue
		}
		want += l.TotalPrice
		if l.Channel == ChannelTokopedia {
			require.False(t, l.HasTime, "Tokopedia rows are date-only")
			tkp += l.TotalPrice
		}
	}
	require.Positive(t, tkp, "fixture must have Tokopedia sales on the chosen day")

	s := summarize(ds, cfg, Filter{From: day, To: day, Channels: AllChannels})
	assert.Equal(t, want, s.NetRevenue)
	require.Len(t, s.Daily, 1)
	assert.Equal(t, want, s.Daily[0].NetRevenue)

	s = summarize(ds, cfg, Filter{From: day, To: day, Channels: []Channel{ChannelTokopedia}})
	assert.Equal(t, tkp, s.NetRevenue)
}

func TestSummarizeClassificationAtQueryTime(t *testing.T) {
	ds, cfg := sampleDataset(t, false)
	base := summarize(ds, cfg, fullFilter(ds)).NetRevenue

	cfg2 := cfg.Clone()
	cfg2.SetStatus("Dikirim", StatusExcluded)
	lower := summarize(ds, cfg2, fullFilter(ds)).NetRevenue
	assert.Less(t, lower, base)

	cfg2.SetStatus("Dikirim", StatusIncluded)
	assert.Equal(t, base, summarize(ds, cfg2, fullFilter(ds)).NetRevenue)
}

func TestDisplayNames(t *testing.T) {
	ds := &Dataset{Lines: []OrderLine{
		{SKU: "A", ProductName: "Zeta"},
		{SKU: "A", ProductName: "Alpha"},
		{SKU: "A", ProductName: "Zeta"},
		{SKU: "B", ProductName: "Yellow"},
		{SKU: "B", ProductName: "Blue"},
	}}
	d := defaultDisplayNames(ds)
	assert.Equal(t, "Zeta", d["A"], "most frequent wins")
	assert.Equal(t, "Blue", d["B"], "tie goes to the alphabetically first name")

	cfg := DefaultConfig()
	assert.Equal(t, "Zeta", resolveDisplayName("A", cfg, d))
	cfg.DisplayNames["A"] = "Custom"
	assert.Equal(t, "Custom", resolveDisplayName("A", cfg, d))
	assert.Equal(t, "C", resolveDisplayName("C", cfg, d))

	sds, scfg := sampleDataset(t, false)
	scfg.DisplayNames["BIO-CAM-BBY"] = "Kamera Andalan"
	s := summarize(sds, scfg, fullFilter(sds))
	assert.Equal(t, "Kamera Andalan", s.ByProduct[0].DisplayName)
	assert.Equal(t, "Kamera Andalan", s.Top10[0].DisplayName)
}
