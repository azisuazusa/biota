// Package util holds small cross-cutting helpers.
package util

import (
	"path"
	"runtime"
)

// GetPackageAndFunctionName returns the caller's "package.Function" name,
// e.g. "sales.(*service).StartImport". It is used to prefix wrapped errors.
func GetPackageAndFunctionName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}
	// fn.Name() is e.g. "github.com/azisuazusa/biota/services/sales.(*service).StartImport".
	return path.Base(fn.Name())
}
