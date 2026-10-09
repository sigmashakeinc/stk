package main

import (
	"strings"
	"testing"
)

// --- itoa ---

func TestItoa(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{123, "123"},
		{1000000, "1000000"},
	}
	for _, tc := range cases {
		if got := itoa(tc.n); got != tc.want {
			t.Errorf("itoa(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

// --- stripCR ---

func TestStripCR(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no CR",
			input: []string{"hello", "world"},
			want:  []string{"hello", "world"},
		},
		{
			name:  "single CR keeps after",
			input: []string{"spin\rfinal"},
			want:  []string{"final"},
		},
		{
			name:  "multiple CRs keeps segment after last",
			input: []string{"a\rb\rc\rfinal"},
			want:  []string{"final"},
		},
		{
			name:  "trailing CR is a CRLF ending and is dropped",
			input: []string{"abc\r"},
			want:  []string{"abc"},
		},
		{
			name:  "CR at start",
			input: []string{"\rafter"},
			want:  []string{"after"},
		},
		{
			name:  "empty line",
			input: []string{""},
			want:  []string{""},
		},
	}
	for _, tc := range cases {
		got := stripCR(tc.input)
		if !sliceEq(got, tc.want) {
			t.Errorf("[%s] stripCR(%v) = %v, want %v", tc.name, tc.input, got, tc.want)
		}
	}
}

// --- collapseDuplicates ---

func TestCollapseDuplicates(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no duplicates",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "single element",
			input: []string{"a"},
			want:  []string{"a"},
		},
		{
			name:  "empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "all identical — 3 copies",
			input: []string{"x", "x", "x"},
			want:  []string{"x (×3)"},
		},
		{
			name:  "pair then single",
			input: []string{"a", "a", "b"},
			want:  []string{"a (×2)", "b"},
		},
		{
			name:  "single then pair",
			input: []string{"a", "b", "b"},
			want:  []string{"a", "b (×2)"},
		},
		{
			name:  "two pairs",
			input: []string{"a", "a", "b", "b"},
			want:  []string{"a (×2)", "b (×2)"},
		},
		{
			name:  "non-adjacent duplicates not collapsed",
			input: []string{"a", "b", "a"},
			want:  []string{"a", "b", "a"},
		},
		{
			name:  "five identical",
			input: []string{"z", "z", "z", "z", "z"},
			want:  []string{"z (×5)"},
		},
	}
	for _, tc := range cases {
		got := collapseDuplicates(tc.input)
		if !sliceEq(got, tc.want) {
			t.Errorf("[%s]\n  got  %v\n  want %v", tc.name, got, tc.want)
		}
	}
}

// --- collapseBlankRuns ---

func TestCollapseBlankRuns(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no blanks",
			input: []string{"a", "b"},
			want:  []string{"a", "b"},
		},
		{
			name:  "single blank preserved",
			input: []string{"a", "", "b"},
			want:  []string{"a", "", "b"},
		},
		{
			name:  "double blank collapsed",
			input: []string{"a", "", "", "b"},
			want:  []string{"a", "", "b"},
		},
		{
			name:  "five blanks collapsed to one",
			input: []string{"", "", "", "", ""},
			want:  []string{""},
		},
		{
			name:  "whitespace-only lines also blank",
			input: []string{"a", "   ", "\t", "b"},
			want:  []string{"a", "   ", "b"},
		},
		{
			name:  "leading blank runs",
			input: []string{"", "", "a"},
			want:  []string{"", "a"},
		},
	}
	for _, tc := range cases {
		got := collapseBlankRuns(tc.input)
		if !sliceEq(got, tc.want) {
			t.Errorf("[%s]\n  got  %v\n  want %v", tc.name, got, tc.want)
		}
	}
}

// --- truncate ---

func TestTruncate_NoOp(t *testing.T) {
	lines := makeLines(10, "line")
	got := truncate(lines, 200)
	if !sliceEq(got, lines) {
		t.Errorf("truncate should be no-op when len<=maxLines")
	}
}

func TestTruncate_ExactBoundary(t *testing.T) {
	lines := makeLines(200, "line")
	got := truncate(lines, 200)
	if !sliceEq(got, lines) {
		t.Errorf("truncate should be no-op at exactly maxLines")
	}
}

func TestTruncate_PreservesErrorLines(t *testing.T) {
	// Build 300 normal lines + 1 error line at position 150.
	lines := makeLines(300, "normal")
	lines[150] = "ERROR: something went wrong"
	got := truncate(lines, 50)

	// The error line must appear somewhere in output.
	found := false
	for _, l := range got {
		if l == "ERROR: something went wrong" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("truncate dropped an ERROR line; output:\n%s", strings.Join(got, "\n"))
	}
	if len(got) > 51 { // 50 + at most 1 marker
		t.Errorf("output too long: %d lines (max 51 incl. marker)", len(got))
	}
}

func TestTruncate_MarkerLinePresent(t *testing.T) {
	lines := makeLines(300, "line")
	got := truncate(lines, 50)
	marker := false
	for _, l := range got {
		if strings.Contains(l, "[stk] trimmed") {
			marker = true
			break
		}
	}
	if !marker {
		t.Errorf("truncate did not insert a [stk] trimmed marker; output:\n%s", strings.Join(got, "\n"))
	}
}

func TestTruncate_TrimCountAccurate(t *testing.T) {
	// 100 normal lines, maxLines=10 → budget=10, headN=6, tailN=4, trimmed=90.
	lines := makeLines(100, "n")
	got := truncate(lines, 10)
	for _, l := range got {
		if strings.Contains(l, "[stk] trimmed") {
			if !strings.Contains(l, "90") {
				t.Errorf("marker should say 90 trimmed, got: %q", l)
			}
			return
		}
	}
	t.Errorf("no trim marker found in output")
}

func TestTruncate_MultipleImportanceKeywords(t *testing.T) {
	lines := makeLines(300, "ok")
	lines[0] = "FATAL: crash"
	lines[100] = "warn: low disk"
	lines[200] = "exception in thread"
	got := truncate(lines, 20)
	preserved := map[string]bool{
		"FATAL: crash":        false,
		"warn: low disk":      false,
		"exception in thread": false,
	}
	for _, l := range got {
		if _, ok := preserved[l]; ok {
			preserved[l] = true
		}
	}
	for k, found := range preserved {
		if !found {
			t.Errorf("truncate dropped importance line: %q", k)
		}
	}
}

func TestTruncate_OnlyImportantLines(t *testing.T) {
	// All lines are important — none should be dropped.
	lines := []string{
		"error: a",
		"warn: b",
		"fatal: c",
	}
	got := truncate(lines, 2)
	// Should have a leading marker + the 3 important lines, or similar.
	errCount := 0
	for _, l := range got {
		if strings.HasPrefix(l, "error:") || strings.HasPrefix(l, "warn:") || strings.HasPrefix(l, "fatal:") {
			errCount++
		}
	}
	if errCount != 3 {
		t.Errorf("expected all 3 importance lines, got %d in: %v", errCount, got)
	}
}

// --- Reduce (integration) ---

func TestReduce_Empty(t *testing.T) {
	got := Reduce("", 200)
	if got != "" {
		t.Errorf("Reduce(\"\") = %q, want \"\"", got)
	}
}

func TestReduce_SmallInputNoOp(t *testing.T) {
	in := "hello\nworld\n"
	got := Reduce(in, 200)
	// Trailing newline creates a final empty element after Split; Join re-adds it.
	if got != in {
		t.Errorf("Reduce small input: got %q, want %q", got, in)
	}
}

func TestReduce_CRStrip(t *testing.T) {
	in := "spin1\rspin2\rfinal\nnext"
	got := Reduce(in, 200)
	if strings.Contains(got, "spin1") || strings.Contains(got, "spin2") {
		t.Errorf("CR-stripped segments should not appear: %q", got)
	}
	if !strings.Contains(got, "final") {
		t.Errorf("final segment after last CR should be kept: %q", got)
	}
}

func TestReduce_DupCollapse(t *testing.T) {
	in := "a\na\na\nb"
	got := Reduce(in, 200)
	if !strings.Contains(got, "a (×3)") {
		t.Errorf("expected dup collapse with ×3, got: %q", got)
	}
}

func TestReduce_BlankRunCollapse(t *testing.T) {
	// Use non-identical whitespace-only lines so dup-collapse (which requires
	// identical strings) leaves them alone; blank-run collapse then merges them.
	in := "a\n\n   \n\t\nb"
	got := Reduce(in, 200)
	// Should have exactly one blank/whitespace line between a and b.
	lines := strings.Split(got, "\n")
	blanks := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blanks++
		}
	}
	if blanks != 1 {
		t.Errorf("expected 1 blank line, got %d in: %q", blanks, got)
	}
}

func TestReduce_Truncation(t *testing.T) {
	// Use numbered lines so dup-collapse does not merge them; truncation fires.
	var sb strings.Builder
	for i := 0; i < 300; i++ {
		sb.WriteString("line " + itoa(i) + "\n")
	}
	got := Reduce(sb.String(), 50)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	// 50 content lines + 1 marker = 51 max.
	if len(lines) > 51 {
		t.Errorf("expected at most 51 lines after truncation, got %d", len(lines))
	}
	marker := false
	for _, l := range lines {
		if strings.Contains(l, "[stk] trimmed") {
			marker = true
		}
	}
	if !marker {
		t.Errorf("expected a [stk] trimmed marker in output")
	}
}

func TestReduce_ZeroMaxUsesDefault(t *testing.T) {
	// Use numbered lines so dup-collapse does not merge them.
	var sb strings.Builder
	for i := 0; i < 300; i++ {
		sb.WriteString("line " + itoa(i) + "\n")
	}
	// maxLines=0 should fall back to DefaultMaxLines (200).
	got := Reduce(sb.String(), 0)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > DefaultMaxLines+1 {
		t.Errorf("maxLines=0 fallback: got %d lines, expected ≤%d", len(lines), DefaultMaxLines+1)
	}
}

func TestReduce_ImportanceLinesPreservedAfterCRAndDupStages(t *testing.T) {
	// Error line after a CR on the same original line.
	in := "progress\rERROR: real error\nnormal\n"
	got := Reduce(in, 200)
	if !strings.Contains(got, "ERROR: real error") {
		t.Errorf("importance line lost through CR stage: %q", got)
	}
}

// --- helpers ---

func makeLines(n int, prefix string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = prefix
	}
	return out
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestStripCRCRLFLineEndings(t *testing.T) {
	got := Reduce("alpha\r\nbeta\r\n", 200)
	want := "alpha\nbeta\n"
	if got != want {
		t.Fatalf("CRLF input: got %q, want %q", got, want)
	}
	got = Reduce("10%\r50%\r100%\r\ndone\r\n", 200)
	want = "100%\ndone\n"
	if got != want {
		t.Fatalf("CRLF with progress: got %q, want %q", got, want)
	}
}
