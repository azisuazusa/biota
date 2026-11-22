package sales

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/azisuazusa/biota/pkg/util"
)

// rawRow is one CSV data record with its 1-based file line (header = 1).
type rawRow struct {
	Line  int
	Cells []string
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// parseCSV reads the header and every data record.
func parseCSV(r io.Reader) (rows []rawRow, colIdx map[string]int, err error) {
	br := bufio.NewReader(r)
	if b, _ := br.Peek(3); bytes.Equal(b, utf8BOM) {
		_, _ = br.Discard(3)
	}
	cr := csv.NewReader(br)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrEmptyCSV)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w: %w", util.GetPackageAndFunctionName(), ErrInvalidCSV, err)
	}

	colIdx = make(map[string]int, len(header))
	for i, h := range header {
		k := strings.ToLower(strings.TrimSpace(h))
		if _, dup := colIdx[k]; !dup {
			colIdx[k] = i
		}
	}
	var missing []string
	for _, c := range RequiredColumns {
		if _, ok := colIdx[c]; !ok {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return nil, nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), &MissingColumnsError{Columns: missing})
	}

	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w: %w", util.GetPackageAndFunctionName(), ErrInvalidCSV, err)
		}
		line, _ := cr.FieldPos(0)
		rows = append(rows, rawRow{Line: line, Cells: rec})
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrEmptyCSV)
	}
	return rows, colIdx, nil
}

// trimmedCells returns the 10 required cells, trimmed, in RequiredColumns order.
func trimmedCells(raw []string, idx map[string]int) []string {
	out := make([]string, len(RequiredColumns))
	for i, c := range RequiredColumns {
		if j := idx[c]; j < len(raw) {
			out[i] = strings.TrimSpace(raw[j])
		}
	}
	return out
}

// normalizeRow converts one raw record into an OrderLine, collecting every
// cell error on the row.
func normalizeRow(line int, raw []string, idx map[string]int) (OrderLine, []ParseError) {
	c := trimmedCells(raw, idx)
	get := func(col string) string {
		for i, rc := range RequiredColumns {
			if rc == col {
				return c[i]
			}
		}
		return ""
	}

	var errs []ParseError
	addErr := func(col, val, reason string) {
		errs = append(errs, ParseError{Line: line, Column: col, Value: val, Reason: reason})
	}

	l := OrderLine{
		SourceLine:  line,
		OrderID:     get("order_id"),
		SKU:         get("sku"),
		ProductName: get("product_name"),
		Status:      get("status"),
		BuyerCity:   get("buyer_city"),
	}
	if l.OrderID == "" {
		addErr("order_id", "", "order_id kosong")
	}
	var err error
	if l.OrderDate, l.HasTime, err = ParseOrderDate(get("order_date")); err != nil {
		addErr("order_date", get("order_date"), err.Error())
	}
	if l.Channel, err = NormalizeChannel(get("channel")); err != nil {
		addErr("channel", get("channel"), err.Error())
	}
	if l.ProductName == "" {
		addErr("product_name", "", "product_name kosong")
	}
	if l.Qty, err = ParseQty(get("qty")); err != nil {
		addErr("qty", get("qty"), err.Error())
	}
	if l.UnitPrice, err = ParseIDR(get("unit_price")); err != nil {
		addErr("unit_price", get("unit_price"), err.Error())
	}
	if l.TotalPrice, err = ParseIDR(get("total_price")); err != nil {
		addErr("total_price", get("total_price"), err.Error())
	}
	return l, errs
}
