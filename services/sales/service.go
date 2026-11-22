package sales

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"

	"github.com/azisuazusa/biota/pkg/util"
)

const maxDisplayNameLen = 200

// Service is the sales use-case port used by the HTTP layer.
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

type service struct {
	mu       sync.RWMutex
	dsRepo   DatasetRepo
	cfgRepo  ConfigRepo
	ds       *Dataset
	cfg      *Config
	pending  *PendingImport
	validate *validator.Validate
}

// New loads the config (seeding defaults) and the last dataset, if any.
func New(ctx context.Context, ds DatasetRepo, cfg ConfigRepo) (Service, error) {
	s := &service{dsRepo: ds, cfgRepo: cfg, validate: validator.New()}
	c, err := cfg.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	c.ensureMaps()
	s.cfg = c

	d, err := ds.Load(ctx)
	switch {
	case errors.Is(err, ErrNoDataset):
	case err != nil:
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	default:
		s.ds = d
	}
	return s, nil
}

func (s *service) StartImport(_ context.Context, fileName string, r io.Reader) (*PendingImport, error) {
	s.mu.RLock()
	cfg := s.cfg.Clone()
	s.mu.RUnlock()

	p, err := buildPendingImport(fileName, r, cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	s.mu.Lock()
	s.pending = p
	s.mu.Unlock()
	return p, nil
}

func (s *service) GetPendingImport(_ context.Context, token string) (*PendingImport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pending == nil || token == "" || s.pending.Token != token {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrPendingNotFound)
	}
	return s.pending, nil
}

func (s *service) ConfirmImport(ctx context.Context, d ReviewDecision) (*Dataset, error) {
	if err := s.validate.Struct(d); err != nil {
		return nil, fmt.Errorf("%s: %w: %w", util.GetPackageAndFunctionName(), ErrPendingNotFound, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil || s.pending.Token != d.Token {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrPendingNotFound)
	}
	ds, cfg, err := applyDecision(s.pending, d, s.cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	if err := s.cfgRepo.Save(ctx, cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	if err := s.dsRepo.Save(ctx, ds); err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	s.ds, s.cfg, s.pending = ds, cfg, nil
	return ds, nil
}

func (s *service) CancelImport(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending != nil && s.pending.Token == token {
		s.pending = nil
	}
	return nil
}

func (s *service) CurrentDataset(_ context.Context) (*Dataset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ds == nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrNoDataset)
	}
	return s.ds, nil
}

func (s *service) Summarize(_ context.Context, f Filter) (*Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ds == nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrNoDataset)
	}
	if DateOnly(f.From).After(DateOnly(f.To)) {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), ErrInvalidFilter)
	}
	return summarize(s.ds, s.cfg, f), nil
}

func (s *service) GetConfig(_ context.Context) (*Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Clone(), nil
}

// updateConfig applies fn to a clone, persists it and swaps it in on success.
func (s *service) updateConfig(ctx context.Context, fn func(c *Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cfg.Clone()
	fn(c)
	if err := s.cfgRepo.Save(ctx, c); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	s.cfg = c
	return nil
}

func (s *service) SetStatusClass(ctx context.Context, status string, class StatusClass) error {
	status = strings.TrimSpace(status)
	if status == "" || (class != StatusIncluded && class != StatusExcluded) {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), &InputError{Message: "Status wajib diisi"})
	}
	if err := s.updateConfig(ctx, func(c *Config) { c.SetStatus(status, class) }); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return nil
}

func (s *service) DeleteStatus(ctx context.Context, status string) error {
	k := statusKey(status)
	if k == "" {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), &InputError{Message: "Status wajib diisi"})
	}
	err := s.updateConfig(ctx, func(c *Config) {
		delete(c.StatusMapping, k)
		delete(c.StatusLabels, k)
	})
	if err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return nil
}

func (s *service) SetDisplayName(ctx context.Context, sku, name string) error {
	sku, name = strings.TrimSpace(sku), strings.TrimSpace(name)
	if sku == "" {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), &InputError{Message: "SKU wajib diisi"})
	}
	if utf8.RuneCountInString(name) > maxDisplayNameLen {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), &InputError{Message: "Nama terlalu panjang"})
	}
	err := s.updateConfig(ctx, func(c *Config) {
		if name == "" {
			delete(c.DisplayNames, sku)
		} else {
			c.DisplayNames[sku] = name
		}
	})
	if err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return nil
}

func (s *service) ListProducts(_ context.Context) ([]ProductInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	defaults := defaultDisplayNames(s.ds)
	skus := map[string]bool{}
	for sku := range defaults {
		skus[sku] = true
	}
	for sku := range s.cfg.DisplayNames {
		skus[sku] = true
	}
	for _, sku := range s.cfg.NameToSKU {
		if sku != "" {
			skus[sku] = true
		}
	}
	out := make([]ProductInfo, 0, len(skus))
	for _, sku := range sortedKeys(skus) {
		out = append(out, ProductInfo{SKU: sku, DefaultName: defaults[sku], Override: s.cfg.DisplayNames[sku]})
	}
	return out, nil
}

func (s *service) SetNameMapping(ctx context.Context, productName, sku string) error {
	productName, sku = strings.TrimSpace(productName), strings.TrimSpace(sku)
	if productName == "" || sku == "" {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), &InputError{Message: "Nama produk dan SKU wajib diisi"})
	}
	if err := s.updateConfig(ctx, func(c *Config) { c.NameToSKU[productName] = sku }); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return nil
}

func (s *service) DeleteNameMapping(ctx context.Context, productName string) error {
	productName = strings.TrimSpace(productName)
	if productName == "" {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), &InputError{Message: "Nama produk dan SKU wajib diisi"})
	}
	if err := s.updateConfig(ctx, func(c *Config) { delete(c.NameToSKU, productName) }); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return nil
}
