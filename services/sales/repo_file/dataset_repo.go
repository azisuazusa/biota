package repo_file //nolint:revive,staticcheck // package name fixed by the plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/azisuazusa/biota/pkg/util"
	"github.com/azisuazusa/biota/services/sales"
)

type datasetRepo struct {
	path string
}

// NewDatasetRepo stores the dataset at dataDir/dataset.json.
func NewDatasetRepo(dataDir string) sales.DatasetRepo {
	return &datasetRepo{path: filepath.Join(dataDir, "dataset.json")}
}

func (r *datasetRepo) Load(_ context.Context) (*sales.Dataset, error) {
	b, err := os.ReadFile(r.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, sales.ErrNoDataset
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	var ds sales.Dataset
	if err := json.Unmarshal(b, &ds); err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return &ds, nil
}

func (r *datasetRepo) Save(_ context.Context, ds *sales.Dataset) error {
	if err := writeJSONAtomic(r.path, ds); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return nil
}
