package sales

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	sampleFile  = "sample_orders_oct2026.csv"
	edgeFile    = "edge_cases.csv"
	missingFile = "missing_column.csv"
	headerOnly  = "header_only.csv"
)

func openFixture(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "test_data", name))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func pendingFor(t *testing.T, name string, cfg *Config) *PendingImport {
	t.Helper()
	p, err := buildPendingImport(name, openFixture(t, name), cfg)
	require.NoError(t, err)
	return p
}

// sampleDataset imports the sample with default review choices (duplicates dropped).
func sampleDataset(t *testing.T, keepAll bool) (*Dataset, *Config) {
	t.Helper()
	cfg := DefaultConfig()
	p := pendingFor(t, sampleFile, cfg)
	keep := map[string]bool{}
	if keepAll {
		for _, g := range p.Duplicates {
			keep[g.Key] = true
		}
	}
	ds, newCfg, err := applyDecision(p, ReviewDecision{Token: p.Token, KeepDuplicates: keep}, cfg)
	require.NoError(t, err)
	return ds, newCfg
}
