// Package sales is the domain for importing marketplace order CSVs and
// computing net-revenue insight. It has no HTTP or filesystem code.
package sales

import "time"

// Channel is a normalized marketplace name.
type Channel string

const (
	ChannelShopee    Channel = "Shopee"
	ChannelTikTok    Channel = "TikTok Shop"
	ChannelTokopedia Channel = "Tokopedia"
)

// AllChannels lists the channels in display order.
var AllChannels = []Channel{ChannelShopee, ChannelTikTok, ChannelTokopedia}

// RequiredColumns are the CSV header columns that must be present.
var RequiredColumns = []string{"order_id", "order_date", "channel", "sku", "product_name", "qty", "unit_price", "total_price", "status", "buyer_city"}

// StatusClass says whether a status counts towards net revenue.
type StatusClass string

const (
	StatusIncluded StatusClass = "include"
	StatusExcluded StatusClass = "exclude"
)

// OrderLine is one normalized CSV row.
type OrderLine struct {
	SourceLine  int       `json:"source_line"`
	OrderID     string    `json:"order_id"`
	OrderDate   time.Time `json:"order_date"` // wall-clock as written in CSV, parsed with time.UTC as a neutral carrier (no TZ conversion)
	HasTime     bool      `json:"has_time"`
	Channel     Channel   `json:"channel"`
	SKU         string    `json:"sku"`
	SKUInferred bool      `json:"sku_inferred"`
	ProductName string    `json:"product_name"`
	Qty         int       `json:"qty"`
	UnitPrice   int64     `json:"unit_price"`
	TotalPrice  int64     `json:"total_price"`
	Status      string    `json:"status"`
	BuyerCity   string    `json:"buyer_city"`
}

// Dataset is the last confirmed, normalized import.
type Dataset struct {
	FileName   string      `json:"file_name"`
	UploadedAt time.Time   `json:"uploaded_at"`
	MinDate    time.Time   `json:"min_date"`
	MaxDate    time.Time   `json:"max_date"`
	Lines      []OrderLine `json:"lines"`
}

// Config holds user settings that persist across uploads.
type Config struct {
	StatusMapping map[string]StatusClass `json:"status_mapping"` // key: lowercased status
	StatusLabels  map[string]string      `json:"status_labels"`  // key: lowercased → original casing for display
	NameToSKU     map[string]string      `json:"name_to_sku"`    // key: exact trimmed product_name
	DisplayNames  map[string]string      `json:"display_names"`  // key: SKU; only user overrides
}

// ParseError describes one cell that could not be parsed.
type ParseError struct {
	Line   int    `json:"line"`
	Column string `json:"column"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

// WarningCode identifies a non-blocking row warning.
type WarningCode string

const (
	WarnPriceMismatch         WarningCode = "price_mismatch"
	WarnChannelPrefixMismatch WarningCode = "channel_prefix_mismatch"
)

// RowWarning flags suspicious data on a row that still gets imported.
type RowWarning struct {
	Line    int         `json:"line"`
	OrderID string      `json:"order_id"`
	Channel Channel     `json:"channel"`
	Code    WarningCode `json:"code"`
	Message string      `json:"message"`
}

// DuplicateGroup is ≥2 rows with identical raw cells.
type DuplicateGroup struct {
	Key   string    `json:"key"`   // sha1 of joined raw cells
	Lines []int     `json:"lines"` // source line numbers
	Line  OrderLine `json:"line"`  // representative normalized line
}

// UnresolvedSKU is a product name whose SKU must be chosen by the user.
type UnresolvedSKU struct {
	ProductName   string   `json:"product_name"`
	Lines         []int    `json:"lines"`
	CandidateSKUs []string `json:"candidate_skus"` // non-empty when file maps name to >1 SKU
}

// UnknownStatus is a status not present in the mapping.
type UnknownStatus struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// PendingImport is an uploaded file awaiting review. It lives in memory only.
type PendingImport struct {
	Token          string
	FileName       string
	CreatedAt      time.Time
	TotalRows      int
	Lines          []OrderLine // valid lines incl. all duplicate copies; SKU may be empty if unresolved
	ParseErrors    []ParseError
	Warnings       []RowWarning
	Duplicates     []DuplicateGroup
	Unresolved     []UnresolvedSKU
	UnknownStatus  []UnknownStatus
	LearnedMapping map[string]string // name→SKU pairs found in the file
	KnownSKUs      []string
}

// ReviewDecision carries the user's answers from the review page.
type ReviewDecision struct {
	Token          string                 `validate:"required"`
	KeepDuplicates map[string]bool        // DuplicateGroup.Key → keep all copies
	SKUChoices     map[string]string      // ProductName → SKU
	StatusChoices  map[string]StatusClass // Status → include/exclude
}

// Filter restricts the analytics.
type Filter struct {
	From     time.Time // inclusive, date only
	To       time.Time // inclusive, date only
	Channels []Channel // empty = none selected → empty result
}

// Summary is the analytics result for one filter.
type Summary struct {
	NetRevenue int64
	Orders     int
	Units      int
	AOV        int64
	ByChannel  []ChannelRow
	ByProduct  []ProductRow // all SKUs, sorted desc
	Top10      []ProductRow
	Daily      []DailyPoint // one per day From..To
}

// ChannelRow is one row of the per-channel table.
type ChannelRow struct {
	Channel    Channel
	Orders     int
	Units      int
	NetRevenue int64
	SharePct   float64
}

// ProductRow is one row of the per-product table.
type ProductRow struct {
	Rank        int
	SKU         string
	DisplayName string
	Units       int
	NetRevenue  int64
	SharePct    float64
}

// DailyPoint is one day of the revenue trend.
type DailyPoint struct {
	Date       time.Time
	NetRevenue int64
}

// ProductInfo is one SKU on the product settings page.
type ProductInfo struct {
	SKU         string
	DefaultName string
	Override    string
}
