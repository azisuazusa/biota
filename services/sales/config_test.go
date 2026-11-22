package sales

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Len(t, cfg.StatusMapping, 12)
	assert.Len(t, cfg.StatusLabels, 12)
	assert.NotNil(t, cfg.NameToSKU)
	assert.NotNil(t, cfg.DisplayNames)
	assert.Equal(t, "Dikomplain - Retur", cfg.StatusLabels["dikomplain - retur"])
}

func TestConfigClassify(t *testing.T) {
	cfg := DefaultConfig()

	c, ok := cfg.Classify(" selesai ")
	require.True(t, ok)
	assert.Equal(t, StatusIncluded, c)

	c, ok = cfg.Classify("REFUNDED")
	require.True(t, ok)
	assert.Equal(t, StatusExcluded, c)

	_, ok = cfg.Classify("Menunggu Pembayaran")
	assert.False(t, ok)
}

func TestConfigSetStatus(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetStatus(" Menunggu Pembayaran ", StatusExcluded)
	c, ok := cfg.Classify("menunggu pembayaran")
	require.True(t, ok)
	assert.Equal(t, StatusExcluded, c)
	assert.Equal(t, "Menunggu Pembayaran", cfg.StatusLabels["menunggu pembayaran"])
}

func TestConfigClone(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NameToSKU["A"] = "SKU-A"
	cfg.DisplayNames["SKU-A"] = "Produk A"

	cl := cfg.Clone()
	cl.SetStatus("Selesai", StatusExcluded)
	cl.NameToSKU["B"] = "SKU-B"
	cl.DisplayNames["SKU-A"] = "Lain"

	c, _ := cfg.Classify("Selesai")
	assert.Equal(t, StatusIncluded, c)
	assert.NotContains(t, cfg.NameToSKU, "B")
	assert.Equal(t, "Produk A", cfg.DisplayNames["SKU-A"])

	empty := (&Config{}).Clone()
	assert.NotNil(t, empty.StatusMapping)
	assert.NotNil(t, empty.NameToSKU)
}
