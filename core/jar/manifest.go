package jar

import (
	"archive/zip"
	"bufio"
	"io"
	"strings"
)

const unknownVersion = "unknown"

func manifestAttributes(zr *zip.Reader) map[string]string {
	file := findFile(zr, "META-INF/MANIFEST.MF")
	if file == nil {
		return nil
	}
	rc, err := file.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()
	return parseManifest(rc)
}

func parseManifest(r io.Reader) map[string]string {
	attrs := make(map[string]string)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var lastKey string
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if lastKey != "" {
				attrs[lastKey] += strings.TrimLeft(line, " \t")
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			lastKey = ""
			continue
		}
		lastKey = strings.ToLower(strings.TrimSpace(key))
		attrs[lastKey] = strings.TrimSpace(value)
	}
	return attrs
}

func resolveVersion(version string, manifest map[string]string) string {
	if version == "" {
		return unknownVersion
	}
	if !strings.Contains(version, "${") {
		return version
	}
	resolved := replacePlaceholders(version, manifest)
	if resolved == "" || strings.Contains(resolved, "${") {
		return unknownVersion
	}
	return resolved
}

func replacePlaceholders(version string, manifest map[string]string) string {
	var b strings.Builder
	for {
		start := strings.Index(version, "${")
		if start < 0 {
			b.WriteString(version)
			return b.String()
		}
		endRel := strings.Index(version[start+2:], "}")
		if endRel < 0 {
			b.WriteString(version)
			return b.String()
		}
		end := start + 2 + endRel
		b.WriteString(version[:start])
		value := manifestValue(manifest, version[start+2:end])
		if value == "" {
			value = version[start : end+1]
		}
		b.WriteString(value)
		version = version[end+1:]
	}
}

func manifestValue(manifest map[string]string, placeholder string) string {
	key := strings.ToLower(strings.TrimSpace(placeholder))
	switch key {
	case "file.jarversion", "file.jar-version", "file.version":
		for _, candidate := range []string{"implementation-version", "jar-version", "version", "implementationversion"} {
			if value := manifest[candidate]; value != "" {
				return value
			}
		}
	case "file.name", "file.title":
		for _, candidate := range []string{"implementation-title", "name", "title"} {
			if value := manifest[candidate]; value != "" {
				return value
			}
		}
	}
	return ""
}
