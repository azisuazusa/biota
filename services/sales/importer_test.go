package sales

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPendingImportSample(t *testing.T) {
	cfg := DefaultConfig()
	p := pendingFor(t, sampleFile, cfg)

	assert.Equal(t, 495, p.TotalRows)
	assert.Len(t, p.Lines, 495)
	assert.Empty(t, p.ParseErrors)
	assert.Empty(t, p.Warnings)
	assert.Len(t, p.Duplicates, 22)
	assert.Empty(t, p.Unresolved)
	assert.Empty(t, p.UnknownStatus)
	assert.Len(t, p.LearnedMapping, 36)
	assert.Len(t, p.KnownSKUs, 12)
	assert.Len(t, p.Token, 32)

	skus := map[string]int{}
	for _, sku := range p.LearnedMapping {
		skus[sku]++
	}
	assert.Len(t, skus, 12)
	for sku, n := range skus {
		assert.Equal(t, 3, n, sku)
	}

	inferred := 0
	for _, l := range p.Lines {
		assert.NotEmpty(t, l.SKU, "line %d", l.SourceLine)
		if l.SKUInferred {
			inferred++
		}
	}
	assert.Equal(t, 76, inferred)

	extra := 0
	for _, g := range p.Duplicates {
		assert.GreaterOrEqual(t, len(g.Lines), 2)
		assert.Equal(t, g.Lines[0], g.Line.SourceLine)
		extra += len(g.Lines) - 1
	}
	assert.Equal(t, 22, extra)
}

func TestApplyDecisionSampleDuplicates(t *testing.T) {
	cfg := DefaultConfig()
	p := pendingFor(t, sampleFile, cfg)

	ds, _, err := applyDecision(p, ReviewDecision{Token: p.Token}, cfg)
	require.NoError(t, err)
	assert.Len(t, ds.Lines, 473)

	keep := map[string]bool{}
	for _, g := range p.Duplicates {
		keep[g.Key] = true
	}
	ds, _, err = applyDecision(p, ReviewDecision{Token: p.Token, KeepDuplicates: keep}, cfg)
	require.NoError(t, err)
	assert.Len(t, ds.Lines, 495)
	assert.Equal(t, "2026-10-01", ds.MinDate.Format("2006-01-02"))
	assert.Equal(t, "2026-10-31", ds.MaxDate.Format("2006-01-02"))
	assert.Equal(t, sampleFile, ds.FileName)
	for i := 1; i < len(ds.Lines); i++ {
		assert.False(t, ds.Lines[i].OrderDate.Before(ds.Lines[i-1].OrderDate))
	}
	assert.Len(t, p.Lines, 495, "pending import is not modified")
}

func TestBuildPendingImportEdgeCases(t *testing.T) {
	cfg := DefaultConfig()
	p := pendingFor(t, edgeFile, cfg)

	assert.Equal(t, 15, p.TotalRows)
	assert.Len(t, p.Lines, 9)
	assert.Len(t, p.ParseErrors, 6)
	assert.Empty(t, p.Duplicates)

	require.Len(t, p.Unresolved, 2)
	assert.Equal(t, UnresolvedSKU{ProductName: "Produk Ganda", Lines: []int{12}, CandidateSKUs: []string{"BIO-B", "BIO-C"}}, p.Unresolved[0])
	assert.Equal(t, UnresolvedSKU{ProductName: "Produk Misterius", Lines: []int{9}}, p.Unresolved[1])
	assert.NotContains(t, p.LearnedMapping, "Produk Ganda")
	assert.Equal(t, "BIO-A", p.LearnedMapping["Produk A"])

	assert.Equal(t, []UnknownStatus{{Status: "Menunggu Pembayaran", Count: 1}}, p.UnknownStatus)

	require.Len(t, p.Warnings, 3)
	assert.Equal(t, 14, p.Warnings[0].Line)
	assert.Equal(t, WarnPriceMismatch, p.Warnings[0].Code)
	assert.Equal(t, 15, p.Warnings[1].Line)
	assert.Equal(t, WarnChannelPrefixMismatch, p.Warnings[1].Code)
	assert.Equal(t, 16, p.Warnings[2].Line)
	assert.Equal(t, WarnChannelPrefixMismatch, p.Warnings[2].Code)

	ds, newCfg, err := applyDecision(p, ReviewDecision{
		Token:         p.Token,
		SKUChoices:    map[string]string{"Produk Ganda": "BIO-B", "Produk Misterius": " BIO-NEW "},
		StatusChoices: map[string]StatusClass{"Menunggu Pembayaran": StatusExcluded},
	}, cfg)
	require.NoError(t, err)
	require.Len(t, ds.Lines, 9)

	byLine := map[int]OrderLine{}
	for _, l := range ds.Lines {
		byLine[l.SourceLine] = l
	}
	for _, bad := range []int{3, 4, 5, 6, 7, 8} {
		assert.NotContains(t, byLine, bad)
	}
	// Warned rows are imported unchanged.
	assert.Equal(t, int64(150000), byLine[14].TotalPrice)
	assert.Equal(t, 2, byLine[14].Qty)
	assert.Equal(t, ChannelShopee, byLine[15].Channel)
	assert.Equal(t, "TKP123", byLine[15].OrderID)
	assert.Equal(t, ChannelTokopedia, byLine[16].Channel)
	// Manual SKU choices filled.
	assert.Equal(t, "BIO-NEW", byLine[9].SKU)
	assert.True(t, byLine[9].SKUInferred)
	assert.Equal(t, "BIO-B", byLine[12].SKU)

	// Config merge.
	assert.Equal(t, "BIO-NEW", newCfg.NameToSKU["Produk Misterius"])
	assert.Equal(t, "BIO-B", newCfg.NameToSKU["Produk Ganda"])
	assert.Equal(t, "BIO-A", newCfg.NameToSKU["Produk A"])
	c, ok := newCfg.Classify("menunggu pembayaran")
	require.True(t, ok)
	assert.Equal(t, StatusExcluded, c)
	// Input config untouched.
	assert.Empty(t, cfg.NameToSKU)
	_, ok = cfg.Classify("Menunggu Pembayaran")
	assert.False(t, ok)
}

func TestApplyDecisionValidation(t *testing.T) {
	cfg := DefaultConfig()
	p := pendingFor(t, edgeFile, cfg)
	status := map[string]StatusClass{"Menunggu Pembayaran": StatusIncluded}
	skus := map[string]string{"Produk Ganda": "BIO-B", "Produk Misterius": "BIO-X"}

	_, _, err := applyDecision(p, ReviewDecision{Token: p.Token, StatusChoices: status, SKUChoices: map[string]string{"Produk Ganda": "BIO-B"}}, cfg)
	assert.ErrorIs(t, err, ErrUnresolvedSKU)

	_, _, err = applyDecision(p, ReviewDecision{Token: p.Token, StatusChoices: status, SKUChoices: map[string]string{"Produk Ganda": "BIO-B", "Produk Misterius": "   "}}, cfg)
	assert.ErrorIs(t, err, ErrUnresolvedSKU)

	_, _, err = applyDecision(p, ReviewDecision{Token: p.Token, StatusChoices: status, SKUChoices: map[string]string{"Produk Ganda": "BIO-B", "Produk Misterius": strings.Repeat("X", 65)}}, cfg)
	assert.ErrorIs(t, err, ErrUnresolvedSKU)

	_, _, err = applyDecision(p, ReviewDecision{Token: p.Token, SKUChoices: skus}, cfg)
	assert.ErrorIs(t, err, ErrUnknownStatus)

	_, _, err = applyDecision(p, ReviewDecision{Token: p.Token, SKUChoices: skus, StatusChoices: map[string]StatusClass{"Menunggu Pembayaran": "maybe"}}, cfg)
	assert.ErrorIs(t, err, ErrUnknownStatus)

	_, _, err = applyDecision(p, ReviewDecision{Token: p.Token, SKUChoices: skus, StatusChoices: status}, cfg)
	assert.NoError(t, err)
}

func TestApplyDecisionNoValidLines(t *testing.T) {
	in := strings.Join(RequiredColumns, ",") + "\n" +
		"SHP1,bad-date,Shopee,BIO-A,Produk A,1,Rp1,Rp1,Selesai,Kota\n" +
		"SHP2,2026-10-01 10:00,Lazada,BIO-A,Produk A,1,Rp1,Rp1,Selesai,Kota\n"
	cfg := DefaultConfig()
	p, err := buildPendingImport("x.csv", strings.NewReader(in), cfg)
	require.NoError(t, err)
	assert.Len(t, p.ParseErrors, 2)
	assert.Empty(t, p.Lines)

	_, _, err = applyDecision(p, ReviewDecision{Token: p.Token}, cfg)
	assert.ErrorIs(t, err, ErrNoValidLines)
}

func TestBuildPendingImportUsesConfigMapping(t *testing.T) {
	in := strings.Join(RequiredColumns, ",") + "\n" +
		"SHP1,2026-10-01 10:00,Shopee,,Foo,1,Rp100,Rp100,Selesai,Kota\n"
	cfg := DefaultConfig()
	cfg.NameToSKU["Foo"] = "BIO-X"
	p, err := buildPendingImport("x.csv", strings.NewReader(in), cfg)
	require.NoError(t, err)
	assert.Empty(t, p.Unresolved)
	require.Len(t, p.Lines, 1)
	assert.Equal(t, "BIO-X", p.Lines[0].SKU)
	assert.True(t, p.Lines[0].SKUInferred)
	assert.Contains(t, p.KnownSKUs, "BIO-X")
}

func TestBuildPendingImportFileMappingBeatsConfig(t *testing.T) {
	in := strings.Join(RequiredColumns, ",") + "\n" +
		"SHP1,2026-10-01 10:00,Shopee,BIO-NEW,Foo,1,Rp100,Rp100,Selesai,Kota\n" +
		"SHP2,2026-10-01 10:00,Shopee,,Foo,1,Rp100,Rp100,Selesai,Kota\n"
	cfg := DefaultConfig()
	cfg.NameToSKU["Foo"] = "BIO-OLD"
	p, err := buildPendingImport("x.csv", strings.NewReader(in), cfg)
	require.NoError(t, err)
	assert.Equal(t, "BIO-NEW", p.Lines[1].SKU)
}
