// Package views holds the templ components and display helpers.
package views

import (
	"strconv"
	"strings"
	"time"
)

var monthsID = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// FormatIDR formats rupiah as Rp1.099.000.
func FormatIDR(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	return sign(neg) + "Rp" + groupThousands(strconv.FormatInt(v, 10))
}

// FormatInt formats an integer with dot thousands separators.
func FormatInt(v int) string {
	neg := v < 0
	if neg {
		v = -v
	}
	return sign(neg) + groupThousands(strconv.Itoa(v))
}

func sign(neg bool) string {
	if neg {
		return "-"
	}
	return ""
}

func groupThousands(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// FormatDateID formats a date as 21 Okt 2026.
func FormatDateID(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + monthsID[t.Month()-1] + " " + strconv.Itoa(t.Year())
}

// FormatDateTimeID formats a timestamp as 21 Okt 2026 08:24.
func FormatDateTimeID(t time.Time) string {
	return FormatDateID(t) + " " + t.Format("15:04")
}

// FormatPct formats a percentage with one decimal and a comma: 60,3%.
func FormatPct(f float64) string {
	return strings.Replace(strconv.FormatFloat(f, 'f', 1, 64), ".", ",", 1) + "%"
}
