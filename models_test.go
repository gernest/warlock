package warlock

import (
	"testing"
)

func TestConfig_defaults(t *testing.T) {
	c := NewConfig(nil)
	if c.DB != "warlock.db" {
		t.Errorf("Expected warlock.db actual %s", c.DB)
	}

}

func TestConfig_custom(t *testing.T) {
	c := &Config{DB: "chaos.db"}
	cfg := NewConfig(c)

	if cfg.DB != c.DB {
		t.Errorf("Expected %s actual %s", c.DB, cfg.DB)
	}
	if cfg.SessMaxAge != 30 {
		t.Errorf("Expected 30 actual %d", cfg.SessMaxAge)
	}
}
