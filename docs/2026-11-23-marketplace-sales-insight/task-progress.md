# BIOTA Marketplace Sales Insight — TODO

Reference: `docs/2026-11-23-marketplace-sales-insight/prd.md`

A Go + chi + templ + HTMX/Alpine/Tailwind/Chart.js app, shipped in Docker. It imports a marketplace order CSV, normalizes it (with a review step), and shows net revenue per channel and product, the top 10 products, and a daily trend, filtered by date and channel. Storage is JSON files on a volume, with no database.

> Greenfield project: no existing code to copy patterns from. The conventions below are set by this plan.
> Error wrapping everywhere: `fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)`.

---

## Phase 0 — Project scaffolding ✅ DONE

- [x] `go.mod`: `go mod init github.com/azisuazusa/biota` (Go ≥ 1.22). Add deps: `github.com/go-chi/chi/v5`, `github.com/a-h/templ`, `github.com/go-playground/validator/v10`, `github.com/stretchr/testify` — _Note: go directive is `go 1.26.0` (validator v10.30 and golang.org/x/* require Go ≥ 1.26); still satisfies Go ≥ 1.22_
- [x] `pkg/util/caller.go`: `func GetPackageAndFunctionName() string` using `runtime.Caller(1)` + `runtime.FuncForPC`. Returns e.g. `sales.(*service).StartImport`
- [x] `pkg/util/caller_test.go`: assert the returned name contains the calling function name
- [x] `test_data/sample_orders_oct2026.csv`: **synthetic** fixture (495 rows) — _Note (2026-11-23): originally a copy of the real October export; replaced before publishing the public repo with generated data that has the same structure (495 rows, 22 exact-duplicate pairs, 76 empty SKUs, 12 SKUs × 3 listing names, all channel spellings, all 3 date and ~6 price formats, 0 warnings). The real export stays local only. Acceptance numbers in PRD §4 and all tests were updated to the synthetic file._
- [x] `test_data/missing_column.csv`: header without `status`, 2 rows
- [x] `test_data/header_only.csv`: valid header, no rows
- [x] `test_data/edge_cases.csv`: rows covering bad date (`2026/13/45`), unknown channel (`Lazada`), qty `0`, qty `abc`, price `Rp-`, empty order_id, empty SKU with a name not in the file, a name mapped to 2 SKUs in the same file, unknown status `Menunggu Pembayaran`, a BOM at file start, mixed-case header, a row with qty `2` × `Rp100.000` but total `Rp150.000` (price warning), a row with channel `Shopee` and order_id `TKP123` (prefix warning), a row with order_id `XYZ1` (prefix warning)
- [x] `.mockery.yaml`: generate mocks for `github.com/azisuazusa/biota/services/sales` interfaces `DatasetRepo`, `ConfigRepo` into `services/sales/mocks/` — _Note: the installed mockery is v3.7.4, so the config uses the v3 format (`template: testify`); generated mocks are API-compatible with v2 testify mocks_
- [x] `Makefile` targets:
  - [x] `generate`: `templ generate && ./bin/tailwindcss -i static/css/input.css -o static/css/app.css --minify`
  - [x] `test`: `go test -count=1 -failfast ./...`
  - [x] `run-lint`: `golangci-lint run ./...`
  - [x] `run`: `make generate && DATA_DIR=./.data go run ./app/http`
  - [x] `mocks`: `mockery`
  - [x] `docker-build`: `docker build -t biota-insight .`
  - [x] `tailwind-install`: download Tailwind standalone CLI for the host OS/arch into `bin/tailwindcss`
- [x] `static/vendor/`: vendor `htmx.min.js` (2.x), `alpine.min.js` (3.x), `chart.umd.min.js` (4.x). Record the versions in `static/vendor/VERSIONS.txt` — _Note: htmx 2.0.11, Alpine 3.17.5, Chart.js 4.6.1_
- [x] `static/css/input.css`: `@import "tailwindcss";` (Tailwind v4) with `@source "../../app/http/views/**/*.templ";`
- [x] `.gitignore`: `bin/`, `.data/`, `*_templ.go`, `static/css/app.css` — _Note: because `*_templ.go` is ignored, a fresh checkout needs `make generate` before `make test`/`go build` (documented in README)_

---

## Phase 1 — Data model (no database; replaces migrations) ✅ DONE

- [x] `services/sales/type.go`: define the entities exactly as in PRD §5 "Key Go entities": `Channel` + 3 constants, `StatusClass` + 2 constants, `OrderLine`, `Dataset`, `Config`, `ParseError`, `DuplicateGroup`, `UnresolvedSKU`, `UnknownStatus`, `PendingImport`, `ReviewDecision`, `Filter`, `Summary`, `ChannelRow`, `ProductRow`, `DailyPoint`, plus `ProductInfo { SKU, DefaultName, Override string }`
- [x] `services/sales/type.go`: `var AllChannels = []Channel{ChannelShopee, ChannelTikTok, ChannelTokopedia}` (display order)
- [x] `services/sales/type.go`: `var RequiredColumns = []string{"order_id","order_date","channel","sku","product_name","qty","unit_price","total_price","status","buyer_city"}`
- [x] `services/sales/defaults.go`: `func DefaultConfig() *Config` seeding `StatusMapping` (lowercased keys) + `StatusLabels`:
  - include: `Selesai`, `Completed`, `Pesanan Selesai`, `Dikirim`, `In transit`, `Sedang Dikirim`
  - exclude: `Batal`, `Cancelled`, `Dibatalkan`, `Pengembalian Dana`, `Refunded`, `Dikomplain - Retur`
  - `NameToSKU`, `DisplayNames` = empty non-nil maps
- [x] `services/sales/config.go`: methods on `*Config`:
  - [x] `Classify(status string) (StatusClass, bool)`: lookup by `strings.ToLower(strings.TrimSpace(status))`
  - [x] `SetStatus(status string, c StatusClass)`: writes both maps
  - [x] `Clone() *Config`: deep copy, so callers never mutate shared state
- [x] `services/sales/error.go`: `var ErrNoDataset, ErrMissingColumns, ErrEmptyCSV, ErrInvalidCSV, ErrPendingNotFound, ErrUnresolvedSKU, ErrUnknownStatus, ErrNoValidLines, ErrInvalidFilter, ErrInvalidInput = errors.New(...)`. Add `type MissingColumnsError struct{ Columns []string }` with `Unwrap() error { return ErrMissingColumns }`
- [x] `services/sales/config_test.go`: Classify is case-insensitive and trims spaces (`" selesai "` → include), returns false for unknown, Clone gives independent maps

---

## Phase 2 — Repository (file adapters) ✅ DONE

- [x] `services/sales/repo.go`: port interfaces:
  ```go
  type DatasetRepo interface {
      Load(ctx context.Context) (*Dataset, error) // ErrNoDataset if file absent
      Save(ctx context.Context, ds *Dataset) error
  }
  type ConfigRepo interface {
      Load(ctx context.Context) (*Config, error) // DefaultConfig() + Save if absent
      Save(ctx context.Context, cfg *Config) error
  }
  ```
- [x] `services/sales/repo_file/atomic.go`: `func writeJSONAtomic(path string, v any) error`: `os.CreateTemp(filepath.Dir(path), ".tmp-*")`, `json.NewEncoder` with indent, `f.Sync()`, `f.Close()`, `os.Rename(tmp, path)`, and remove tmp on any error
- [x] `services/sales/repo_file/dataset_repo.go`: `type datasetRepo struct{ path string }`, `func NewDatasetRepo(dataDir string) sales.DatasetRepo` (path = `dataDir/dataset.json`):
  - [x] `Load`: `os.ReadFile`. On `errors.Is(err, fs.ErrNotExist)` return `sales.ErrNoDataset`. Otherwise `json.Unmarshal`, wrap errors
  - [x] `Save`: `writeJSONAtomic`
- [x] `services/sales/repo_file/config_repo.go`: `type configRepo struct{ path string }`, `func NewConfigRepo(dataDir string) sales.ConfigRepo` (path = `dataDir/config.json`):
  - [x] `Load`: if absent → `cfg := sales.DefaultConfig()`, `Save(cfg)`, return it. Initialise nil maps after unmarshal
  - [x] `Save`: `writeJSONAtomic`
- [x] `services/sales/repo_file/repo_test.go`: `testify/suite` using `t.TempDir()`:
  - [x] Dataset Load on empty dir → `ErrNoDataset`
  - [x] Dataset Save → Load round-trip keeps every field, including `time.Time` and `HasTime`
  - [x] Config Load on empty dir creates `config.json` with the 12 default statuses
  - [x] Config Save → Load round-trip
  - [x] Atomic write leaves no `.tmp-*` files behind; overwriting an existing file works
  - [x] Load of corrupt JSON returns a wrapped error (not `ErrNoDataset`)
- [x] Run: `go test -count=1 -failfast ./services/sales/repo_file/...`
- [x] `make mocks`: generates `services/sales/mocks/DatasetRepo.go`, `ConfigRepo.go`

---

## Phase 3 — Business logic ✅ DONE

### 3a. Normalizers (pure functions)
- [x] `services/sales/normalize.go`:
  - [x] `func NormalizeChannel(raw string) (Channel, error)`: lowercase+trim, alias map `shopee`→Shopee, `tiktok shop`/`tiktok`→TikTok Shop, `tokopedia`/`tokped`→Tokopedia, else error "Channel tidak dikenali"
  - [x] `func ParseOrderDate(raw string) (t time.Time, hasTime bool, err error)`: try `time.ParseInLocation` with `time.UTC` for layouts `"2006-01-02 15:04"` (true), `"02/01/2006 15:04:05"` (true), `"Jan 02, 2006"` (false), in that order
  - [x] `func ParseIDR(raw string) (int64, error)`: strip case-insensitive `rp` / `idr` prefix, spaces, `.`, `,`; `strconv.ParseInt`; reject ≤0
  - [x] `func ParseQty(raw string) (int, error)`: `strconv.Atoi`, reject <1
  - [x] `func DateOnly(t time.Time) time.Time`: truncate to 00:00 UTC
- [x] `services/sales/warnings.go`:
  - [x] `var channelPrefix = map[Channel]string{ChannelShopee: "SHP", ChannelTikTok: "TTS", ChannelTokopedia: "TKP"}`
  - [x] `func checkRowWarnings(l OrderLine) []RowWarning`: `price_mismatch` when `int64(l.Qty)*l.UnitPrice != l.TotalPrice` (message uses `FormatIDR`-equivalent `formatRp` helper in the same package, so `services/sales` doesn't import views); `channel_prefix_mismatch` when `!strings.HasPrefix(strings.ToUpper(l.OrderID), channelPrefix[l.Channel])` (message shows the first 3 chars of order_id and the channel)
- [x] `services/sales/warnings_test.go`: matching row → no warnings; 2×100000≠150000 → price warning with the exact message; `Shopee`+`TKP123` → prefix warning; `XYZ1` → prefix warning; a row with both problems → 2 warnings
- [x] `services/sales/normalize_test.go`: table tests:
  - [x] Channel: all 10 sample spellings map correctly; `Lazada` errors
  - [x] Date: `2026-10-21 08:24`, `14/10/2026 23:26:00` (→ day=14, month=10), `Oct 01, 2026` (hasTime=false); `2026/13/45` errors
  - [x] IDR: `Rp1.099.000`, `Rp 1.099.000`, `1,099,000`, `IDR 1,099,000`, `IDR 2,199,000`, `1099000`, `Rp 2.198.000` → expected ints; `Rp-`, `""`, `abc` error
  - [x] Qty: `1`,`3` ok; `0`,`-1`,`abc` error

### 3b. CSV parsing
- [x] `services/sales/parser.go`: `func parseCSV(r io.Reader) (rawRows [][]string, colIdx map[string]int, err error)`: — _Note: returns `[]rawRow{Line, Cells}` instead of `[][]string` so each record keeps its real file line (via `csv.Reader.FieldPos`), which stays correct even with multi-line quoted cells_
  - [x] Wrap reader to strip a UTF-8 BOM; `csv.NewReader` with `FieldsPerRecord = -1`, `TrimLeadingSpace = true`
  - [x] Header map built from lowercase+trimmed names; missing required → `&MissingColumnsError{Columns}`
  - [x] Zero data rows → `ErrEmptyCSV`; reader error → wrap `ErrInvalidCSV`
- [x] `services/sales/parser.go`: `func normalizeRow(line int, raw []string, idx map[string]int) (OrderLine, []ParseError)`: trims every cell, runs the 3a normalizers, checks non-empty `order_id`/`product_name`, keeps raw `sku` (may be empty), raw `status`, `buyer_city`. Collects **all** errors for the row (column, raw value, reason). Line number = 1-based file line (header = 1)
- [x] `services/sales/parser_test.go`: sample file → 495 raw rows, 0 errors; `missing_column.csv` → `MissingColumnsError{["status"]}`; `header_only.csv` → `ErrEmptyCSV`; `edge_cases.csv` → expected error lines/columns; BOM + mixed-case header accepted

### 3c. Import / review building
- [x] `services/sales/importer.go`: `func buildPendingImport(fileName string, r io.Reader, cfg *Config) (*PendingImport, error)`:
  - [x] Parse + normalize. Valid rows → `Lines`, invalid → `ParseErrors` (row skipped)
  - [x] **Warnings**: run `checkRowWarnings` on every valid line → `Warnings` (sorted by line). They never block and never change lines
  - [x] **Duplicates**: key = `sha1(strings.Join(trimmedRawCells, "\x1f"))` over valid rows; groups with ≥2 lines → `Duplicates` (lines sorted, representative = first)
  - [x] **Learned mapping**: from lines with non-empty SKU build `name → set(SKU)`. Names with exactly 1 SKU → `LearnedMapping`
  - [x] **SKU inference** for empty-SKU lines: learned (single) → `cfg.NameToSKU` → else unresolved. A name with >1 SKU in the file is always unresolved with `CandidateSKUs` sorted. Set `SKUInferred=true` when filled
  - [x] Group unresolved by product name → `Unresolved` (sorted by name)
  - [x] **Unknown statuses**: distinct statuses (case-insensitive) where `cfg.Classify` is false → `UnknownStatus{Status, Count}` (sorted by count desc)
  - [x] `KnownSKUs`: union of SKUs in lines + `cfg.NameToSKU` values + `cfg.DisplayNames` keys, sorted
  - [x] `Token`: 16 random bytes hex (`crypto/rand`)
- [x] `services/sales/importer.go`: `func applyDecision(p *PendingImport, d ReviewDecision, cfg *Config) (*Dataset, *Config, error)`:
  - [x] Every `Unresolved.ProductName` has non-empty trimmed `d.SKUChoices[name]` (≤64 chars) else `ErrUnresolvedSKU`
  - [x] Every `UnknownStatus.Status` has `d.StatusChoices[status]` ∈ {include, exclude} else `ErrUnknownStatus`
  - [x] Duplicates: for groups where `KeepDuplicates[key]` is false, drop all lines except the first; keep all when true
  - [x] Fill SKUs of unresolved lines from choices (`SKUInferred=true`)
  - [x] 0 lines left → `ErrNoValidLines`
  - [x] New config = `cfg.Clone()` + merge `LearnedMapping` + SKU choices into `NameToSKU` + status choices via `SetStatus`
  - [x] Dataset: `FileName`, `UploadedAt=time.Now()`, `MinDate`/`MaxDate` = min/max `DateOnly(OrderDate)`, lines sorted by `OrderDate`, `SourceLine`
- [x] `services/sales/importer_test.go` (sample fixture + default config):
  - [x] 495 rows → 0 parse errors, 0 warnings, 22 duplicate groups, 0 unresolved, 0 unknown statuses, `LearnedMapping` has exactly 36 names (12 SKUs × 3 listing names)
  - [x] All 76 empty-SKU lines get a SKU with `SKUInferred=true`
  - [x] Default decision (keep none) → 473 lines; keep all → 495 lines
  - [x] `edge_cases.csv`: unresolved name prompts; name mapped to 2 SKUs → unresolved with 2 candidates; `Menunggu Pembayaran` listed as unknown; 3 warnings (1 price, 2 prefix) and those rows still present in `Lines` and in the confirmed dataset with their original `total_price` and channel
  - [x] Confirm missing SKU choice → `ErrUnresolvedSKU`; missing status choice → `ErrUnknownStatus`; all rows invalid → `ErrNoValidLines`
  - [x] Config merge: manual SKU choice + learned pairs + new status present in returned config; input cfg untouched
  - [x] Config mapping used: with `cfg.NameToSKU["Foo"]="BIO-X"`, an empty-SKU "Foo" row resolves without a prompt

### 3d. Analytics
- [x] `services/sales/analytics.go`: `func summarize(ds *Dataset, cfg *Config, f Filter) *Summary`:
  - [x] Line passes if `DateOnly(OrderDate)` ∈ [From, To], `Channel` ∈ `f.Channels`, and `cfg.Classify(Status)` == include
  - [x] NetRevenue = Σ TotalPrice; Orders = distinct OrderID; Units = Σ Qty; AOV = `int64(math.Round(float64(net)/float64(orders)))`, 0 when orders=0
  - [x] ByChannel: one row per channel in `f.Channels` (in `AllChannels` order, zeros included); per-channel distinct orders; SharePct = rev/net*100 (0 if net=0)
  - [x] ByProduct: group by SKU; DisplayName = `displayName(sku)`; sort rev desc, SKU asc; Rank 1..n; Top10 = first ≤10
  - [x] Daily: one point for every day From..To inclusive, 0 where no sales
- [x] `services/sales/analytics.go`: `func defaultDisplayNames(ds *Dataset) map[string]string`: most frequent ProductName per SKU, tie → alphabetically first. `func resolveDisplayName(sku string, cfg *Config, defaults map[string]string) string`: override → default → SKU
- [x] `services/sales/analytics_test.go`: on the sample with duplicates dropped, full range, all channels, assert PRD §4 values exactly:
  - [x] Net 360363000, Orders 353, Units 517, AOV 1020858
  - [x] Shopee 228176000, TikTok Shop 87666000, Tokopedia 44521000
  - [x] ByProduct has 12 rows, in the order and with the values listed in PRD §4; Top10 len 10, first = BIO-CAM-BBY 67548000 / 52 units
  - [x] Daily has 31 points (1–31 Oct 2026), Σ = net
  - [x] Channel filter `[Tokopedia]` → net 44521000, ByChannel has 1 row
  - [x] Empty channel list → all zeros, empty ByProduct
  - [x] Single-day range works (From == To); Tokopedia date-only rows counted on their date
  - [x] Moving `Dikirim` to exclude in config lowers net (classification at query time)
  - [x] Display-name default and override; tie-break alphabetical

### 3e. Service
- [x] `services/sales/service.go`: `type service struct{ mu sync.RWMutex; dsRepo DatasetRepo; cfgRepo ConfigRepo; ds *Dataset; cfg *Config; pending *PendingImport; validate *validator.Validate }`, `func New(ctx context.Context, ds DatasetRepo, cfg ConfigRepo) (Service, error)` loads config (seeding defaults) and dataset (`ErrNoDataset` → nil dataset, not an error)
- [x] `services/sales/service.go`: implement the `Service` interface from PRD §5:
  - [x] `StartImport`: build pending with a cfg snapshot, replacing any existing pending
  - [x] `GetPendingImport`: token mismatch or no pending → `ErrPendingNotFound`
  - [x] `ConfirmImport`: validate struct; token check; `applyDecision`; `cfgRepo.Save(newCfg)` then `dsRepo.Save(newDs)`; on success swap in-memory `ds`/`cfg`, clear `pending`. If a save fails, return the wrapped error and leave the in-memory state unchanged — _Note: a validator failure (empty token) is reported as `ErrPendingNotFound` (404 "Sesi impor tidak ditemukan…")_
  - [x] `CancelImport`: clear pending if the token matches
  - [x] `CurrentDataset`: nil → `ErrNoDataset`
  - [x] `Summarize`: `ErrNoDataset` if none; `From > To` → `ErrInvalidFilter`; RLock
  - [x] `SetStatusClass` / `DeleteStatus` / `SetDisplayName` / `SetNameMapping` / `DeleteNameMapping`: validate (rules in PRD §5 table, `ErrInvalidInput`), mutate a clone, `cfgRepo.Save`, swap on success
  - [x] `ListProducts`: SKUs from dataset ∪ config `DisplayNames` keys ∪ `NameToSKU` values; DefaultName from `defaultDisplayNames`
- [x] `services/sales/service_test.go`: `testify/suite` with `mocks.DatasetRepo` / `mocks.ConfigRepo`:
  - [x] `New` with `ErrNoDataset` → ok, `CurrentDataset` returns `ErrNoDataset`
  - [x] Start → Confirm happy path calls `ConfigRepo.Save` then `DatasetRepo.Save` once each (`AssertExpectations`)
  - [x] Confirm with wrong token → `ErrPendingNotFound`, no Save calls
  - [x] `DatasetRepo.Save` error → error returned, previous dataset still served, pending kept
  - [x] A second StartImport invalidates the first token
  - [x] Cancel clears pending; dataset unchanged
  - [x] Summarize From>To → `ErrInvalidFilter`
  - [x] `SetDisplayName("BIO-CAM-BF","")` removes the override; a 201-char name → `ErrInvalidInput`
- [x] Run: `go test -count=1 -failfast ./services/sales/...`

---

## Phase 4 — HTTP controller ✅ DONE

- [x] `app/http/controller/sales/type.go`: `type DashboardQuery struct{ From, To string; Channels []string }` + `func parseDashboardQuery(r *http.Request, ds *sales.Dataset) (sales.Filter, string /*errMsg*/)`: empty from/to → dataset Min/MaxDate; no `channel` params on the first load (no `filtered=1` flag) → all channels; with `filtered=1` and no channel → empty list; invalid date → error "Rentang tanggal tidak valid" — _Note: also rejects dates outside the dataset's min/max (PRD §5 Filters: "outside the dataset's dates show an inline error")_
- [x] `app/http/controller/sales/type.go`: `func parseReviewDecision(r *http.Request) sales.ReviewDecision`: reads `token`, `keep_dup[<key>]=on`, `sku[<name>]`, `status[<status>]`. Use `r.PostForm` key prefix parsing; keys are URL-encoded by the browser
- [x] `app/http/controller/sales/controller.go`: `type Controller struct{ svc sales.Service; maxUploadBytes int64 }`, `func New(svc sales.Service, maxUploadMB int) *Controller`. Handlers (signature `func (c *Controller) X(w http.ResponseWriter, r *http.Request)`):
  - [x] `Dashboard`: `ErrNoDataset` → render empty state; else full page with filter + widgets — _Note: invalid filter renders the page with an inline error and HTTP 400_
  - [x] `DashboardWidgets`: HTMX partial only — _Note: sets `HX-Push-Url: /?<query>` so the pushed URL is the bookmarkable full page, not the partial; invalid filter → inline error in the widget area (400) rather than #flash, matching "inline error" in PRD §5_
  - [x] `UploadPage`
  - [x] `Upload`: `http.MaxBytesReader`; `r.ParseMultipartForm`; `FormFile("file")`; check `.csv` extension (case-insensitive); `svc.StartImport`; success → `w.Header().Set("HX-Redirect", "/upload/review?token="+tok)`
  - [x] `ReviewPage`
  - [x] `Confirm`: on `ErrUnresolvedSKU`/`ErrUnknownStatus` re-render review with an inline error (422); success → `HX-Redirect: /`
  - [x] `Cancel` → `HX-Redirect: /upload`
  - [x] `StatusSettings`, `SaveStatus`, `DeleteStatus` (chi `URLParam("status")`, URL-decoded)
  - [x] `ProductSettings`, `SaveDisplayName`
  - [x] `SKUMappingSettings`, `SaveSKUMapping`, `DeleteSKUMapping`
  - [x] Render helper: `func render(w http.ResponseWriter, r *http.Request, status int, c templ.Component)`
- [x] `app/http/controller/sales/error.go`: `func (c *Controller) handleError(w http.ResponseWriter, r *http.Request, err error)`: map domain errors to HTTP code + Indonesian message per the PRD §5 error table (`errors.Is`/`errors.As` for `MissingColumnsError`). Set `HX-Retarget: #flash`, `HX-Reswap: innerHTML`, render `components.Flash(msg)`; log 5xx with `slog.Error` — _Note: non-HTMX requests get a full `views.ErrorPage` instead of a bare flash partial. Field-specific `ErrInvalidInput` messages travel in `sales.InputError{Message}` (unwraps to `ErrInvalidInput`)_
- [x] `app/http/controller/sales/controller_test.go`: `httptest` + real service on `t.TempDir()` repos:
  - [x] POST `/upload` with sample → 200 + `HX-Redirect` containing a token
  - [x] POST `/upload` with `.txt` → 422 "File harus berformat .csv"
  - [x] POST `/upload` over the size limit → 413/422 with the size message
  - [x] `missing_column.csv` → 422 message lists `status`
  - [x] Confirm → GET `/dashboard/widgets` body contains `Rp360.363.000`
  - [x] GET `/dashboard/widgets?from=2026-10-10&to=2026-10-01` → 400 message
  - [x] GET `/` with no dataset → empty-state text

---

## Phase 5 — UI (templ views) ✅ DONE

- [x] `app/http/views/format.go`: `func FormatIDR(v int64) string` → `Rp1.099.000` (handles 0 → `Rp0`); `func FormatDateID(t time.Time) string` → `21 Okt 2026` (month names Jan Feb Mar Apr Mei Jun Jul Agu Sep Okt Nov Des); `func FormatPct(f float64) string` → `60,3%`. Tests in `app/http/views/format_test.go`
- [x] `app/http/views/layout.templ`: `templ Layout(title string, active string)`: `<html lang="id">`, `static/css/app.css`, vendored htmx/alpine/chart scripts (`defer`), nav links Dashboard / Unggah Data / Pengaturan (Status, Produk, Pemetaan SKU), `<div id="flash">` — _Note: adds `<meta name="htmx-config">` with `responseHandling` so HTMX swaps 4xx/5xx responses (HTMX 2 does not by default; otherwise #flash and inline errors would never show)_
- [x] `app/http/views/components/flash.templ`: `templ Flash(msg string)`: dismissible red alert (Alpine `x-data="{open:true}"`)
- [x] `app/http/views/components/kpi.templ`: `templ KPICard(label, value string)`
- [x] `app/http/views/components/chart.templ`: `templ Chart(id, kind string, data any)`: `<canvas>` + `<script type="application/json" id="{id}-data">` (via `templ.JSONString`) + `x-data="chart('{id}','{kind}')"` — _Note: uses `templ.JSONScript(id+"-data", data)` (renders the same `<script type="application/json">` block)_
- [x] `static/js/charts.js`: `Alpine.data('chart', (id, kind) => ({ init(){…}, destroy(){ this.c?.destroy() } }))`. Kinds: `bar` (channel), `hbar` (top 10, `indexAxis:'y'`), `line` (daily). Tooltip/axis ticks formatted as `Rp` via `Intl.NumberFormat('id-ID')`. Loaded before Alpine
- [x] `app/http/views/dashboard.templ`:
  - [x] `templ DashboardPage(ds *sales.Dataset, f sales.Filter, s *sales.Summary, errMsg string)`: header (file name, `Diunggah` time, `Periode` min–max) + filter form + `<div id="widgets">`
  - [x] Filter form: `hx-get="/dashboard/widgets" hx-target="#widgets" hx-trigger="change" hx-push-url="true"`, hidden `filtered=1`, `<input type="date" name="from|to" min max>`, 3 channel checkboxes `name="channel"`
  - [x] `templ DashboardEmpty()`: "Belum ada data. Unggah file CSV terlebih dahulu." + link
  - [x] `templ Widgets(s *sales.Summary, errMsg string)`: 4 KPI cards (Pendapatan Bersih, Pesanan, Unit Terjual, Rata-rata Nilai Pesanan); "Pendapatan per Channel" table + bar chart; "Top 10 Produk" hbar chart; "Pendapatan per Produk" table (Rank, SKU, Nama Produk, Unit, Pendapatan Bersih, %); "Tren Pendapatan Harian" line chart; "Tidak ada data untuk filter ini" when Orders=0
- [x] `app/http/views/upload.templ`: `templ UploadPage(maxMB int)`: form `hx-post="/upload" hx-encoding="multipart/form-data"`, `accept=".csv"`, loading indicator (`hx-indicator`), note "Unggahan baru akan menggantikan seluruh data sebelumnya."
- [x] `app/http/views/review.templ`: `templ ReviewPage(p *sales.PendingImport, errMsg string)`:
  - [x] Summary counts: total rows, valid, parse errors, warnings, duplicate groups, unresolved SKUs, unknown statuses
  - [x] "Baris gagal dibaca" table (Baris, Kolom, Nilai, Alasan) + note "akan dilewati"
  - [x] "Peringatan" amber table (Baris, Order ID, Channel, Pesan) + note "Baris tetap diimpor"; no inputs
  - [x] "Duplikat" table: line numbers, order_id, product, qty, total, status; checkbox `keep_dup[{key}]` unchecked, label "Simpan duplikat"
  - [x] "SKU belum dikenali": per name a `<input list="known-skus" name="sku[{name}]" required>` + `<datalist>` of `KnownSKUs` (candidates first) — _Note: one `<datalist id="known-skus-{i}">` per name, so each name's candidate SKUs can be listed first. Confirm button is disabled by Alpine until the form is valid; server validation still applies_
  - [x] "Status baru": radio `status[{s}]` include ("Masuk pendapatan") / exclude ("Tidak dihitung"), required
  - [x] Buttons: "Konfirmasi impor" (`hx-post="/upload/confirm"`), "Batal" (`hx-post="/upload/cancel"`) — _Note: on validation errors the review is re-rendered via `views.ReviewBody(p, &decision, msg)` keeping the user's answers_
- [x] `app/http/views/settings_status.templ`: table Status / Klasifikasi (toggle, `hx-post` on change) / Hapus (`hx-delete`, with `hx-confirm` **not** used, because browser dialogs are avoided; use an Alpine inline confirm instead); add form
- [x] `app/http/views/settings_products.templ`: rows SKU / Nama default / Nama tampilan input (`hx-post` on `change`, placeholder = default)
- [x] `app/http/views/settings_sku_mapping.templ`: rows Nama produk / SKU (editable) / Hapus; add form; note "Berlaku untuk unggahan berikutnya"
- [x] Run: `make generate && go build ./...`

---

## Phase 6 — Wiring & packaging ✅ DONE

- [x] `app/router.go`: `func NewRouter(c *salesctrl.Controller) http.Handler` using chi: middleware `middleware.RequestID`, `middleware.RealIP`, `middleware.Logger`, `middleware.Recoverer`; register every route from the PRD §5 routes table; `r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))`; `/healthz` — _Note: `middleware.RealIP` dropped: chi v5.5.0 deprecates it for IP spoofing (GHSA-3fxj-6jh8-hvhx) and golangci-lint fails on it; it would only have affected logged client IPs_
- [x] `app/http/main.go`: read env `PORT` (8080), `DATA_DIR` (/data), `MAX_UPLOAD_MB` (10); `os.MkdirAll(DATA_DIR, 0o755)`; `repo_file.NewDatasetRepo`, `repo_file.NewConfigRepo`; `sales.New(ctx, …)`; `salesctrl.New(svc, maxMB)`; `app.NewRouter`; `http.Server` with timeouts (Read 30s, Write 60s); graceful shutdown on SIGINT/SIGTERM; `slog` JSON handler
- [x] `Dockerfile`: multi-stage
  - [x] stage `build` (`golang:1.23-alpine`): install `templ` (`go install github.com/a-h/templ/cmd/templ@<pinned>`), download the Tailwind standalone CLI (pinned version, matching arch), `templ generate`, tailwind build `--minify`, `CGO_ENABLED=0 go build -o /out/server ./app/http` — _Note: uses `golang:1.26-alpine` because go.mod needs Go ≥ 1.26 (see Phase 0 go.mod note); Tailwind `linux-<arch>-musl` binary, templ v0.3.977 and Tailwind v4.3.5 pinned_
  - [x] stage runtime (`gcr.io/distroless/static-debian12:nonroot`): copy `/out/server` and `static/`; `ENV DATA_DIR=/data PORT=8080`; `VOLUME /data`; `EXPOSE 8080`; `USER nonroot`; `ENTRYPOINT ["/server"]` — _Note: also copies an empty `/data` owned by nonroot so a fresh named volume is writable, and adds a Dockerfile `HEALTHCHECK` using `/server -healthcheck`_
- [x] `docker-compose.yml`: service `app`, port `8080:8080`, named volume `biota-data:/data`, healthcheck (`/server -healthcheck` flag that GETs `/healthz`, because distroless has no curl)
- [x] `app/http/main.go`: `-healthcheck` flag → HTTP GET `http://127.0.0.1:$PORT/healthz`, exit 0/1
- [x] `README.md`: how to run (`docker compose up -d`, then open http://localhost:8080), env vars, data volume, `make run` for local dev

---

## Phase 7 — QA

- [x] `make test` passes
- [x] `make run-lint` passes — _Note: golangci-lint v2.13.1, default linters, 0 issues_
- [x] `gofmt -l .` prints nothing
- [ ] `make docker-build && docker compose up -d`: `/healthz` returns 200 — _Note: not verified: the Docker CLI is installed but the Docker daemon was not running in this environment ("Cannot connect to the Docker daemon"). Needs a manual `make docker-build && docker compose up -d && curl localhost:8080/healthz`_
- [x] Fresh volume: dashboard shows the empty state; `config.json` is created with 12 statuses — _Note: verified with the local binary on an empty `DATA_DIR=./.data` (not a Docker volume)_
- [x] Upload the sample CSV → review shows 495 rows, 0 parse errors, 0 warnings, 22 duplicate groups (all unchecked), 0 unresolved SKU, 0 new status — _Note: verified via curl against the running app: 495 total / 495 valid / 0 failed / 0 warnings / 22 groups (22 checkboxes, none checked) / 0 / 0_
- [x] Confirm with defaults → dashboard full range 1–31 Okt 2026: Pendapatan Bersih **Rp360.363.000**, Pesanan **353**, Unit **517**, AOV **Rp1.020.858** — _Note: verified via curl on `/dashboard/widgets`_
- [x] Channel table: Shopee Rp228.176.000 · TikTok Shop Rp87.666.000 · Tokopedia Rp44.521.000; the bar chart matches — _Note: table values and the bar chart's JSON data payload verified via curl; the rendered chart itself needs a manual browser check_
- [x] Product table: 12 rows in the PRD §4 order; Top 10 chart shows 10 bars, first BIO-CAM-BBY Rp67.548.000 — _Note: 12 table rows and a 10-value Top 10 chart payload (first 67.548.000) verified via curl; rendered chart needs a manual browser check_
- [x] Daily trend: 31 points, including zero days if any — _Note: 31-point payload summing to 360.363.000 verified via curl; rendered chart needs a manual browser check_
- [ ] Uncheck TikTok Shop + Tokopedia → every widget shows Shopee only (net Rp228.176.000); URL updates; reloading the URL keeps the filter — _Note: server side verified (Shopee-only widgets = Rp228.176.000, `HX-Push-Url: /?filtered=1&…&channel=Shopee`, reloading that URL keeps only Shopee checked). The in-browser checkbox → HTMX swap → URL bar update needs a manual browser check_
- [x] Uncheck all channels → "Tidak ada data untuk filter ini" — _Note: verified via curl (`filtered=1` with no channel)_
- [x] Set from > to → inline "Rentang tanggal tidak valid" — _Note: verified via curl: 400 with the inline message in the widget area_
- [x] Re-upload the sample, tick "Simpan duplikat" on all groups → net Rp375.865.000, units 545 — _Note: verified via curl posting all 22 `keep_dup[…]=on`: Rp375.865.000, 353 orders, 545 units_
- [x] Upload `edge_cases.csv` → parse errors listed with line/column/reason; unresolved SKU picker and new-status radio shown; confirm blocked until answered; after confirm, bad rows absent; new status saved in `/settings/status`; manual SKU saved in `/settings/sku-mapping`; "Peringatan" lists the price-mismatch row and both prefix-mismatch rows, confirm is not blocked by them, and after confirm those rows count in the dashboard using their `total_price` and channel column — _Note: verified via curl: 6 parse errors, 3 warnings, 2 SKU pickers, 1 status radio; confirm without answers → 422; after confirm net Rp1.100.000 / 8 orders, status and manual SKU saved. Client-side disabling of the confirm button (Alpine) needs a manual browser check_
- [x] Upload again a file where the previously manual name has an empty SKU → resolved automatically (no prompt) — _Note: verified via curl: 0 unresolved SKUs_
- [x] `/settings/status`: switch `Dikirim` to "Tidak dihitung" → dashboard net decreases immediately; switch back → Rp360.363.000 — _Note: verified via curl: Rp336.926.000 → back to Rp360.363.000_
- [x] `/settings/products`: rename BIO-CAM-BF → name shown in table and chart; clear → reverts to the most frequent listing name — _Note: verified via curl (table + chart payload); rendered chart label needs a manual browser check_
- [x] Upload `missing_column.csv` → "Kolom wajib tidak ditemukan: status"; current dataset unchanged — _Note: verified via curl: 422 + message, dataset unchanged_
- [x] Upload a `.txt` / oversized file → proper error; current dataset unchanged — _Note: verified via curl: .txt → 422, 11 MB → 413, dataset unchanged_
- [x] Cancel on review → dataset unchanged
- [ ] `docker compose restart` → dashboard still shows the last dataset, settings kept — _Note: Docker daemon unavailable. Persistence verified instead by restarting the local binary on the same `DATA_DIR` (dataset and settings kept). Needs a manual Docker check_
- [ ] All UI text in Bahasa Indonesia; all money in `Rp1.234.567` format; no browser `alert/confirm` dialogs used — _Note: code checked: no `hx-confirm`/`alert(`/`confirm(` anywhere, deletes use Alpine inline confirmation, money via `FormatIDR`. A visual pass over every page in a browser is still needed_
