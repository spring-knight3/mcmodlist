package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type cliConfig struct {
	Format  string `toml:"format"`
	Columns string `toml:"columns"`
	Color   string `toml:"color"`
}

func loadConfig() (cliConfig, string) {
	var cfg cliConfig
	base, err := os.UserConfigDir()
	if err != nil {
		return cfg, ""
	}
	path := filepath.Join(base, "mcmodlist", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, ""
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cliConfig{}, fmt.Sprintf("warning: ignoring malformed config %s: %v", path, err)
	}
	var warnings []string
	cfg.Format = strings.ToLower(strings.TrimSpace(cfg.Format))
	if cfg.Format != "" && !formats[cfg.Format] {
		warnings = append(warnings, fmt.Sprintf("invalid format %q", cfg.Format))
		cfg.Format = ""
	}
	cfg.Color = strings.ToLower(strings.TrimSpace(cfg.Color))
	if cfg.Color != "" && cfg.Color != "auto" && cfg.Color != "always" && cfg.Color != "never" {
		warnings = append(warnings, fmt.Sprintf("invalid color %q", cfg.Color))
		cfg.Color = ""
	}
	cfg.Columns = strings.TrimSpace(cfg.Columns)
	if cfg.Columns != "" {
		if _, err := parseColumns(cfg.Columns); err != nil {
			warnings = append(warnings, fmt.Sprintf("invalid columns: %v", err))
			cfg.Columns = ""
		}
	}
	if len(warnings) > 0 {
		return cfg, fmt.Sprintf("warning: ignoring invalid config values in %s: %s", path, strings.Join(warnings, "; "))
	}
	return cfg, ""
}
