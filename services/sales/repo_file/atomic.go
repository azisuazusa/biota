// Package repo_file implements the sales ports with JSON files.
package repo_file //nolint:revive,staticcheck // package name fixed by the plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/azisuazusa/biota/pkg/util"
)

// writeJSONAtomic writes v as indented JSON to a temp file in the same
// directory and renames it over path, so a crash never leaves a partial file.
func writeJSONAtomic(path string, v any) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(v); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("%s: %w", util.GetPackageAndFunctionName(), err)
	}
	return nil
}
