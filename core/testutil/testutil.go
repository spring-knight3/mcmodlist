// Package testutil provides hermetic helpers for building fixture files and jars.
package testutil

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// WriteFile writes data to path, creating parent directories.
func WriteFile(t testing.TB, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// WriteTextFile writes s to path.
func WriteTextFile(t testing.TB, path, s string) {
	t.Helper()
	WriteFile(t, path, []byte(s))
}

// TextEntries converts a name->text map to name->bytes.
func TextEntries(entries map[string]string) map[string][]byte {
	out := make(map[string][]byte, len(entries))
	for name, text := range entries {
		out[name] = []byte(text)
	}
	return out
}

// JarBytes builds an in-memory jar from entries.
func JarBytes(t testing.TB, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

// BuildJar writes entries as a jar file.
func BuildJar(t testing.TB, path string, entries map[string][]byte) {
	t.Helper()
	WriteFile(t, path, JarBytes(t, entries))
}

// BuildTextJar writes text entries as a jar file.
func BuildTextJar(t testing.TB, path string, entries map[string]string) {
	t.Helper()
	BuildJar(t, path, TextEntries(entries))
}
