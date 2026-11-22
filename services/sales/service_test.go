package sales_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/azisuazusa/biota/services/sales"
	"github.com/azisuazusa/biota/services/sales/mocks"
)

type ServiceSuite struct {
	suite.Suite
	ctx    context.Context
	dsRepo *mocks.DatasetRepo
	cfg    *mocks.ConfigRepo
	svc    sales.Service
}

func TestServiceSuite(t *testing.T) { suite.Run(t, new(ServiceSuite)) }

func (s *ServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.dsRepo = mocks.NewDatasetRepo(s.T())
	s.cfg = mocks.NewConfigRepo(s.T())
	s.cfg.EXPECT().Load(mock.Anything).Return(sales.DefaultConfig(), nil).Once()
	s.dsRepo.EXPECT().Load(mock.Anything).Return(nil, sales.ErrNoDataset).Once()
	svc, err := sales.New(s.ctx, s.dsRepo, s.cfg)
	s.Require().NoError(err)
	s.svc = svc
}

func (s *ServiceSuite) sample() *os.File {
	f, err := os.Open(filepath.Join("..", "..", "test_data", "sample_orders_oct2026.csv"))
	s.Require().NoError(err)
	s.T().Cleanup(func() { _ = f.Close() })
	return f
}

func (s *ServiceSuite) start() *sales.PendingImport {
	p, err := s.svc.StartImport(s.ctx, "sample_orders_oct2026.csv", s.sample())
	s.Require().NoError(err)
	return p
}

func (s *ServiceSuite) confirmOK() *sales.Dataset {
	p := s.start()
	saveCfg := s.cfg.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	s.dsRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once().NotBefore(saveCfg)
	ds, err := s.svc.ConfirmImport(s.ctx, sales.ReviewDecision{Token: p.Token})
	s.Require().NoError(err)
	return ds
}

func (s *ServiceSuite) TestNewWithoutDataset() {
	_, err := s.svc.CurrentDataset(s.ctx)
	s.ErrorIs(err, sales.ErrNoDataset)
	_, err = s.svc.Summarize(s.ctx, sales.Filter{})
	s.ErrorIs(err, sales.ErrNoDataset)
}

func (s *ServiceSuite) TestNewLoadErrors() {
	cfg := mocks.NewConfigRepo(s.T())
	cfg.EXPECT().Load(mock.Anything).Return(nil, errors.New("disk")).Once()
	_, err := sales.New(s.ctx, mocks.NewDatasetRepo(s.T()), cfg)
	s.Error(err)

	cfg = mocks.NewConfigRepo(s.T())
	cfg.EXPECT().Load(mock.Anything).Return(sales.DefaultConfig(), nil).Once()
	ds := mocks.NewDatasetRepo(s.T())
	ds.EXPECT().Load(mock.Anything).Return(nil, errors.New("corrupt")).Once()
	_, err = sales.New(s.ctx, ds, cfg)
	s.Error(err)
}

func (s *ServiceSuite) TestStartConfirmHappyPath() {
	ds := s.confirmOK()
	s.Len(ds.Lines, 473)

	cur, err := s.svc.CurrentDataset(s.ctx)
	s.Require().NoError(err)
	s.Same(ds, cur)

	sum, err := s.svc.Summarize(s.ctx, sales.Filter{From: ds.MinDate, To: ds.MaxDate, Channels: sales.AllChannels})
	s.Require().NoError(err)
	s.Equal(int64(360363000), sum.NetRevenue)

	cfg, err := s.svc.GetConfig(s.ctx)
	s.Require().NoError(err)
	s.Len(cfg.NameToSKU, 36)

	// Pending is cleared.
	_, err = s.svc.GetPendingImport(s.ctx, "anything")
	s.ErrorIs(err, sales.ErrPendingNotFound)
}

func (s *ServiceSuite) TestConfirmWrongToken() {
	s.start()
	_, err := s.svc.ConfirmImport(s.ctx, sales.ReviewDecision{Token: "nope"})
	s.ErrorIs(err, sales.ErrPendingNotFound)
	_, err = s.svc.ConfirmImport(s.ctx, sales.ReviewDecision{})
	s.ErrorIs(err, sales.ErrPendingNotFound)
	// No Save expectations were set: mockery's cleanup fails the test if Save was called.
}

func (s *ServiceSuite) TestDatasetSaveErrorKeepsState() {
	first := s.confirmOK()

	p := s.start()
	s.cfg.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	s.dsRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(errors.New("disk full")).Once()
	keep := map[string]bool{}
	for _, g := range p.Duplicates {
		keep[g.Key] = true
	}
	_, err := s.svc.ConfirmImport(s.ctx, sales.ReviewDecision{Token: p.Token, KeepDuplicates: keep})
	s.Require().Error(err)
	s.Contains(err.Error(), "disk full")

	cur, err := s.svc.CurrentDataset(s.ctx)
	s.Require().NoError(err)
	s.Same(first, cur)

	got, err := s.svc.GetPendingImport(s.ctx, p.Token)
	s.Require().NoError(err)
	s.Same(p, got)
}

func (s *ServiceSuite) TestSecondStartInvalidatesFirst() {
	p1 := s.start()
	p2 := s.start()
	s.NotEqual(p1.Token, p2.Token)
	_, err := s.svc.GetPendingImport(s.ctx, p1.Token)
	s.ErrorIs(err, sales.ErrPendingNotFound)
	_, err = s.svc.GetPendingImport(s.ctx, p2.Token)
	s.NoError(err)
}

func (s *ServiceSuite) TestStartImportInvalidFileKeepsPending() {
	p := s.start()
	_, err := s.svc.StartImport(s.ctx, "x.csv", strings.NewReader("a,b\n1,2\n"))
	s.ErrorIs(err, sales.ErrMissingColumns)
	_, err = s.svc.GetPendingImport(s.ctx, p.Token)
	s.NoError(err)
}

func (s *ServiceSuite) TestCancel() {
	first := s.confirmOK()
	p := s.start()
	s.NoError(s.svc.CancelImport(s.ctx, p.Token))
	_, err := s.svc.GetPendingImport(s.ctx, p.Token)
	s.ErrorIs(err, sales.ErrPendingNotFound)
	cur, err := s.svc.CurrentDataset(s.ctx)
	s.Require().NoError(err)
	s.Same(first, cur)
}

func (s *ServiceSuite) TestSummarizeInvalidFilter() {
	s.confirmOK()
	_, err := s.svc.Summarize(s.ctx, sales.Filter{
		From:     time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Channels: sales.AllChannels,
	})
	s.ErrorIs(err, sales.ErrInvalidFilter)
}

func (s *ServiceSuite) TestSetDisplayName() {
	s.cfg.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Twice()
	s.Require().NoError(s.svc.SetDisplayName(s.ctx, "BIO-CAM-BF", "Kamera Andalan"))
	cfg, _ := s.svc.GetConfig(s.ctx)
	s.Equal("Kamera Andalan", cfg.DisplayNames["BIO-CAM-BF"])

	s.Require().NoError(s.svc.SetDisplayName(s.ctx, "BIO-CAM-BF", ""))
	cfg, _ = s.svc.GetConfig(s.ctx)
	s.NotContains(cfg.DisplayNames, "BIO-CAM-BF")

	err := s.svc.SetDisplayName(s.ctx, "BIO-CAM-BF", strings.Repeat("a", 201))
	s.ErrorIs(err, sales.ErrInvalidInput)
	var ie *sales.InputError
	s.Require().ErrorAs(err, &ie)
	s.Equal("Nama terlalu panjang", ie.Message)
}

func (s *ServiceSuite) TestConfigSaveErrorKeepsConfig() {
	s.cfg.EXPECT().Save(mock.Anything, mock.Anything).Return(errors.New("disk")).Once()
	s.Error(s.svc.SetStatusClass(s.ctx, "Selesai", sales.StatusExcluded))
	cfg, _ := s.svc.GetConfig(s.ctx)
	c, _ := cfg.Classify("Selesai")
	s.Equal(sales.StatusIncluded, c)
}

func (s *ServiceSuite) TestStatusAndMappingSettings() {
	s.cfg.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Times(4)

	s.Require().NoError(s.svc.SetStatusClass(s.ctx, "Menunggu Pembayaran", sales.StatusExcluded))
	s.ErrorIs(s.svc.SetStatusClass(s.ctx, " ", sales.StatusExcluded), sales.ErrInvalidInput)
	s.ErrorIs(s.svc.SetStatusClass(s.ctx, "X", "maybe"), sales.ErrInvalidInput)
	s.Require().NoError(s.svc.DeleteStatus(s.ctx, "menunggu pembayaran"))
	cfg, _ := s.svc.GetConfig(s.ctx)
	_, ok := cfg.Classify("Menunggu Pembayaran")
	s.False(ok)

	s.Require().NoError(s.svc.SetNameMapping(s.ctx, "Foo", "BIO-X"))
	s.ErrorIs(s.svc.SetNameMapping(s.ctx, "Foo", ""), sales.ErrInvalidInput)
	products, err := s.svc.ListProducts(s.ctx)
	s.Require().NoError(err)
	s.Equal([]sales.ProductInfo{{SKU: "BIO-X"}}, products)

	s.Require().NoError(s.svc.DeleteNameMapping(s.ctx, "Foo"))
	cfg, _ = s.svc.GetConfig(s.ctx)
	s.Empty(cfg.NameToSKU)
}

func (s *ServiceSuite) TestListProductsDefaults() {
	s.confirmOK()
	products, err := s.svc.ListProducts(s.ctx)
	s.Require().NoError(err)
	s.Len(products, 12)
	for _, p := range products {
		s.NotEmpty(p.DefaultName, p.SKU)
		s.Empty(p.Override)
	}
}
