// Package model defines the shared data structures returned by mcmodlist core.
package model

// Loader identifies a mod loader or compatibility layer base.
type Loader string

const (
	LoaderFabric   Loader = "fabric"
	LoaderQuilt    Loader = "quilt"
	LoaderForge    Loader = "forge"
	LoaderNeoForge Loader = "neoforge"
	LoaderUnknown  Loader = ""
	LayerConnector        = "connector"
)

// ParseLoader normalises a loader name from launcher or jar metadata.
func ParseLoader(s string) Loader {
	switch s {
	case "fabric", "fabric-loader":
		return LoaderFabric
	case "quilt", "quilt-loader":
		return LoaderQuilt
	case "forge", "minecraftforge", "net.minecraftforge":
		return LoaderForge
	case "neoforge", "neo-forge", "net.neoforged":
		return LoaderNeoForge
	default:
		return LoaderUnknown
	}
}

// Valid reports whether l is a known loader.
func (l Loader) Valid() bool {
	switch l {
	case LoaderFabric, LoaderQuilt, LoaderForge, LoaderNeoForge:
		return true
	default:
		return false
	}
}

// Instance describes a detected Minecraft instance.
type Instance struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Launcher      string `json:"launcher"`
	Minecraft     string `json:"minecraft"`
	Loader        Loader `json:"loader"`
	LoaderVersion string `json:"loaderVersion"`
}

// Status describes how a mod entry should be treated.
type Status string

const (
	StatusEnabled  Status = "enabled"
	StatusDisabled Status = "disabled"
	StatusNested   Status = "nested"
)

// Mod describes one mod found in an instance.
type Mod struct {
	Name        string            `json:"name"`
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Authors     []string          `json:"authors"`
	Jar         string            `json:"jar"`
	Loader      Loader            `json:"loader"`
	Layer       string            `json:"layer,omitempty"`
	Side        string            `json:"side,omitempty"`
	Status      Status            `json:"status"`
	Recognized  bool              `json:"recognized"`
	Depends     []string          `json:"depends,omitempty"`
	Minecraft   string            `json:"minecraft,omitempty"`
	Change      string            `json:"change,omitempty"`
	OldVersion  string            `json:"oldVersion,omitempty"`
	Description string            `json:"description,omitempty"`
	License     string            `json:"license,omitempty"`
	Links       map[string]string `json:"links,omitempty"`
}

// Counts summarises mods by loader/layer label. Labels are lowercase and use
// "loader" or "loader/layer" forms such as "fabric" or "fabric/connector".
type Counts struct {
	Total        int            `json:"total"`
	Disabled     int            `json:"disabled"`
	Unrecognised int            `json:"unrecognised"`
	Loaders      map[string]int `json:"loaders"`
}

// NewCounts returns an empty Counts value.
func NewCounts() Counts {
	return Counts{Loaders: make(map[string]int)}
}

// Issue describes a diagnostic finding. Phase 1 only populates warnings.
type Issue struct {
	Type    string `json:"type"`
	Mod     string `json:"mod"`
	Message string `json:"message"`
}

// Warning describes a non-fatal problem encountered while scanning.
type Warning struct {
	Type    string `json:"type"`
	Mod     string `json:"mod"`
	Message string `json:"message"`
}
