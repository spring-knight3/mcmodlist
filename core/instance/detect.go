// Package instance detects Minecraft instances and reads launcher metadata.
package instance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mcmodlist/core/model"
)

// Kind identifies why instance detection failed.
type Kind int

const (
	// KindNotDirectory means the path is missing or is not a directory.
	KindNotDirectory Kind = iota
	// KindUnreadable means the path could not be read.
	KindUnreadable
	// KindNotInstance means the path is a directory but not a Minecraft instance.
	KindNotInstance
	// KindModsFolder means the path looks like a mods folder, not an instance.
	KindModsFolder
)

// Error is a typed instance detection failure.
type Error struct {
	Kind       Kind
	Path       string
	Suggestion string
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.message(), e.Err)
	}
	return e.message()
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) message() string {
	switch e.Kind {
	case KindNotDirectory:
		return fmt.Sprintf("not a directory: %s", e.Path)
	case KindUnreadable:
		return fmt.Sprintf("cannot read instance path %s", e.Path)
	case KindModsFolder:
		return fmt.Sprintf("%s looks like a mods folder, not an instance", e.Path)
	default:
		return fmt.Sprintf("%s is not a Minecraft instance", e.Path)
	}
}

// IsModsFolder reports whether err is a mods-folder detection error.
func IsModsFolder(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == KindModsFolder
}

// Detection describes a detected instance.
type Detection struct {
	Root    string
	ModsDir string
	Info    model.Instance
}

// Detect validates root as a Minecraft instance and returns its mods directory
// plus any launcher metadata that could be read.
func Detect(root string) (Detection, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Detection{}, &Error{Kind: KindUnreadable, Path: root, Err: err}
	}
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return Detection{}, &Error{Kind: KindNotDirectory, Path: abs}
		}
		return Detection{}, &Error{Kind: KindUnreadable, Path: abs, Err: err}
	}
	if !st.IsDir() {
		return Detection{}, &Error{Kind: KindNotDirectory, Path: abs}
	}

	if looksLikeModsFolder(abs) {
		return Detection{}, &Error{Kind: KindModsFolder, Path: abs, Suggestion: filepath.Dir(abs)}
	}

	det := Detection{Root: abs}
	det.ModsDir = findModsDir(abs)
	det.Info = readLauncherMetadata(abs)
	if det.Info.Name == "" {
		det.Info.Name = filepath.Base(abs)
	}
	det.Info.Path = abs

	if det.ModsDir == "" && det.Info.Launcher == "" {
		return Detection{}, &Error{Kind: KindNotInstance, Path: abs}
	}
	return det, nil
}

func looksLikeModsFolder(path string) bool {
	if filepath.Base(path) != "mods" {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".jar") || strings.HasSuffix(name, ".jar.disabled") {
			return true
		}
	}
	return false
}

func findModsDir(root string) string {
	for _, candidate := range []string{filepath.Join(root, "mods"), filepath.Join(root, ".minecraft", "mods")} {
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
	}
	return ""
}
