// Package app builds the HTTP router.
package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	salesctrl "github.com/azisuazusa/biota/app/http/controller/sales"
)

// NewRouter registers every route from PRD §5.
func NewRouter(c *salesctrl.Controller) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)

	r.Get("/", c.Dashboard)
	r.Get("/dashboard/widgets", c.DashboardWidgets)

	r.Get("/upload", c.UploadPage)
	r.Post("/upload", c.Upload)
	r.Get("/upload/review", c.ReviewPage)
	r.Post("/upload/confirm", c.Confirm)
	r.Post("/upload/cancel", c.Cancel)

	r.Get("/settings/status", c.StatusSettings)
	r.Post("/settings/status", c.SaveStatus)
	r.Delete("/settings/status/{status}", c.DeleteStatus)

	r.Get("/settings/products", c.ProductSettings)
	r.Post("/settings/products", c.SaveDisplayName)

	r.Get("/settings/sku-mapping", c.SKUMappingSettings)
	r.Post("/settings/sku-mapping", c.SaveSKUMapping)
	r.Delete("/settings/sku-mapping", c.DeleteSKUMapping)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	return r
}
