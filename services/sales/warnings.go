package sales

import (
	"fmt"
	"strconv"
	"strings"
)

var channelPrefix = map[Channel]string{
	ChannelShopee:    "SHP",
	ChannelTikTok:    "TTS",
	ChannelTokopedia: "TKP",
}

// formatRp formats rupiah as Rp1.099.000. It mirrors views.FormatIDR so the
// domain does not import the view layer.
func formatRp(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatInt(v, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-Rp" + b.String()
	}
	return "Rp" + b.String()
}

// checkRowWarnings returns non-blocking warnings for a parsed line.
func checkRowWarnings(l OrderLine) []RowWarning {
	var out []RowWarning
	if want := int64(l.Qty) * l.UnitPrice; want != l.TotalPrice {
		out = append(out, RowWarning{
			Line: l.SourceLine, OrderID: l.OrderID, Channel: l.Channel, Code: WarnPriceMismatch,
			Message: fmt.Sprintf("Jumlah × harga satuan (%s) tidak sama dengan total (%s)", formatRp(want), formatRp(l.TotalPrice)),
		})
	}
	if !strings.HasPrefix(strings.ToUpper(l.OrderID), channelPrefix[l.Channel]) {
		prefix := []rune(l.OrderID)
		if len(prefix) > 3 {
			prefix = prefix[:3]
		}
		out = append(out, RowWarning{
			Line: l.SourceLine, OrderID: l.OrderID, Channel: l.Channel, Code: WarnChannelPrefixMismatch,
			Message: fmt.Sprintf("Prefix order_id (%s) tidak sesuai channel %s", string(prefix), l.Channel),
		})
	}
	return out
}
