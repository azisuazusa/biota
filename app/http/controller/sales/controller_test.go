package sales

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/azisuazusa/biota/services/sales"
	"github.com/azisuazusa/biota/services/sales/repo_file"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "test_data", name))
	require.NoError(t, err)
	return b
}

func newController(t *testing.T, maxMB int) *Controller {
	t.Helper()
	dir := t.TempDir()
	svc, err := sales.New(context.Background(), repo_file.NewDatasetRepo(dir), repo_file.NewConfigRepo(dir))
	require.NoError(t, err)
	return New(svc, maxMB)
}

func uploadReq(t *testing.T, fileName string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	r := httptest.NewRequest(http.MethodPost, "/upload", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("HX-Request", "true")
	return r
}

func formReq(method, target string, form url.Values) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	return r
}

var tokenRe = regexp.MustCompile(`token=([0-9a-f]{32})`)

func upload(t *testing.T, c *Controller, name string) string {
	t.Helper()
	w := httptest.NewRecorder()
	c.Upload(w, uploadReq(t, name, fixture(t, name)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	m := tokenRe.FindStringSubmatch(w.Header().Get("HX-Redirect"))
	require.Len(t, m, 2, "HX-Redirect must carry a token: %q", w.Header().Get("HX-Redirect"))
	return m[1]
}

func confirm(t *testing.T, c *Controller, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c.Confirm(w, formReq(http.MethodPost, "/upload/confirm", form))
	return w
}

func get(c http.HandlerFunc, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.Header.Set("HX-Request", "true")
	c(w, r)
	return w
}

func TestUploadSampleRedirectsToReview(t *testing.T) {
	c := newController(t, 10)
	tok := upload(t, c, "sample_orders_oct2026.csv")

	w := get(c.ReviewPage, "/upload/review?token="+tok)
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Tinjau Impor")
	assert.Contains(t, body, "Simpan duplikat")
	assert.Equal(t, 22, strings.Count(body, `name="keep_dup[`))
	assert.NotContains(t, body, "checked", "duplicate checkboxes default to unchecked")
	assert.NotContains(t, body, "Baris gagal dibaca</h2>")
	assert.NotContains(t, body, "SKU belum dikenali</h2>")
}

func TestUploadRejectsNonCSV(t *testing.T) {
	c := newController(t, 10)
	w := httptest.NewRecorder()
	c.Upload(w, uploadReq(t, "data.txt", []byte("x")))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "File harus berformat .csv")
	assert.Equal(t, "#flash", w.Header().Get("HX-Retarget"))
}

func TestUploadMissingFile(t *testing.T) {
	c := newController(t, 10)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	require.NoError(t, mw.WriteField("x", "y"))
	require.NoError(t, mw.Close())
	r := httptest.NewRequest(http.MethodPost, "/upload", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	c.Upload(w, r)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "File wajib diunggah")
}

func TestUploadTooLarge(t *testing.T) {
	c := newController(t, 1)
	big := append([]byte(strings.Join(sales.RequiredColumns, ",")+"\n"), bytes.Repeat([]byte("a"), (1<<20)+10)...)
	w := httptest.NewRecorder()
	c.Upload(w, uploadReq(t, "big.csv", big))
	assert.Contains(t, []int{http.StatusRequestEntityTooLarge, http.StatusUnprocessableEntity}, w.Code)
	assert.Contains(t, w.Body.String(), "Ukuran file melebihi batas 1 MB")

	// Way over the body allowance too.
	huge := bytes.Repeat([]byte("a"), 3<<20)
	w = httptest.NewRecorder()
	c.Upload(w, uploadReq(t, "huge.csv", huge))
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), "Ukuran file melebihi batas 1 MB")
}

func TestUploadMissingColumn(t *testing.T) {
	c := newController(t, 10)
	w := httptest.NewRecorder()
	c.Upload(w, uploadReq(t, "missing_column.csv", fixture(t, "missing_column.csv")))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Kolom wajib tidak ditemukan: status")
}

func TestUploadHeaderOnly(t *testing.T) {
	c := newController(t, 10)
	w := httptest.NewRecorder()
	c.Upload(w, uploadReq(t, "header_only.csv", fixture(t, "header_only.csv")))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "File CSV tidak berisi data")
}

func TestConfirmThenWidgets(t *testing.T) {
	c := newController(t, 10)
	tok := upload(t, c, "sample_orders_oct2026.csv")

	w := confirm(t, c, url.Values{"token": {tok}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "/", w.Header().Get("HX-Redirect"))

	w = get(c.DashboardWidgets, "/dashboard/widgets")
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Rp360.363.000")
	assert.Contains(t, body, "Rp1.020.858")
	assert.Contains(t, body, "Rp228.176.000")
	assert.Contains(t, body, "Rp87.666.000")
	assert.Contains(t, body, "Rp44.521.000")
	assert.Contains(t, body, "Rp67.548.000")
	assert.Contains(t, body, ">353<")
	assert.Contains(t, body, ">517<")
	assert.Equal(t, "/", w.Header().Get("HX-Push-Url"))

	// Shopee only, URL pushed as the full-page URL.
	w = get(c.DashboardWidgets, "/dashboard/widgets?filtered=1&from=2026-10-01&to=2026-10-31&channel=Shopee")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Rp228.176.000")
	assert.NotContains(t, w.Body.String(), "Rp360.363.000")
	assert.Equal(t, "/?filtered=1&from=2026-10-01&to=2026-10-31&channel=Shopee", w.Header().Get("HX-Push-Url"))

	// All channels unchecked.
	w = get(c.DashboardWidgets, "/dashboard/widgets?filtered=1&from=2026-10-01&to=2026-10-31")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Tidak ada data untuk filter ini")

	// Full page with the filter keeps the checkbox state.
	w = get(c.Dashboard, "/?filtered=1&channel=Tokopedia")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Rp44.521.000")
	assert.Contains(t, w.Body.String(), "sample_orders_oct2026.csv")
	assert.Contains(t, w.Body.String(), "1 Okt 2026")
	assert.Contains(t, w.Body.String(), "31 Okt 2026")
}

func TestConfirmKeepAllDuplicates(t *testing.T) {
	c := newController(t, 10)
	tok := upload(t, c, "sample_orders_oct2026.csv")
	body := get(c.ReviewPage, "/upload/review?token="+tok).Body.String()

	form := url.Values{"token": {tok}}
	for _, m := range regexp.MustCompile(`name="(keep_dup\[[0-9a-f]+\])"`).FindAllStringSubmatch(body, -1) {
		form.Set(m[1], "on")
	}
	require.Len(t, form, 23)
	w := confirm(t, c, form)
	require.Equal(t, http.StatusOK, w.Code)

	w = get(c.DashboardWidgets, "/dashboard/widgets")
	assert.Contains(t, w.Body.String(), "Rp375.865.000")
	assert.Contains(t, w.Body.String(), ">545<")
}

func TestInvalidFilter(t *testing.T) {
	c := newController(t, 10)
	tok := upload(t, c, "sample_orders_oct2026.csv")
	require.Equal(t, http.StatusOK, confirm(t, c, url.Values{"token": {tok}}).Code)

	w := get(c.DashboardWidgets, "/dashboard/widgets?from=2026-10-10&to=2026-10-01")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Rentang tanggal tidak valid")

	w = get(c.DashboardWidgets, "/dashboard/widgets?from=2026-09-01&to=2026-10-01")
	assert.Equal(t, http.StatusBadRequest, w.Code, "outside the dataset's dates")

	w = get(c.DashboardWidgets, "/dashboard/widgets?from=bad")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = get(c.Dashboard, "/?from=2026-10-10&to=2026-10-01")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Rentang tanggal tidak valid")
}

func TestDashboardEmptyState(t *testing.T) {
	c := newController(t, 10)
	w := get(c.Dashboard, "/")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Belum ada data. Unggah file CSV terlebih dahulu.")
}

func TestEdgeCasesReviewAndConfirm(t *testing.T) {
	c := newController(t, 10)
	tok := upload(t, c, "edge_cases.csv")

	body := get(c.ReviewPage, "/upload/review?token="+tok).Body.String()
	assert.Contains(t, body, "Baris gagal dibaca")
	assert.Contains(t, body, "Format tanggal tidak dikenali")
	assert.Contains(t, body, "Peringatan")
	assert.Contains(t, body, "Jumlah × harga satuan (Rp200.000) tidak sama dengan total (Rp150.000)")
	assert.Contains(t, body, "Prefix order_id (TKP) tidak sesuai channel Shopee")
	assert.Contains(t, body, "Prefix order_id (XYZ) tidak sesuai channel Tokopedia")
	assert.Contains(t, body, `name="sku[Produk Misterius]"`)
	assert.Contains(t, body, `name="status[Menunggu Pembayaran]"`)

	// Missing answers → 422 with the review re-rendered.
	w := confirm(t, c, url.Values{"token": {tok}, "status[Menunggu Pembayaran]": {"include"}})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Pilih SKU untuk semua produk yang belum dikenali")
	assert.Contains(t, w.Body.String(), `id="review"`)

	w = confirm(t, c, url.Values{"token": {tok}, "sku[Produk Misterius]": {"BIO-NEW"}, "sku[Produk Ganda]": {"BIO-B"}})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Tentukan klasifikasi untuk semua status baru")
	assert.Contains(t, w.Body.String(), `value="BIO-NEW"`, "answers are kept when re-rendering")

	w = confirm(t, c, url.Values{
		"token":                       {tok},
		"sku[Produk Misterius]":       {"BIO-NEW"},
		"sku[Produk Ganda]":           {"BIO-B"},
		"status[Menunggu Pembayaran]": {"exclude"},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Valid rows: 2 (100k) + 9 (50k) + 10/11/12 (200k each) + 14 (150k) + 15 (100k) + 16 (100k); 13 excluded.
	w = get(c.DashboardWidgets, "/dashboard/widgets")
	assert.Contains(t, w.Body.String(), "Rp1.100.000")

	assert.Contains(t, get(c.StatusSettings, "/settings/status").Body.String(), "Menunggu Pembayaran")
	sm := get(c.SKUMappingSettings, "/settings/sku-mapping").Body.String()
	assert.Contains(t, sm, "Produk Misterius")
	assert.Contains(t, sm, "BIO-NEW")
}

func TestConfirmBadTokenAndCancel(t *testing.T) {
	c := newController(t, 10)
	w := confirm(t, c, url.Values{"token": {"deadbeef"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Sesi impor tidak ditemukan. Silakan unggah ulang.")

	w = get(c.ReviewPage, "/upload/review?token=nope")
	assert.Equal(t, http.StatusNotFound, w.Code)

	tok := upload(t, c, "sample_orders_oct2026.csv")
	w = httptest.NewRecorder()
	c.Cancel(w, formReq(http.MethodPost, "/upload/cancel", url.Values{"token": {tok}}))
	assert.Equal(t, "/upload", w.Header().Get("HX-Redirect"))
	assert.Equal(t, http.StatusNotFound, get(c.ReviewPage, "/upload/review?token="+tok).Code)
	assert.Contains(t, get(c.Dashboard, "/").Body.String(), "Belum ada data")
}

func TestSettingsHandlers(t *testing.T) {
	c := newController(t, 10)
	tok := upload(t, c, "sample_orders_oct2026.csv")
	require.Equal(t, http.StatusOK, confirm(t, c, url.Values{"token": {tok}}).Code)

	// Status toggle applies immediately.
	w := httptest.NewRecorder()
	c.SaveStatus(w, formReq(http.MethodPost, "/settings/status", url.Values{"status": {"Dikirim"}, "class": {"exclude"}}))
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, get(c.DashboardWidgets, "/dashboard/widgets").Body.String(), "Rp360.363.000")
	w = httptest.NewRecorder()
	c.SaveStatus(w, formReq(http.MethodPost, "/settings/status", url.Values{"status": {"Dikirim"}, "class": {"include"}}))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, get(c.DashboardWidgets, "/dashboard/widgets").Body.String(), "Rp360.363.000")

	w = httptest.NewRecorder()
	c.SaveStatus(w, formReq(http.MethodPost, "/settings/status", url.Values{"status": {""}, "class": {"include"}}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Status wajib diisi")

	// Add then delete a status via the chi URL param.
	w = httptest.NewRecorder()
	c.SaveStatus(w, formReq(http.MethodPost, "/settings/status", url.Values{"status": {"Menunggu Pembayaran"}, "class": {"exclude"}}))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Menunggu Pembayaran")
	router := chi.NewRouter()
	router.Delete("/settings/status/{status}", c.DeleteStatus)
	w = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/settings/status/"+url.PathEscape("Menunggu Pembayaran"), nil)
	r.Header.Set("HX-Request", "true")
	router.ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "Menunggu Pembayaran")

	// Display name override shows in the dashboard and reverts when cleared.
	w = httptest.NewRecorder()
	c.SaveDisplayName(w, formReq(http.MethodPost, "/settings/products", url.Values{"sku": {"BIO-CAM-BF"}, "name": {"Kamera Andalan"}}))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, get(c.DashboardWidgets, "/dashboard/widgets").Body.String(), "Kamera Andalan")
	w = httptest.NewRecorder()
	c.SaveDisplayName(w, formReq(http.MethodPost, "/settings/products", url.Values{"sku": {"BIO-CAM-BF"}, "name": {""}}))
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, get(c.DashboardWidgets, "/dashboard/widgets").Body.String(), "Kamera Andalan")
	w = httptest.NewRecorder()
	c.SaveDisplayName(w, formReq(http.MethodPost, "/settings/products", url.Values{"sku": {"BIO-CAM-BF"}, "name": {strings.Repeat("a", 201)}}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Nama terlalu panjang")
	assert.Equal(t, 12, strings.Count(get(c.ProductSettings, "/settings/products").Body.String(), `name="name"`))

	// SKU mapping add / validation / delete (DELETE params come in the query string).
	w = httptest.NewRecorder()
	c.SaveSKUMapping(w, formReq(http.MethodPost, "/settings/sku-mapping", url.Values{"product_name": {"Produk Baru"}, "sku": {"BIO-NEW"}}))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Produk Baru")
	w = httptest.NewRecorder()
	c.SaveSKUMapping(w, formReq(http.MethodPost, "/settings/sku-mapping", url.Values{"product_name": {"X"}}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Nama produk dan SKU wajib diisi")
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodDelete, "/settings/sku-mapping?product_name="+url.QueryEscape("Produk Baru"), nil)
	r.Header.Set("HX-Request", "true")
	c.DeleteSKUMapping(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "Produk Baru")
}

func TestParseReviewDecision(t *testing.T) {
	form := url.Values{
		"token":              {" abc "},
		"keep_dup[k1]":       {"on"},
		"sku[Nama [A] Baru]": {" BIO-1 "},
		"status[Menunggu]":   {"include"},
		"other":              {"x"},
	}
	r := formReq(http.MethodPost, "/upload/confirm", form)
	require.NoError(t, r.ParseForm())
	d := parseReviewDecision(r)
	assert.Equal(t, "abc", d.Token)
	assert.True(t, d.KeepDuplicates["k1"])
	assert.Equal(t, "BIO-1", d.SKUChoices["Nama [A] Baru"])
	assert.Equal(t, sales.StatusIncluded, d.StatusChoices["Menunggu"])
}
