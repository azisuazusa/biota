package sales

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"github.com/azisuazusa/biota/app/http/views"
	"github.com/azisuazusa/biota/services/sales"
)

// multipartOverhead is extra body allowance for multipart boundaries and headers.
const multipartOverhead = 1 << 20

// Controller holds the sales HTTP handlers.
type Controller struct {
	svc            sales.Service
	maxUploadMB    int
	maxUploadBytes int64
}

// New builds a controller. maxUploadMB limits the uploaded file size.
func New(svc sales.Service, maxUploadMB int) *Controller {
	return &Controller{svc: svc, maxUploadMB: maxUploadMB, maxUploadBytes: int64(maxUploadMB) << 20}
}

func isHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

// render writes a component with the given status. It renders into a buffer
// first so a template error never sends a half-written page.
func render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	var buf bytes.Buffer
	if err := c.Render(r.Context(), &buf); err != nil {
		slog.ErrorContext(r.Context(), "render failed", "path", r.URL.Path, "error", err)
		http.Error(w, "Terjadi kesalahan", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// redirect sends HX-Redirect to HTMX clients and a 303 to plain browsers.
func redirect(w http.ResponseWriter, r *http.Request, to string) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", to)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
}

// Dashboard renders the full dashboard page.
func (c *Controller) Dashboard(w http.ResponseWriter, r *http.Request) {
	ds, err := c.svc.CurrentDataset(r.Context())
	if errors.Is(err, sales.ErrNoDataset) {
		render(w, r, http.StatusOK, views.DashboardEmpty())
		return
	}
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	f, errMsg := parseDashboardQuery(r, ds)
	if errMsg != "" {
		render(w, r, http.StatusBadRequest, views.DashboardPage(ds, f, nil, errMsg))
		return
	}
	s, err := c.svc.Summarize(r.Context(), f)
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	render(w, r, http.StatusOK, views.DashboardPage(ds, f, s, ""))
}

// DashboardWidgets renders only the widget area for HTMX filter changes.
func (c *Controller) DashboardWidgets(w http.ResponseWriter, r *http.Request) {
	if isHTMX(r) {
		// Push the bookmarkable full-page URL, not the partial's URL.
		push := "/"
		if r.URL.RawQuery != "" {
			push += "?" + r.URL.RawQuery
		}
		w.Header().Set("HX-Push-Url", push)
	}
	ds, err := c.svc.CurrentDataset(r.Context())
	if errors.Is(err, sales.ErrNoDataset) {
		_, msg := errorResponse(err)
		render(w, r, http.StatusOK, views.Widgets(nil, msg))
		return
	}
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	f, errMsg := parseDashboardQuery(r, ds)
	if errMsg != "" {
		render(w, r, http.StatusBadRequest, views.Widgets(nil, errMsg))
		return
	}
	s, err := c.svc.Summarize(r.Context(), f)
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	render(w, r, http.StatusOK, views.Widgets(s, ""))
}

// UploadPage renders the upload form.
func (c *Controller) UploadPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, views.UploadPage(c.maxUploadMB))
}

// Upload parses the CSV into a pending import and redirects to review.
func (c *Controller) Upload(w http.ResponseWriter, r *http.Request) {
	sizeMsg := fmt.Sprintf("Ukuran file melebihi batas %d MB", c.maxUploadMB)
	r.Body = http.MaxBytesReader(w, r.Body, c.maxUploadBytes+multipartOverhead)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			c.flash(w, r, http.StatusRequestEntityTooLarge, sizeMsg)
			return
		}
		c.flash(w, r, http.StatusUnprocessableEntity, "File wajib diunggah")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, hdr, err := r.FormFile("file")
	if err != nil {
		c.flash(w, r, http.StatusUnprocessableEntity, "File wajib diunggah")
		return
	}
	defer func() { _ = file.Close() }()

	name := filepath.Base(hdr.Filename)
	if !strings.EqualFold(filepath.Ext(name), ".csv") {
		c.flash(w, r, http.StatusUnprocessableEntity, "File harus berformat .csv")
		return
	}
	if hdr.Size > c.maxUploadBytes {
		c.flash(w, r, http.StatusRequestEntityTooLarge, sizeMsg)
		return
	}

	p, err := c.svc.StartImport(r.Context(), name, file)
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	redirect(w, r, "/upload/review?token="+url.QueryEscape(p.Token))
}

// ReviewPage renders the pending import for review.
func (c *Controller) ReviewPage(w http.ResponseWriter, r *http.Request) {
	p, err := c.svc.GetPendingImport(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	render(w, r, http.StatusOK, views.ReviewPage(p, ""))
}

// Confirm applies the review decisions and replaces the dataset.
func (c *Controller) Confirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		c.handleError(w, r, fmt.Errorf("%w: %w", sales.ErrInvalidInput, err))
		return
	}
	d := parseReviewDecision(r)
	_, err := c.svc.ConfirmImport(r.Context(), d)
	if errors.Is(err, sales.ErrUnresolvedSKU) || errors.Is(err, sales.ErrUnknownStatus) {
		p, perr := c.svc.GetPendingImport(r.Context(), d.Token)
		if perr != nil {
			c.handleError(w, r, perr)
			return
		}
		_, msg := errorResponse(err)
		if !isHTMX(r) {
			render(w, r, http.StatusUnprocessableEntity, views.ReviewPage(p, msg))
			return
		}
		render(w, r, http.StatusUnprocessableEntity, views.ReviewBody(p, &d, msg))
		return
	}
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	redirect(w, r, "/")
}

// Cancel discards the pending import.
func (c *Controller) Cancel(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		c.handleError(w, r, fmt.Errorf("%w: %w", sales.ErrInvalidInput, err))
		return
	}
	if err := c.svc.CancelImport(r.Context(), r.PostForm.Get("token")); err != nil {
		c.handleError(w, r, err)
		return
	}
	redirect(w, r, "/upload")
}

func (c *Controller) renderStatusTable(w http.ResponseWriter, r *http.Request, full bool) {
	cfg, err := c.svc.GetConfig(r.Context())
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	if full {
		render(w, r, http.StatusOK, views.StatusSettingsPage(cfg))
		return
	}
	render(w, r, http.StatusOK, views.StatusTable(cfg))
}

// StatusSettings renders the status mapping page.
func (c *Controller) StatusSettings(w http.ResponseWriter, r *http.Request) {
	c.renderStatusTable(w, r, true)
}

// SaveStatus adds or updates a status classification.
func (c *Controller) SaveStatus(w http.ResponseWriter, r *http.Request) {
	err := c.svc.SetStatusClass(r.Context(), r.FormValue("status"), sales.StatusClass(r.FormValue("class")))
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	c.renderStatusTable(w, r, !isHTMX(r))
}

// DeleteStatus removes a status from the mapping.
func (c *Controller) DeleteStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if r.URL.RawPath != "" {
		if s, err := url.PathUnescape(status); err == nil {
			status = s
		}
	}
	if err := c.svc.DeleteStatus(r.Context(), status); err != nil {
		c.handleError(w, r, err)
		return
	}
	c.renderStatusTable(w, r, !isHTMX(r))
}

func (c *Controller) renderProducts(w http.ResponseWriter, r *http.Request, full bool) {
	products, err := c.svc.ListProducts(r.Context())
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	if full {
		render(w, r, http.StatusOK, views.ProductSettingsPage(products))
		return
	}
	render(w, r, http.StatusOK, views.ProductTable(products))
}

// ProductSettings renders the display-name page.
func (c *Controller) ProductSettings(w http.ResponseWriter, r *http.Request) {
	c.renderProducts(w, r, true)
}

// SaveDisplayName sets or clears a SKU's display name.
func (c *Controller) SaveDisplayName(w http.ResponseWriter, r *http.Request) {
	if err := c.svc.SetDisplayName(r.Context(), r.FormValue("sku"), r.FormValue("name")); err != nil {
		c.handleError(w, r, err)
		return
	}
	c.renderProducts(w, r, !isHTMX(r))
}

func (c *Controller) renderSKUMapping(w http.ResponseWriter, r *http.Request, full bool) {
	cfg, err := c.svc.GetConfig(r.Context())
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	if full {
		render(w, r, http.StatusOK, views.SKUMappingSettingsPage(cfg))
		return
	}
	render(w, r, http.StatusOK, views.SKUMappingTable(cfg))
}

// SKUMappingSettings renders the product_name → SKU mapping page.
func (c *Controller) SKUMappingSettings(w http.ResponseWriter, r *http.Request) {
	c.renderSKUMapping(w, r, true)
}

// SaveSKUMapping adds or updates a mapping.
func (c *Controller) SaveSKUMapping(w http.ResponseWriter, r *http.Request) {
	if err := c.svc.SetNameMapping(r.Context(), r.FormValue("product_name"), r.FormValue("sku")); err != nil {
		c.handleError(w, r, err)
		return
	}
	c.renderSKUMapping(w, r, !isHTMX(r))
}

// DeleteSKUMapping removes a mapping. HTMX sends DELETE params in the query string.
func (c *Controller) DeleteSKUMapping(w http.ResponseWriter, r *http.Request) {
	if err := c.svc.DeleteNameMapping(r.Context(), r.FormValue("product_name")); err != nil {
		c.handleError(w, r, err)
		return
	}
	c.renderSKUMapping(w, r, !isHTMX(r))
}
