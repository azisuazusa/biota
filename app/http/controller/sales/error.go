package sales

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/azisuazusa/biota/app/http/views"
	"github.com/azisuazusa/biota/app/http/views/components"
	"github.com/azisuazusa/biota/services/sales"
)

// errorResponse maps a domain error to an HTTP status and UI message.
func errorResponse(err error) (int, string) {
	var mce *sales.MissingColumnsError
	var ie *sales.InputError
	switch {
	case errors.As(err, &mce):
		return http.StatusUnprocessableEntity, "Kolom wajib tidak ditemukan: " + strings.Join(mce.Columns, ", ")
	case errors.Is(err, sales.ErrNoDataset):
		return http.StatusOK, "Belum ada data. Unggah file CSV terlebih dahulu."
	case errors.Is(err, sales.ErrEmptyCSV):
		return http.StatusUnprocessableEntity, "File CSV tidak berisi data"
	case errors.Is(err, sales.ErrInvalidCSV):
		return http.StatusUnprocessableEntity, "File CSV tidak dapat dibaca"
	case errors.Is(err, sales.ErrPendingNotFound):
		return http.StatusNotFound, "Sesi impor tidak ditemukan. Silakan unggah ulang."
	case errors.Is(err, sales.ErrUnresolvedSKU):
		return http.StatusUnprocessableEntity, "Pilih SKU untuk semua produk yang belum dikenali"
	case errors.Is(err, sales.ErrUnknownStatus):
		return http.StatusUnprocessableEntity, "Tentukan klasifikasi untuk semua status baru"
	case errors.Is(err, sales.ErrNoValidLines):
		return http.StatusUnprocessableEntity, "Tidak ada baris valid untuk diimpor"
	case errors.Is(err, sales.ErrInvalidFilter):
		return http.StatusBadRequest, msgInvalidFilter
	case errors.As(err, &ie):
		return http.StatusBadRequest, ie.Message
	case errors.Is(err, sales.ErrInvalidInput):
		return http.StatusBadRequest, "Input tidak valid"
	default:
		return http.StatusInternalServerError, "Terjadi kesalahan saat menyimpan data"
	}
}

// handleError renders the mapped message as an alert into #flash.
func (c *Controller) handleError(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := errorResponse(err)
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	}
	c.flash(w, r, status, msg)
}

// flash renders an error alert into #flash with the given status.
func (c *Controller) flash(w http.ResponseWriter, r *http.Request, status int, msg string) {
	if !isHTMX(r) {
		render(w, r, status, views.ErrorPage(msg))
		return
	}
	w.Header().Set("HX-Retarget", "#flash")
	w.Header().Set("HX-Reswap", "innerHTML")
	render(w, r, status, components.Flash(msg))
}
