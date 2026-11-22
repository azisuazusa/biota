package sales

import (
	"strconv"
	"strings"
	"time"
)

// reasonError is a row parse error whose text is the Indonesian reason shown
// in the review page.
type reasonError string

func (e reasonError) Error() string { return string(e) }

// Reasons shown in the review page for row parse errors.
const (
	reasonChannel reasonError = "Channel tidak dikenali"
	reasonDate    reasonError = "Format tanggal tidak dikenali"
	reasonQty     reasonError = "Jumlah tidak valid"
	reasonPrice   reasonError = "Format harga tidak dikenali"
)

var channelAliases = map[string]Channel{
	"shopee":      ChannelShopee,
	"tiktok shop": ChannelTikTok,
	"tiktok":      ChannelTikTok,
	"tokopedia":   ChannelTokopedia,
	"tokped":      ChannelTokopedia,
}

// NormalizeChannel maps a raw channel spelling to a Channel.
func NormalizeChannel(raw string) (Channel, error) {
	if c, ok := channelAliases[strings.ToLower(strings.TrimSpace(raw))]; ok {
		return c, nil
	}
	return "", reasonChannel
}

var dateLayouts = []struct {
	layout  string
	hasTime bool
}{
	{"2006-01-02 15:04", true},
	{"02/01/2006 15:04:05", true},
	{"Jan 02, 2006", false},
}

// ParseOrderDate tries the known layouts in order. Times are wall-clock
// values carried in time.UTC with no timezone conversion.
func ParseOrderDate(raw string) (t time.Time, hasTime bool, err error) {
	s := strings.TrimSpace(raw)
	for _, l := range dateLayouts {
		if t, err := time.ParseInLocation(l.layout, s, time.UTC); err == nil {
			return t, l.hasTime, nil
		}
	}
	return time.Time{}, false, reasonDate
}

// ParseIDR parses a rupiah amount in any of the export's formats.
func ParseIDR(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "idr"):
		s = s[3:]
	case strings.HasPrefix(lower, "rp"):
		s = s[2:]
	}
	s = strings.NewReplacer(" ", "", ".", "", ",", "").Replace(s)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v <= 0 {
		return 0, reasonPrice
	}
	return v, nil
}

// ParseQty parses a quantity ≥ 1.
func ParseQty(raw string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v < 1 {
		return 0, reasonQty
	}
	return v, nil
}

// DateOnly truncates t to 00:00 UTC on its calendar date.
func DateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
