package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	var stderr bytes.Buffer
	cmd := newRootCommand(&stderr)
	cmd.SetOut(&stderr)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "--mode") {
		t.Fatalf("help output = %q, want --mode", stderr.String())
	}
}

func TestRootCommandValidation(t *testing.T) {
	var stderr bytes.Buffer
	cmd := newRootCommand(&stderr)
	cmd.SetArgs([]string{"--mode", "bad", "--prefix-len", "56", "--interface", "wan0", "--once"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--mode must be") {
		t.Fatalf("Execute() error = %v, want mode validation error", err)
	}
}

func TestRootCommandRejectsArgs(t *testing.T) {
	var stderr bytes.Buffer
	cmd := newRootCommand(&stderr)
	cmd.SetArgs([]string{"extra"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "accepts 0 arg") {
		t.Fatalf("Execute() error = %v, want positional arg error", err)
	}
}
