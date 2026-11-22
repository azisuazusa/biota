package sales

import "context"

// DatasetRepo persists the last confirmed dataset.
type DatasetRepo interface {
	Load(ctx context.Context) (*Dataset, error) // ErrNoDataset if file absent
	Save(ctx context.Context, ds *Dataset) error
}

// ConfigRepo persists user settings.
type ConfigRepo interface {
	Load(ctx context.Context) (*Config, error) // DefaultConfig() + Save if absent
	Save(ctx context.Context, cfg *Config) error
}
