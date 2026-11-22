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

type configRepo struct {
	path string
}

// NewConfigRepo stores the config at dataDir/config.json.
func NewConfigRepo(dataDir string) sales.ConfigRepo {
	return &configRepo{path: filepath.Join(dataDir, "config.json")}
}

func (r *configRepo) Load(ctx context.Context) (*sales.Config, error) {
	b, err := os.ReadFile(r.path)
	if errors.Is(err, fs.ErrNotExist) {
		cfg := sales.DefaultConfig()
		if err := r.Save(ctx, cfg); err != nil {
			return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	var cfg sales.Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	cfg.EnsureMaps()
	return &cfg, nil
}

func (r *configRepo) Save(_ context.Context, cfg *sales.Config) error {
	if err := writeJSONAtomic(r.path, cfg); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return nil
}
