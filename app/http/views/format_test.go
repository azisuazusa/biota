package views

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatIDR(t *testing.T) {
	assert.Equal(t, "Rp0", FormatIDR(0))
	assert.Equal(t, "Rp1", FormatIDR(1))
	assert.Equal(t, "Rp999", FormatIDR(999))
	assert.Equal(t, "Rp1.000", FormatIDR(1000))
	assert.Equal(t, "Rp1.099.000", FormatIDR(1099000))
	assert.Equal(t, "Rp1.020.858", FormatIDR(1020858))
	assert.Equal(t, "Rp360.363.000", FormatIDR(360363000))
	assert.Equal(t, "-Rp5.000", FormatIDR(-5000))
}

func TestFormatInt(t *testing.T) {
	assert.Equal(t, "473", FormatInt(473))
	assert.Equal(t, "1.234", FormatInt(1234))
}

func TestFormatDateID(t *testing.T) {
	assert.Equal(t, "21 Okt 2026", FormatDateID(time.Date(2026, 10, 21, 8, 24, 0, 0, time.UTC)))
	assert.Equal(t, "1 Mei 2026", FormatDateID(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, "31 Des 2026", FormatDateID(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, "3 Agu 2026 09:05", FormatDateTimeID(time.Date(2026, 8, 3, 9, 5, 0, 0, time.UTC)))
}

func TestFormatPct(t *testing.T) {
	assert.Equal(t, "60,3%", FormatPct(60.3189))
	assert.Equal(t, "0,0%", FormatPct(0))
	assert.Equal(t, "100,0%", FormatPct(100))
}
