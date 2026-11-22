package sales

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeChannel(t *testing.T) {
	cases := map[string]Channel{
		"Shopee": ChannelShopee, "shopee": ChannelShopee, "SHOPEE": ChannelShopee,
		"Tokopedia": ChannelTokopedia, "tokopedia": ChannelTokopedia, "TOKOPEDIA": ChannelTokopedia, "Tokped": ChannelTokopedia,
		"TikTok Shop": ChannelTikTok, "TIKTOK SHOP": ChannelTikTok, "Tiktok": ChannelTikTok,
		"  shopee  ": ChannelShopee,
	}
	for raw, want := range cases {
		got, err := NormalizeChannel(raw)
		require.NoError(t, err, raw)
		assert.Equal(t, want, got, raw)
	}
	_, err := NormalizeChannel("Lazada")
	assert.EqualError(t, err, "Channel tidak dikenali")
}

func TestParseOrderDate(t *testing.T) {
	cases := []struct {
		raw     string
		want    time.Time
		hasTime bool
	}{
		{"2026-10-21 08:24", time.Date(2026, 10, 21, 8, 24, 0, 0, time.UTC), true},
		{"14/10/2026 23:26:00", time.Date(2026, 10, 14, 23, 26, 0, 0, time.UTC), true},
		{"Oct 01, 2026", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), false},
	}
	for _, c := range cases {
		got, hasTime, err := ParseOrderDate(c.raw)
		require.NoError(t, err, c.raw)
		assert.True(t, c.want.Equal(got), c.raw)
		assert.Equal(t, time.UTC, got.Location())
		assert.Equal(t, c.hasTime, hasTime, c.raw)
	}
	got, _, err := ParseOrderDate("14/10/2026 23:26:00")
	require.NoError(t, err)
	assert.Equal(t, 14, got.Day())
	assert.Equal(t, time.October, got.Month())

	for _, bad := range []string{"2026/13/45", "", "31-10-2026"} {
		_, _, err := ParseOrderDate(bad)
		assert.EqualError(t, err, "Format tanggal tidak dikenali", bad)
	}
}

func TestParseIDR(t *testing.T) {
	cases := map[string]int64{
		"Rp1.099.000":   1099000,
		"Rp 1.099.000":  1099000,
		"1,099,000":     1099000,
		"IDR 1,099,000": 1099000,
		"IDR 2,199,000": 2199000,
		"1099000":       1099000,
		"Rp 2.198.000":  2198000,
		"rp159.000":     159000,
		"idr 159,000":   159000,
	}
	for raw, want := range cases {
		got, err := ParseIDR(raw)
		require.NoError(t, err, raw)
		assert.Equal(t, want, got, raw)
	}
	for _, bad := range []string{"Rp-", "", "abc", "Rp0", "-5000", "Rp"} {
		_, err := ParseIDR(bad)
		assert.EqualError(t, err, "Format harga tidak dikenali", bad)
	}
}

func TestParseQty(t *testing.T) {
	for raw, want := range map[string]int{"1": 1, "3": 3, " 2 ": 2} {
		got, err := ParseQty(raw)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	for _, bad := range []string{"0", "-1", "abc", "", "1.5"} {
		_, err := ParseQty(bad)
		assert.EqualError(t, err, "Jumlah tidak valid", bad)
	}
}

func TestDateOnly(t *testing.T) {
	got := DateOnly(time.Date(2026, 10, 21, 8, 24, 5, 9, time.UTC))
	assert.Equal(t, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), got)
}
