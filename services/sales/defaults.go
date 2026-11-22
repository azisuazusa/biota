package sales

// DefaultConfig returns the config seeded on first start.
func DefaultConfig() *Config {
	cfg := &Config{
		StatusMapping: map[string]StatusClass{},
		StatusLabels:  map[string]string{},
		NameToSKU:     map[string]string{},
		DisplayNames:  map[string]string{},
	}
	for _, s := range []string{"Selesai", "Completed", "Pesanan Selesai", "Dikirim", "In transit", "Sedang Dikirim"} {
		cfg.SetStatus(s, StatusIncluded)
	}
	for _, s := range []string{"Batal", "Cancelled", "Dibatalkan", "Pengembalian Dana", "Refunded", "Dikomplain - Retur"} {
		cfg.SetStatus(s, StatusExcluded)
	}
	return cfg
}
