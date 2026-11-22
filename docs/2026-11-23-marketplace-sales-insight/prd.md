# PRD: BIOTA Marketplace Sales Insight

**Date:** 2026-11-23
**Status:** Draft

---

## 1. Problem Statement

BIOTA sells on three marketplaces (Shopee, TikTok Shop, Tokopedia) and gets one combined order export per period, e.g. the October 2026 export (495 rows). Columns: `order_id, order_date, channel, sku, product_name, qty, unit_price, total_price, status, buyer_city`. Each row is one order line, and an order can have several lines (52 orders in the sample have more than one row).

Nobody can read business numbers from this file directly, because profiling the sample found these data quality problems:

| Problem | Evidence in sample |
|---|---|
| Channel spelled 10 different ways | `Shopee`/`shopee`/`SHOPEE`, `Tokopedia`/`tokopedia`/`TOKOPEDIA`/`Tokped`, `TikTok Shop`/`TIKTOK SHOP`/`Tiktok` |
| Date format depends on the channel | Shopee `2026-10-21 08:24`, TikTok `14/10/2026 23:26:00` (DD/MM/YYYY), Tokopedia `Oct 01, 2026` (no time) |
| ~6 price formats | `Rp1.099.000`, `Rp 1.099.000`, `1,099,000`, `IDR 1,099,000`, `1099000` |
| Missing SKU | 76 rows have an empty `sku`. Every one of those `product_name`s also appears on another row that has a SKU, and no name maps to two SKUs |
| 3 listing names per SKU | e.g. `BIO-CAM-BF` = "BIOTA Smart Camera Battery Fixed - CCTV Wifi Tanpa Kabel" / "Kamera CCTV Battery Fixed BIOTA" / "BIOTA Smart Home - Kamera Battery Fixed CCTV Wifi" |
| Exact duplicate rows | 22 rows identical to another row in every column |
| Status wording differs per marketplace | 12 distinct status values across 3 languages/styles |

The app turns this file into reliable net-revenue insight: per channel, per product, top 10 products, and a daily trend, filterable by date and channel.

---

## 2. Why This Matters

Without the app, getting net revenue means hand-cleaning the CSV every period: unifying channel names, converting three date and six price formats, filling in SKUs, deciding which statuses count, and removing duplicates. Each step can go wrong silently. On the sample, dropping the 22 duplicate rows versus keeping them changes net revenue by **Rp15.502.000** (Rp360.363.000 vs Rp375.865.000). Counting cancelled or refunded orders would overstate it even more. Decisions about channels and products need numbers computed the same way every time.

---

## 3. Who Is This For

**Primary:** The BIOTA business owner or operator, a single user. They upload the marketplace export at `/upload`, resolve issues at `/upload/review`, and read insights at `/` (dashboard).
**Secondary:** None. The app has no login and no multi-user features.

---

## 4. What Success Looks Like

Reference values below come from the **synthetic** fixture `test_data/sample_orders_oct2026.csv`, with the default review choices (duplicates dropped). They are the acceptance numbers for QA. The fixture copies the real export's structure: 495 rows, 22 duplicate pairs, 76 empty SKUs, 12 SKUs × 3 names, and every format variant. Its order IDs, dates, quantities, statuses and cities are random. The real export isn't committed, because the repo is public.

| Metric | Baseline | Target | Timeframe |
|--------|----------|--------|-----------|
| Rows parsed from sample CSV without parse error | Manual cleaning | 495 / 495 | At launch |
| Duplicate groups shown in review for sample | N/A | 22 groups, all defaulted to "drop" | At launch |
| Missing SKUs resolved automatically for sample | N/A | 76 / 76 (no manual prompt) | At launch |
| Row warnings shown for sample | N/A | 0 (every sample row has matching qty×price and prefix) | At launch |
| Warnings for a row with wrong total or wrong prefix | N/A | Listed in review, row still imported | At launch |
| Net revenue, full range, all channels | N/A | **Rp360.363.000** | At launch |
| Net revenue per channel | N/A | Shopee Rp228.176.000 · TikTok Shop Rp87.666.000 · Tokopedia Rp44.521.000 | At launch |
| Included orders / units / AOV | N/A | 353 orders · 517 units · AOV Rp1.020.858 (rounded) | At launch |
| Top product by net revenue | N/A | BIO-CAM-BBY, Rp67.548.000, 52 units | At launch |
| Products in revenue table / top-10 chart | N/A | 12 rows / 10 bars | At launch |
| If all duplicates are kept in review | N/A | Net revenue Rp375.865.000, 353 orders, 545 units | At launch |

Full expected per-product values (duplicates dropped), sorted by net revenue:
BIO-CAM-BBY 67.548.000 (52 u) · BIO-CAM-BF 57.148.000 (52) · BIO-PET-FD 48.546.000 (54) · BIO-CAM-E4G 43.980.000 (20) · BIO-CAM-OPZ 32.154.000 (46) · BIO-CAM-DB 29.563.000 (37) · BIO-CAM-IPZ 20.205.000 (45) · BIO-CAM-IPL 17.799.000 (51) · BIO-PET-WF 13.566.000 (34) · BIO-LGT-HEX 13.491.000 (9) · BIO-SD-V128 9.699.000 (61) · BIO-SD-S64 6.664.000 (56).

---

## 5. Scope

### In Scope

**Tech stack & delivery**
- Go, **chi** router (`github.com/go-chi/chi/v5`), **templ** views (`github.com/a-h/templ`), HTMX, TailwindCSS, Alpine.js, **Chart.js**.
- Shipped as a **Docker image**. Single user, **no authentication**.
- UI language **Bahasa Indonesia**. Money formatted `Rp1.099.000` (dot thousands separator, no decimals). Dates shown as `21 Okt 2026`.
- HTMX, Alpine.js and Chart.js are vendored into `static/vendor/`, so the container needs no internet access at runtime.

**Storage (no database)**
- Decided: **no database**. Each confirmed upload **replaces all previous data**, and only the latest CSV is analyzed. Reason: you chose replace-all over accumulating history.
- Two JSON files are saved on a Docker volume (`DATA_DIR`, default `/data`):
  - `dataset.json`: the last confirmed, normalized dataset plus metadata (original filename, uploaded_at, counts). It is loaded into memory at startup so the dashboard survives restarts.
  - `config.json`: status mapping, product_name→SKU mapping, and SKU display names. It persists across uploads.
- Writes are atomic (write a temp file in the same dir, then `os.Rename`) so a crash can't corrupt the files.
- A pending (not yet confirmed) import lives **in memory only**. A new upload discards the previous pending import. A restart discards it as well, which is acceptable because the user simply re-uploads.

**CSV parsing & normalization**
- Header must contain all 10 columns: `order_id, order_date, channel, sku, product_name, qty, unit_price, total_price, status, buyer_city`. They are matched case-insensitively after trimming, in any order, and a UTF-8 BOM is tolerated. If any column is missing, the **whole upload is rejected** with a message naming the missing columns. This is a file-level error, not a row error.
- Every cell is whitespace-trimmed before processing.
- **Channel**: case-insensitive alias map →
  | Raw (lowercased) | Normalized |
  |---|---|
  | `shopee` | `Shopee` |
  | `tiktok shop`, `tiktok` | `TikTok Shop` |
  | `tokopedia`, `tokped` | `Tokopedia` |
  Any other value is a row parse error. The channel column is authoritative. The `order_id` prefix is checked only to raise a warning (see **Row warnings** below).
- **order_date**: try these Go layouts in order. The first match wins, and if none match it's a row parse error:
  | Layout | Example | Has time |
  |---|---|---|
  | `2006-01-02 15:04` | `2026-10-21 08:24` | yes |
  | `02/01/2006 15:04:05` | `14/10/2026 23:26:00` (day first) | yes |
  | `Jan 02, 2006` | `Oct 01, 2026` | no |
  Stored as local wall-clock time with **no timezone conversion**. Date-only values get time 00:00 and `HasTime=false`. Filtering and grouping use the calendar date only.
- **Prices** (`unit_price`, `total_price`): remove the `Rp`/`IDR` prefix (case-insensitive), spaces, `.` and `,`, then parse what's left as a positive integer IDR. An empty or non-numeric result, or a value ≤ 0, is a row parse error. This works because no IDR amount in the source has decimals.
- **qty**: integer ≥ 1. Anything else is a row parse error.
- **order_id** and **product_name** must not be empty. If empty, it's a row parse error.
- **status**: kept as the raw trimmed text. It's classified against the mapping case-insensitively, with no other transformation.
- **buyer_city**: kept as is. It is shown nowhere in this version, but stored for future use.

**Row warnings (non-blocking)**
- Warnings are checked on rows that parsed successfully. They **do not block** confirm, and the rows **are imported unchanged**. They exist only to make suspicious data visible.
  | Code | Condition | Effect on numbers | Message |
  |---|---|---|---|
  | `price_mismatch` | `qty × unit_price ≠ total_price` | None. Net revenue always uses `total_price` | "Jumlah × harga satuan (Rp{a}) tidak sama dengan total (Rp{b})" |
  | `channel_prefix_mismatch` | `order_id` prefix doesn't match the normalized channel. Expected prefixes: `SHP`→Shopee, `TTS`→TikTok Shop, `TKP`→Tokopedia. An order_id with none of these prefixes also warns | None. The channel column is used | "Prefix order_id ({prefix}) tidak sesuai channel {channel}" |
- A row can carry both warnings. Warnings on duplicate copies are reported once per source line.

**SKU inference**
- If `sku` is non-empty, use it as is.
- If `sku` is empty, resolve it from `product_name` in this order:
  1. Name→SKU pairs from rows **in the same file** that have a SKU.
  2. The saved `config.json` name→SKU mapping.
  3. Still unresolved, or the same file maps that name to **more than one** SKU: the name is listed in review as **unresolved**. The user picks an existing SKU (dropdown of all SKUs known from the file and config) or types a new one. Every unresolved name must be assigned before confirm is enabled.
- On confirm, all name→SKU pairs from the file and every manual choice are **saved to `config.json`**, so later uploads resolve them automatically.

**Duplicate detection**
- A duplicate group is ≥2 rows whose **10 trimmed raw cell values are identical**. The sample has no rows that differ only in formatting.
- Review lists each group: the row content, how many times it appears, and the source line numbers. Each group has one checkbox, **"Simpan duplikat"**, **unchecked by default** (drop). Unchecked keeps one copy, checked keeps every copy.

**Status classification**
- Default mapping (seeded into `config.json` on first start):
  | Included in net revenue (shipping / completed) | Excluded (cancelled / returned) |
  |---|---|
  | Selesai, Completed, Pesanan Selesai, Dikirim, In transit, Sedang Dikirim | Batal, Cancelled, Dibatalkan, Pengembalian Dana, Refunded, Dikomplain - Retur |
- A status in the uploaded file that isn't in the mapping is listed in review as **unknown**, with its row count. The user must choose "Masuk pendapatan" (include) or "Tidak dihitung" (exclude) for each one before confirm is enabled. The choice is saved to `config.json`.
- The mapping can also be edited on `/settings/status`. An edit takes effect immediately on the current dataset, because classification runs at query time and isn't stored per row.

**Upload flow**
1. `/upload`: file input (`.csv`, max size from `MAX_UPLOAD_MB`, default 10), submitted via HTMX `multipart/form-data`.
2. The server parses, normalizes, infers SKUs, detects duplicates and unknown statuses, then saves a pending import in memory with a random token.
3. `/upload/review?token=…` shows summary counts (total rows, valid rows, parse errors, warnings, duplicate groups, unresolved SKUs, unknown statuses) and these sections:
   - **Baris gagal dibaca** (parse errors): line number, column, raw value, reason. These rows are **skipped on confirm**.
   - **Peringatan** (row warnings): line number, order_id, channel, warning message. Shown in amber. These rows **are imported**, and no user action is needed.
   - **Duplikat**: per-group keep/drop as described above.
   - **SKU belum dikenali**: per-name SKU picker.
   - **Status baru**: per-status include/exclude.
   If no section has items, review shows only the counts and the confirm button.
4. **Konfirmasi impor**: the server validates that every unresolved SKU and unknown status has an answer. It applies duplicate choices, saves config updates, **replaces** `dataset.json` and the in-memory dataset, clears the pending import, then redirects to `/` (dashboard, full range).
5. **Batal**: discards the pending import. The current dataset is unchanged.

**Dashboard (`/`)**
- Empty state when no dataset exists: "Belum ada data. Unggah file CSV terlebih dahulu." with a link to `/upload`.
- Header shows the current dataset's filename, upload time, and date range covered.
- **Filters** (apply to every widget below):
  - Date range: two `<input type="date">` fields (Dari / Sampai), day-level and **inclusive**. They **default to the dataset's earliest and latest order date**. **No presets**: the user decided against them, because presets anchored to today would hide historical uploads. Invalid values (from > to, or outside the dataset's dates) show an inline error.
  - Channel: multi-select checkboxes for Shopee, TikTok Shop and Tokopedia, all checked by default. If all are unchecked, every widget shows the "Tidak ada data untuk filter ini" empty state.
  - Changing a filter triggers an HTMX `hx-get` that re-renders the widget area and pushes the query string (`?from=YYYY-MM-DD&to=YYYY-MM-DD&channel=Shopee&channel=…`) to the URL, so filtered views can be bookmarked or reloaded.
- **Definitions** (all respect the filters):
  - *Included line*: a line whose status is mapped to "include".
  - **Net revenue** = SUM(`total_price`) of included lines. Decided: no fee or discount deduction, because the CSV has no such columns.
  - **Pesanan (orders)** = COUNT(DISTINCT `order_id`) over included lines.
  - **Unit terjual** = SUM(`qty`) over included lines.
  - **AOV** = net revenue ÷ orders, rounded to the nearest rupiah. Shown as `Rp0` when orders = 0.
  - A product means a **SKU**, labeled by its display name.
- **Widgets:**
  | Widget | Content |
  |---|---|
  | KPI cards (4) | Pendapatan Bersih, Pesanan, Unit Terjual, Rata-rata Nilai Pesanan (AOV) |
  | Pendapatan per Channel | Table: Channel, Pesanan, Unit, Pendapatan Bersih, % share. Plus Chart.js bar chart |
  | Pendapatan per Produk | Table of **all** SKUs with included lines: Rank, SKU, Nama Produk, Unit, Pendapatan Bersih, % share. Sorted by revenue desc, ties by SKU asc |
  | Top 10 Produk | Chart.js horizontal bar chart of the first 10 rows of the product table |
  | Tren Pendapatan Harian | Chart.js line chart, one point **per calendar day** in the selected range, with days that have no sales plotted as 0 |
- Charts receive data through a `<script type="application/json">` block rendered by templ. A small Alpine component (`x-data="chart(...)"`) creates the chart, and destroys and recreates it after each HTMX swap.

**Settings**
- `/settings/status`: table of every mapped status with an include/exclude toggle, plus a form to add a new status. Saved via HTMX.
- `/settings/products`: one row per known SKU (from the dataset and config) with an editable display name. Default is the **most frequent `product_name` for that SKU in the current dataset**, with ties broken by the alphabetically first name. Clearing the field reverts to the default.
- `/settings/sku-mapping`: list of product_name→SKU pairs from `config.json`, each with edit and delete. Changes apply to the **next** upload only. The current dataset already has SKUs resolved and isn't re-processed.

**Architecture (Port-and-Adapter)**
- `services/sales/` is the domain: entities, normalization, import/review logic, analytics, and port interfaces `DatasetRepo` and `ConfigRepo`. It has no HTTP or filesystem code.
- `services/sales/repo_file/` is the adapter that implements the ports with JSON files.
- `app/http/controller/sales/` holds chi handlers that parse requests, call the service, and render templ views.
- `app/http/views/` holds the templ components.
- `app/router.go` builds the chi router. `app/http/main.go` wires config → repos → service → controller → router.

**Key Go entities** (in `services/sales/type.go`)
```go
type Channel string

const (
    ChannelShopee    Channel = "Shopee"
    ChannelTikTok    Channel = "TikTok Shop"
    ChannelTokopedia Channel = "Tokopedia"
)

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

type Dataset struct {
    FileName   string      `json:"file_name"`
    UploadedAt time.Time   `json:"uploaded_at"`
    MinDate    time.Time   `json:"min_date"`
    MaxDate    time.Time   `json:"max_date"`
    Lines      []OrderLine `json:"lines"`
}

type Config struct {
    StatusMapping map[string]StatusClass `json:"status_mapping"` // key: lowercased status
    StatusLabels  map[string]string      `json:"status_labels"`  // key: lowercased → original casing for display
    NameToSKU     map[string]string      `json:"name_to_sku"`    // key: exact trimmed product_name
    DisplayNames  map[string]string      `json:"display_names"`  // key: SKU; only user overrides
}

type ParseError struct {
    Line   int    `json:"line"`
    Column string `json:"column"`
    Value  string `json:"value"`
    Reason string `json:"reason"`
}

type WarningCode string

const (
    WarnPriceMismatch         WarningCode = "price_mismatch"
    WarnChannelPrefixMismatch WarningCode = "channel_prefix_mismatch"
)

type RowWarning struct {
    Line    int         `json:"line"`
    OrderID string      `json:"order_id"`
    Channel Channel     `json:"channel"`
    Code    WarningCode `json:"code"`
    Message string      `json:"message"`
}

type DuplicateGroup struct {
    Key   string    `json:"key"`   // sha1 of joined raw cells
    Lines []int     `json:"lines"` // source line numbers
    Line  OrderLine `json:"line"`  // representative normalized line
}

type UnresolvedSKU struct {
    ProductName   string   `json:"product_name"`
    Lines         []int    `json:"lines"`
    CandidateSKUs []string `json:"candidate_skus"` // non-empty when file maps name to >1 SKU
}

type UnknownStatus struct {
    Status string `json:"status"`
    Count  int    `json:"count"`
}

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

type ReviewDecision struct {
    Token          string                 `validate:"required"`
    KeepDuplicates map[string]bool        // DuplicateGroup.Key → keep all copies
    SKUChoices     map[string]string      // ProductName → SKU
    StatusChoices  map[string]StatusClass // Status → include/exclude
}

type Filter struct {
    From     time.Time // inclusive, date only
    To       time.Time // inclusive, date only
    Channels []Channel // empty = none selected → empty result
}

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

type ChannelRow struct { Channel Channel; Orders, Units int; NetRevenue int64; SharePct float64 }
type ProductRow struct { Rank int; SKU, DisplayName string; Units int; NetRevenue int64; SharePct float64 }
type DailyPoint struct { Date time.Time; NetRevenue int64 }
```

**Ports** (in `services/sales/repo.go`)
```go
type DatasetRepo interface {
    Load(ctx context.Context) (*Dataset, error) // returns ErrNoDataset if file absent
    Save(ctx context.Context, ds *Dataset) error
}

type ConfigRepo interface {
    Load(ctx context.Context) (*Config, error) // seeds defaults if file absent
    Save(ctx context.Context, cfg *Config) error
}
```

**Service** (in `services/sales/service.go`)
```go
type Service interface {
    StartImport(ctx context.Context, fileName string, r io.Reader) (*PendingImport, error)
    GetPendingImport(ctx context.Context, token string) (*PendingImport, error)
    ConfirmImport(ctx context.Context, d ReviewDecision) (*Dataset, error)
    CancelImport(ctx context.Context, token string) error

    CurrentDataset(ctx context.Context) (*Dataset, error)
    Summarize(ctx context.Context, f Filter) (*Summary, error)

    GetConfig(ctx context.Context) (*Config, error)
    SetStatusClass(ctx context.Context, status string, class StatusClass) error
    DeleteStatus(ctx context.Context, status string) error
    SetDisplayName(ctx context.Context, sku, name string) error // empty name = revert to default
    ListProducts(ctx context.Context) ([]ProductInfo, error)    // SKU, default name, override
    SetNameMapping(ctx context.Context, productName, sku string) error
    DeleteNameMapping(ctx context.Context, productName string) error
}
```
The service holds the in-memory dataset and the single pending import, guarded by a `sync.RWMutex`.

**Routes** (`app/router.go`)
| Method | Path | Handler | Notes |
|---|---|---|---|
| GET | `/` | `Dashboard` | Full page. Reads filter query params |
| GET | `/dashboard/widgets` | `DashboardWidgets` | HTMX partial (KPIs, tables, charts) |
| GET | `/upload` | `UploadPage` | |
| POST | `/upload` | `Upload` | multipart `file`. On success, HX-Redirect to review |
| GET | `/upload/review` | `ReviewPage` | `?token=` |
| POST | `/upload/confirm` | `Confirm` | form fields: `token`, `keep_dup[<key>]`, `sku[<name>]`, `status[<status>]` |
| POST | `/upload/cancel` | `Cancel` | |
| GET/POST | `/settings/status` | `StatusSettings` / `SaveStatus` | DELETE `/settings/status/{status}` |
| GET/POST | `/settings/products` | `ProductSettings` / `SaveDisplayName` | |
| GET/POST | `/settings/sku-mapping` | `SKUMappingSettings` / `SaveSKUMapping` | DELETE `/settings/sku-mapping` with `product_name` form field |
| GET | `/static/*` | file server | |
| GET | `/healthz` | `200 ok` | for Docker healthcheck |

**Validation rules**
| Field | Rule | Error (shown in UI) |
|---|---|---|
| Upload file | required, extension `.csv`, ≤ `MAX_UPLOAD_MB` | "File wajib diunggah", "File harus berformat .csv", "Ukuran file melebihi batas {N} MB" |
| CSV header | all 10 columns present | "Kolom wajib tidak ditemukan: {kolom}" |
| CSV body | ≥1 data row | "File CSV tidak berisi data" |
| Row: order_id / product_name | non-empty | "{kolom} kosong" |
| Row: order_date | matches one of 3 layouts | "Format tanggal tidak dikenali" |
| Row: channel | in alias map | "Channel tidak dikenali" |
| Row: qty | int ≥ 1 | "Jumlah tidak valid" |
| Row: unit/total price | positive int after cleanup | "Format harga tidak dikenali" |
| Confirm: SKU choices | one per unresolved name, non-empty, ≤64 chars | "Pilih SKU untuk semua produk yang belum dikenali" |
| Confirm: status choices | one per unknown status, `include`/`exclude` | "Tentukan klasifikasi untuk semua status baru" |
| Confirm: valid lines | ≥1 valid line after skipping errors | "Tidak ada baris valid untuk diimpor" |
| Filter from/to | `YYYY-MM-DD`, from ≤ to | "Rentang tanggal tidak valid" |
| Settings: status | non-empty, class in {include, exclude} | "Status wajib diisi" |
| Settings: display name | ≤ 200 chars | "Nama terlalu panjang" |
| Settings: SKU mapping | product_name and SKU non-empty | "Nama produk dan SKU wajib diisi" |

**Error handling** (`services/sales/error.go` → mapped in `app/http/controller/sales/error.go`)
| Domain error | HTTP | UI message |
|---|---|---|
| `ErrNoDataset` | 200 (empty state) | "Belum ada data…" |
| `ErrMissingColumns` (wraps column list) | 422 | "Kolom wajib tidak ditemukan: …" |
| `ErrEmptyCSV` | 422 | "File CSV tidak berisi data" |
| `ErrInvalidCSV` (csv reader failure) | 422 | "File CSV tidak dapat dibaca" |
| `ErrPendingNotFound` (bad/expired token) | 404 | "Sesi impor tidak ditemukan. Silakan unggah ulang." |
| `ErrUnresolvedSKU` | 422 | "Pilih SKU untuk semua produk yang belum dikenali" |
| `ErrUnknownStatus` | 422 | "Tentukan klasifikasi untuk semua status baru" |
| `ErrNoValidLines` | 422 | "Tidak ada baris valid untuk diimpor" |
| `ErrInvalidFilter` | 400 | "Rentang tanggal tidak valid" |
| `ErrInvalidInput` | 400 | field-specific message |
| storage I/O failure (other) | 500 | "Terjadi kesalahan saat menyimpan data" (full error logged via `log/slog`) |

HTMX error responses render an inline alert partial into `#flash`, using `HX-Retarget: #flash`.

### Out of Scope
- **Database / history across uploads.** Decided: replace-all on each upload, so month-over-month comparison isn't possible.
- **Authentication / multiple users.** Decided: single user, no login.
- **Date presets** (Last 7 days, This month…). Decided: removed.
- **Buyer-city analytics and city filter.** Decided: channel is the only filter besides date. `buyer_city` is stored but not shown.
- **Marketplace fees, vouchers, shipping cost.** Not in the CSV. Net revenue = SUM(total_price).
- **Gross revenue, cancel/return rate KPIs.** Decided: net-only KPIs.
- **Re-processing the current dataset when the SKU mapping changes.** Only the next upload uses it.
- **Export (CSV/PDF) of insights.**
- **Auto-correcting** price or channel mismatches. They are only flagged as warnings.
- **English UI / i18n switching.**

---

## 6. Constraints & Dependencies

- Go ≥ 1.22. Module path `github.com/azisuazusa/biota`.
- Libraries: `github.com/go-chi/chi/v5`, `github.com/a-h/templ`, `github.com/go-playground/validator/v10`, `github.com/stretchr/testify`, `github.com/vektra/mockery/v2` (dev tool). Standard library `encoding/csv`, `log/slog`.
- Frontend: Tailwind CSS standalone CLI (no Node in the runtime image), HTMX 2.x, Alpine.js 3.x, Chart.js 4.x, all vendored under `static/vendor/`.
- Environment variables: `PORT` (default `8080`), `DATA_DIR` (default `/data`), `MAX_UPLOAD_MB` (default `10`, confirmed).
- Docker: multi-stage build (templ generate + tailwind build + `go build`) → distroless or alpine runtime. `VOLUME /data`, `EXPOSE 8080`, healthcheck on `/healthz`.
- Layering: `services/sales` must not import `net/http`, chi, templ, or `os` file APIs. Filesystem access lives only in `repo_file`.
- Error wrapping convention: `fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)` with helper `pkg/util/caller.go`.
- Money is held as `int64` rupiah everywhere. Floats are used only for `SharePct`.
- Sample file copied to `test_data/sample_orders_oct2026.csv` as the main fixture.

---

## 7. Related Docs

| Document | Purpose |
|----------|---------|
| `docs/prd-format.md` | Format this PRD follows |
| `docs/task-progress-format.md` | Format of the task list |
| `docs/2026-11-23-marketplace-sales-insight/task-progress.md` | Implementation plan by phase |
| `test_data/sample_orders_oct2026.csv` | Sample export; source of the acceptance numbers in §4 |

---

## 8. Open Questions

None at this time. Resolved on 2026-11-23:
- Module path: `github.com/azisuazusa/biota`.
- Max upload size: 10 MB default.
- `qty × unit_price ≠ total_price`: shown as a non-blocking review warning.
- Channel vs order_id prefix mismatch: shown as a non-blocking review warning, and the channel column is used.
