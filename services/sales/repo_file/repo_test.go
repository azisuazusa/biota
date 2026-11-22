package repo_file //nolint:revive,staticcheck // package name fixed by the plan

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/azisuazusa/biota/services/sales"
)

type RepoSuite struct {
	suite.Suite
	dir string
	ctx context.Context
}

func TestRepoSuite(t *testing.T) { suite.Run(t, new(RepoSuite)) }

func (s *RepoSuite) SetupTest() {
	s.dir = s.T().TempDir()
	s.ctx = context.Background()
}

func (s *RepoSuite) tmpFiles() []string {
	m, err := filepath.Glob(filepath.Join(s.dir, ".tmp-*"))
	s.Require().NoError(err)
	return m
}

func (s *RepoSuite) TestDatasetLoadEmpty() {
	_, err := NewDatasetRepo(s.dir).Load(s.ctx)
	s.ErrorIs(err, sales.ErrNoDataset)
}

func (s *RepoSuite) TestDatasetRoundTrip() {
	r := NewDatasetRepo(s.dir)
	d1 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 10, 21, 8, 24, 0, 0, time.UTC)
	ds := &sales.Dataset{
		FileName:   "biota.csv",
		UploadedAt: time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC),
		MinDate:    d1,
		MaxDate:    time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		Lines: []sales.OrderLine{
			{SourceLine: 2, OrderID: "TKP1", OrderDate: d1, HasTime: false, Channel: sales.ChannelTokopedia, SKU: "BIO-A", SKUInferred: true, ProductName: "A", Qty: 2, UnitPrice: 100, TotalPrice: 200, Status: "Selesai", BuyerCity: "Medan"},
			{SourceLine: 3, OrderID: "SHP1", OrderDate: d2, HasTime: true, Channel: sales.ChannelShopee, SKU: "BIO-B", ProductName: "B", Qty: 1, UnitPrice: 1099000, TotalPrice: 1099000, Status: "Batal", BuyerCity: "Bandung"},
		},
	}
	s.Require().NoError(r.Save(s.ctx, ds))
	got, err := r.Load(s.ctx)
	s.Require().NoError(err)
	s.Equal(ds, got)
	s.True(got.Lines[1].OrderDate.Equal(d2))
	s.True(got.Lines[1].HasTime)
	s.False(got.Lines[0].HasTime)
}

func (s *RepoSuite) TestConfigLoadSeedsDefaults() {
	cfg, err := NewConfigRepo(s.dir).Load(s.ctx)
	s.Require().NoError(err)
	s.Len(cfg.StatusMapping, 12)
	_, err = os.Stat(filepath.Join(s.dir, "config.json"))
	s.NoError(err)

	// Loading again reads the file.
	cfg2, err := NewConfigRepo(s.dir).Load(s.ctx)
	s.Require().NoError(err)
	s.Equal(cfg, cfg2)
}

func (s *RepoSuite) TestConfigRoundTrip() {
	r := NewConfigRepo(s.dir)
	cfg := sales.DefaultConfig()
	cfg.SetStatus("Menunggu Pembayaran", sales.StatusExcluded)
	cfg.NameToSKU["Produk A"] = "BIO-A"
	cfg.DisplayNames["BIO-A"] = "Nama A"
	s.Require().NoError(r.Save(s.ctx, cfg))
	got, err := r.Load(s.ctx)
	s.Require().NoError(err)
	s.Equal(cfg, got)
}

func (s *RepoSuite) TestConfigNilMapsInitialised() {
	s.Require().NoError(os.WriteFile(filepath.Join(s.dir, "config.json"), []byte(`{}`), 0o600))
	cfg, err := NewConfigRepo(s.dir).Load(s.ctx)
	s.Require().NoError(err)
	s.NotNil(cfg.StatusMapping)
	s.NotNil(cfg.StatusLabels)
	s.NotNil(cfg.NameToSKU)
	s.NotNil(cfg.DisplayNames)
}

func (s *RepoSuite) TestAtomicWriteOverwriteNoTemp() {
	r := NewDatasetRepo(s.dir)
	s.Require().NoError(r.Save(s.ctx, &sales.Dataset{FileName: "a.csv"}))
	s.Require().NoError(r.Save(s.ctx, &sales.Dataset{FileName: "b.csv"}))
	got, err := r.Load(s.ctx)
	s.Require().NoError(err)
	s.Equal("b.csv", got.FileName)
	s.Empty(s.tmpFiles())
}

func (s *RepoSuite) TestAtomicWriteFailureCleansUp() {
	// A value json cannot encode makes the write fail after the temp file exists.
	err := writeJSONAtomic(filepath.Join(s.dir, "x.json"), map[string]any{"c": make(chan int)})
	s.Error(err)
	s.Empty(s.tmpFiles())
	_, statErr := os.Stat(filepath.Join(s.dir, "x.json"))
	s.True(os.IsNotExist(statErr))
}

func (s *RepoSuite) TestCorruptJSON() {
	s.Require().NoError(os.WriteFile(filepath.Join(s.dir, "dataset.json"), []byte("{not json"), 0o600))
	_, err := NewDatasetRepo(s.dir).Load(s.ctx)
	s.Error(err)
	s.NotErrorIs(err, sales.ErrNoDataset)
	s.Contains(err.Error(), "repo_file.(*datasetRepo).Load")

	s.Require().NoError(os.WriteFile(filepath.Join(s.dir, "config.json"), []byte("[]x"), 0o600))
	_, err = NewConfigRepo(s.dir).Load(s.ctx)
	s.Error(err)
}
