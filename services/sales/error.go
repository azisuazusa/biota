package sales

import (
	"errors"
	"strings"
)

var (
	ErrNoDataset       = errors.New("no dataset")
	ErrMissingColumns  = errors.New("missing required columns")
	ErrEmptyCSV        = errors.New("csv has no data rows")
	ErrInvalidCSV      = errors.New("csv cannot be read")
	ErrPendingNotFound = errors.New("pending import not found")
	ErrUnresolvedSKU   = errors.New("unresolved sku without choice")
	ErrUnknownStatus   = errors.New("unknown status without choice")
	ErrNoValidLines    = errors.New("no valid lines to import")
	ErrInvalidFilter   = errors.New("invalid filter")
	ErrInvalidInput    = errors.New("invalid input")
)

// MissingColumnsError names the required columns absent from the header.
type MissingColumnsError struct {
	Columns []string
}

func (e *MissingColumnsError) Error() string {
	return "missing required columns: " + strings.Join(e.Columns, ", ")
}

func (e *MissingColumnsError) Unwrap() error { return ErrMissingColumns }

// InputError is an ErrInvalidInput carrying a field-specific UI message.
type InputError struct {
	Message string
}

func (e *InputError) Error() string { return "invalid input: " + e.Message }

func (e *InputError) Unwrap() error { return ErrInvalidInput }
