package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	line, err := readLine(bufio.NewReader(strings.NewReader("{\"v\":1}\n{\"v\":1,\"id\":2}\n")))
	if err != nil || string(line) != "{\"v\":1}\n" {
		t.Fatalf("first line: %q %v", line, err)
	}
	line, err = readLine(bufio.NewReader(strings.NewReader("{\"v\":1,\"id\":2}\n")))
	if err != nil || string(line) != "{\"v\":1,\"id\":2}\n" {
		t.Fatalf("second line: %q %v", line, err)
	}
}

func TestReadLineOversized(t *testing.T) {
	big := append(bytes.Repeat([]byte("x"), maxLine+10), '\n')
	_, err := readLine(bufio.NewReader(bytes.NewReader(big)))
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversized error, got %v", err)
	}
}

func TestReadLineEOF(t *testing.T) {
	line, err := readLine(bufio.NewReader(strings.NewReader("{\"v\":1}")))
	if err == nil || string(line) != "{\"v\":1}" {
		t.Fatalf("expected EOF with data, got %q %v", line, err)
	}
}
