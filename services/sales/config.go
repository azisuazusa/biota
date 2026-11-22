package sales

import (
	"maps"
	"strings"
)

func statusKey(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

// Classify looks up a status case-insensitively after trimming.
func (c *Config) Classify(status string) (StatusClass, bool) {
	class, ok := c.StatusMapping[statusKey(status)]
	return class, ok
}

// SetStatus maps a status to a class and remembers its display casing.
func (c *Config) SetStatus(status string, class StatusClass) {
	if c.StatusMapping == nil {
		c.StatusMapping = map[string]StatusClass{}
	}
	if c.StatusLabels == nil {
		c.StatusLabels = map[string]string{}
	}
	k := statusKey(status)
	c.StatusMapping[k] = class
	c.StatusLabels[k] = strings.TrimSpace(status)
}

// Clone returns a deep copy, so callers never mutate shared state.
func (c *Config) Clone() *Config {
	out := &Config{
		StatusMapping: maps.Clone(c.StatusMapping),
		StatusLabels:  maps.Clone(c.StatusLabels),
		NameToSKU:     maps.Clone(c.NameToSKU),
		DisplayNames:  maps.Clone(c.DisplayNames),
	}
	out.ensureMaps()
	return out
}

// ensureMaps replaces nil maps with empty ones.
func (c *Config) ensureMaps() {
	if c.StatusMapping == nil {
		c.StatusMapping = map[string]StatusClass{}
	}
	if c.StatusLabels == nil {
		c.StatusLabels = map[string]string{}
	}
	if c.NameToSKU == nil {
		c.NameToSKU = map[string]string{}
	}
	if c.DisplayNames == nil {
		c.DisplayNames = map[string]string{}
	}
}

// EnsureMaps replaces nil maps with empty ones (used by adapters after decoding).
func (c *Config) EnsureMaps() { c.ensureMaps() }
