package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestAppendSavings_WritesLedgerLine verifies that a normal reduction call writes
// exactly one valid JSONL line with the correct tool/ts/orig_bytes/comp_bytes.
func TestAppendSavings_WritesLedgerLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SIGMASHAKE_HOME", dir)
	t.Setenv("STK_SAVINGS", "") // default-on

	appendSavings(1000, 200)

	data, err := os.ReadFile(filepath.Join(dir, "token-savings.jsonl"))
	if err != nil {
		t.Fatalf("expected ledger file to be written: %v", err)
	}
	// Strip trailing newline before unmarshalling.
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatalf("ledger line must end with \\n, got: %q", data)
	}
	var rec map[string]any
	if err := json.Unmarshal(data[:len(data)-1], &rec); err != nil {
		t.Fatalf("ledger line is not valid JSON: %v — raw: %s", err, data)
	}
	if got := rec["tool"]; got != "stk" {
		t.Errorf("tool = %q, want %q", got, "stk")
	}
	if orig, ok := rec["orig_bytes"].(float64); !ok || int(orig) != 1000 {
		t.Errorf("orig_bytes = %v, want 1000", rec["orig_bytes"])
	}
	if comp, ok := rec["comp_bytes"].(float64); !ok || int(comp) != 200 {
		t.Errorf("comp_bytes = %v, want 200", rec["comp_bytes"])
	}
	if _, ok := rec["ts"].(float64); !ok {
		t.Errorf("ts missing or not a number: %v", rec["ts"])
	}
}

// TestAppendSavings_OptOut_Zero verifies STK_SAVINGS=0 suppresses all writes.
func TestAppendSavings_OptOut_Zero(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SIGMASHAKE_HOME", dir)
	t.Setenv("STK_SAVINGS", "0")

	appendSavings(500, 100)

	if _, err := os.Stat(filepath.Join(dir, "token-savings.jsonl")); !os.IsNotExist(err) {
		t.Error("expected no ledger file with STK_SAVINGS=0")
	}
}

// TestAppendSavings_OptOut_False verifies STK_SAVINGS=false suppresses all writes.
func TestAppendSavings_OptOut_False(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SIGMASHAKE_HOME", dir)
	t.Setenv("STK_SAVINGS", "false")

	appendSavings(500, 100)

	if _, err := os.Stat(filepath.Join(dir, "token-savings.jsonl")); !os.IsNotExist(err) {
		t.Error("expected no ledger file with STK_SAVINGS=false")
	}
}

// TestAppendSavings_WriteFail_NonFatal verifies that an unwritable ledger path
// is silently discarded — no panic, no changed output, no exit.
func TestAppendSavings_WriteFail_NonFatal(t *testing.T) {
	dir := t.TempDir()
	// Occupy the ledger path with a directory so OpenFile fails.
	occupied := filepath.Join(dir, "token-savings.jsonl")
	if err := os.MkdirAll(occupied, 0o755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	t.Setenv("SIGMASHAKE_HOME", dir)
	t.Setenv("STK_SAVINGS", "")

	// Must not panic or call os.Exit.
	appendSavings(100, 50)
}

// TestAppendSavings_CompBytesIsEmittedLength verifies the contract: compBytes
// passed to appendSavings equals len(reduced) — the byte count of the string
// printed to stdout, matching how main.go calls it.
func TestAppendSavings_CompBytesIsEmittedLength(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SIGMASHAKE_HOME", dir)
	t.Setenv("STK_SAVINGS", "")

	input := "alpha\nbeta\ngamma\n"
	reduced := Reduce(input, DefaultMaxLines)
	origBytes := len(input)
	compBytes := len(reduced)

	appendSavings(origBytes, compBytes)

	data, err := os.ReadFile(filepath.Join(dir, "token-savings.jsonl"))
	if err != nil {
		t.Fatalf("ledger not written: %v", err)
	}
	var rec map[string]any
	if err := json.Unmarshal(data[:len(data)-1], &rec); err != nil {
		t.Fatalf("bad JSON: %v — raw: %s", err, data)
	}
	if got := int(rec["comp_bytes"].(float64)); got != compBytes {
		t.Errorf("comp_bytes = %d, want %d (len of reduced string)", got, compBytes)
	}
	if got := int(rec["orig_bytes"].(float64)); got != origBytes {
		t.Errorf("orig_bytes = %d, want %d", got, origBytes)
	}
}

// TestRun_Raw_NoLedger verifies that --raw passthrough never calls appendSavings
// (early return before the capture path is reached).
func TestRun_Raw_NoLedger(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SIGMASHAKE_HOME", dir)
	t.Setenv("STK_SAVINGS", "")
	t.Setenv("STK_DISABLE", "") // ensure not already set

	code := run([]string{"--raw", "echo", "hello"})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "token-savings.jsonl")); !os.IsNotExist(err) {
		t.Error("expected no ledger file in --raw mode")
	}
}

// TestRun_STKDisable_NoLedger verifies that STK_DISABLE=1 (raw passthrough)
// never calls appendSavings.
func TestRun_STKDisable_NoLedger(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SIGMASHAKE_HOME", dir)
	t.Setenv("STK_SAVINGS", "")
	t.Setenv("STK_DISABLE", "1")

	code := run([]string{"echo", "hello"})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "token-savings.jsonl")); !os.IsNotExist(err) {
		t.Error("expected no ledger file when STK_DISABLE=1")
	}
}
