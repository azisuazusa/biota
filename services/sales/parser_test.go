package sales

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCSVSample(t *testing.T) {
	rows, idx, err := parseCSV(openFixture(t, sampleFile))
	require.NoError(t, err)
	require.Len(t, rows, 495)
	assert.Equal(t, 2, rows[0].Line)
	assert.Equal(t, 496, rows[494].Line)

	nErr := 0
	for _, r := range rows {
		_, errs := normalizeRow(r.Line, r.Cells, idx)
		nErr += len(errs)
	}
	assert.Zero(t, nErr)
}

func TestParseCSVMissingColumn(t *testing.T) {
	_, _, err := parseCSV(openFixture(t, missingFile))
	require.ErrorIs(t, err, ErrMissingColumns)
	var mce *MissingColumnsError
	require.True(t, errors.As(err, &mce))
	assert.Equal(t, []string{"status"}, mce.Columns)
}

func TestParseCSVHeaderOnly(t *testing.T) {
	_, _, err := parseCSV(openFixture(t, headerOnly))
	assert.ErrorIs(t, err, ErrEmptyCSV)

	_, _, err = parseCSV(strings.NewReader(""))
	assert.ErrorIs(t, err, ErrEmptyCSV)
}

func TestParseCSVInvalid(t *testing.T) {
	in := strings.Join(RequiredColumns, ",") + "\n" + `a,"b` + "\"x,c\n"
	_, _, err := parseCSV(strings.NewReader(in))
	assert.ErrorIs(t, err, ErrInvalidCSV)
}

func TestParseCSVEdgeCases(t *testing.T) {
	rows, idx, err := parseCSV(openFixture(t, edgeFile))
	require.NoError(t, err, "BOM and mixed-case, reordered header must be accepted")
	require.Len(t, rows, 15)

	var got []ParseError
	valid := 0
	for _, r := range rows {
		_, errs := normalizeRow(r.Line, r.Cells, idx)
		if len(errs) == 0 {
			valid++
		}
		got = append(got, errs...)
	}
	assert.Equal(t, 9, valid)
	assert.Equal(t, []ParseError{
		{Line: 3, Column: "order_date", Value: "2026/13/45", Reason: "Format tanggal tidak dikenali"},
		{Line: 4, Column: "channel", Value: "Lazada", Reason: "Channel tidak dikenali"},
		{Line: 5, Column: "qty", Value: "0", Reason: "Jumlah tidak valid"},
		{Line: 6, Column: "qty", Value: "abc", Reason: "Jumlah tidak valid"},
		{Line: 7, Column: "unit_price", Value: "Rp-", Reason: "Format harga tidak dikenali"},
		{Line: 8, Column: "order_id", Value: "", Reason: "order_id kosong"},
	}, got)

	// First data row is mapped through the reordered header.
	l, errs := normalizeRow(rows[0].Line, rows[0].Cells, idx)
	require.Empty(t, errs)
	assert.Equal(t, "Selesai", l.Status)
	assert.Equal(t, "Jakarta", l.BuyerCity)
}

func TestNormalizeRowCollectsAllErrors(t *testing.T) {
	idx := map[string]int{}
	for i, c := range RequiredColumns {
		idx[c] = i
	}
	_, errs := normalizeRow(9, []string{"", "x", "y", "", "", "0", "a", "b", "s", "c"}, idx)
	cols := []string{}
	for _, e := range errs {
		cols = append(cols, e.Column)
		assert.Equal(t, 9, e.Line)
	}
	assert.Equal(t, []string{"order_id", "order_date", "channel", "product_name", "qty", "unit_price", "total_price"}, cols)

	// Short records: missing cells are treated as empty.
	_, errs = normalizeRow(10, []string{"SHP1"}, idx)
	assert.NotEmpty(t, errs)
}
