package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// savingsLedgerPath returns the path to the shared append-only token-savings
// ledger. Honors SIGMASHAKE_HOME env; falls back to
// os.UserHomeDir()+"/.sigmashake".
func savingsLedgerPath() string {
	if base := os.Getenv("SIGMASHAKE_HOME"); base != "" {
		return filepath.Join(base, "token-savings.jsonl")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".sigmashake", "token-savings.jsonl")
	}
	return filepath.Join(home, ".sigmashake", "token-savings.jsonl")
}

// appendSavings appends one reduction event to the shared token-savings ledger.
//
//   - origBytes: byte length of the original combined stdout+stderr captured
//     from the wrapped command.
//   - compBytes: byte length of the reduced output actually emitted to stdout.
//
// Best-effort: any I/O error is silently discarded — must not alter the emitted
// output, the exit code, or fail the command.
// Skipped when STK_SAVINGS is "0" or "false".
func appendSavings(origBytes, compBytes int) {
	switch os.Getenv("STK_SAVINGS") {
	case "0", "false":
		return
	}
	path := savingsLedgerPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755) // best-effort; ignore error
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return // non-fatal
	}
	defer f.Close()
	line := fmt.Sprintf(
		`{"tool":"stk","ts":%d,"orig_bytes":%d,"comp_bytes":%d}`+"\n",
		time.Now().Unix(), origBytes, compBytes,
	)
	_, _ = f.Write([]byte(line))
}
