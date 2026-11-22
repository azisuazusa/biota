package sales

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func line(orderID string, ch Channel, qty int, unit, total int64) OrderLine {
	return OrderLine{SourceLine: 7, OrderID: orderID, Channel: ch, Qty: qty, UnitPrice: unit, TotalPrice: total}
}

func TestCheckRowWarnings(t *testing.T) {
	assert.Empty(t, checkRowWarnings(line("SHP1", ChannelShopee, 2, 100000, 200000)))
	assert.Empty(t, checkRowWarnings(line("TTS1", ChannelTikTok, 1, 5, 5)))
	assert.Empty(t, checkRowWarnings(line("tkp1", ChannelTokopedia, 1, 5, 5)))

	w := checkRowWarnings(line("SHP12", ChannelShopee, 2, 100000, 150000))
	require.Len(t, w, 1)
	assert.Equal(t, WarnPriceMismatch, w[0].Code)
	assert.Equal(t, "Jumlah × harga satuan (Rp200.000) tidak sama dengan total (Rp150.000)", w[0].Message)
	assert.Equal(t, 7, w[0].Line)
	assert.Equal(t, "SHP12", w[0].OrderID)

	w = checkRowWarnings(line("TKP123", ChannelShopee, 1, 5, 5))
	require.Len(t, w, 1)
	assert.Equal(t, WarnChannelPrefixMismatch, w[0].Code)
	assert.Equal(t, "Prefix order_id (TKP) tidak sesuai channel Shopee", w[0].Message)

	w = checkRowWarnings(line("XYZ1", ChannelTokopedia, 1, 5, 5))
	require.Len(t, w, 1)
	assert.Equal(t, "Prefix order_id (XYZ) tidak sesuai channel Tokopedia", w[0].Message)

	w = checkRowWarnings(line("AB", ChannelTikTok, 3, 10, 20))
	require.Len(t, w, 2)
	assert.Equal(t, WarnPriceMismatch, w[0].Code)
	assert.Equal(t, WarnChannelPrefixMismatch, w[1].Code)
	assert.Equal(t, "Prefix order_id (AB) tidak sesuai channel TikTok Shop", w[1].Message)
}

func TestFormatRp(t *testing.T) {
	assert.Equal(t, "Rp0", formatRp(0))
	assert.Equal(t, "Rp999", formatRp(999))
	assert.Equal(t, "Rp1.000", formatRp(1000))
	assert.Equal(t, "Rp1.099.000", formatRp(1099000))
	assert.Equal(t, "Rp360.363.000", formatRp(360363000))
}
