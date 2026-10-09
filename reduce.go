package main

import (
	"regexp"
	"strings"
)

// DefaultMaxLines is the default ceiling for Reduce output line count.
const DefaultMaxLines = 200

// importanceRe matches lines that must never be dropped during truncation.
var importanceRe = regexp.MustCompile(`(?i)(error|warn|fail|fatal|panic|denied|refus|exception|cannot|undefined)`)

// Reduce applies a deterministic, token-free pipeline to input and returns a
// condensed string whose line count is at most maxLines.
//
// Pipeline (in order):
//  1. CR strip   — for lines containing '\r', keep only the segment after the final '\r'.
//  2. Consecutive-duplicate collapse — a run of N identical lines (N>1) → one copy + " (×N)" suffix.
//  3. Blank-run collapse — consecutive blank/whitespace-only lines → one blank line.
//  4. Truncation — if count > maxLines, importance lines are always preserved;
//     remaining lines get a head+tail window with a single "[stk] trimmed <M> line(s)" marker.
func Reduce(input string, maxLines int) string {
	if input == "" {
		return ""
	}
	if maxLines <= 0 {
		maxLines = DefaultMaxLines
	}

	lines := strings.Split(input, "\n")
	lines = stripCR(lines)
	lines = collapseDuplicates(lines)
	lines = collapseBlankRuns(lines)
	lines = truncate(lines, maxLines)
	return strings.Join(lines, "\n")
}

// stripCR keeps only the segment after the last '\r' in each line.
// This handles progress-bar/spinner output that overwrites in a terminal.
func stripCR(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		// A CRLF line ending (Windows) leaves one trailing '\r'. Drop it
		// first, or the segment after the last '\r' would be empty.
		l = strings.TrimSuffix(l, "\r")
		if idx := strings.LastIndex(l, "\r"); idx >= 0 {
			l = l[idx+1:]
		}
		out = append(out, l)
	}
	return out
}

// collapseDuplicates collapses runs of N>1 identical consecutive lines into
// one copy of the line with a " (×N)" suffix.
func collapseDuplicates(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := make([]string, 0, len(lines))
	prev := lines[0]
	count := 1
	for _, l := range lines[1:] {
		if l == prev {
			count++
		} else {
			out = append(out, formatDup(prev, count))
			prev = l
			count = 1
		}
	}
	out = append(out, formatDup(prev, count))
	return out
}

func formatDup(line string, count int) string {
	if count > 1 {
		return line + " (×" + itoa(count) + ")"
	}
	return line
}

// collapseBlankRuns collapses consecutive blank/whitespace-only lines to a
// single blank line.
func collapseBlankRuns(lines []string) []string {
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, l := range lines {
		isBlank := strings.TrimSpace(l) == ""
		if isBlank && prevBlank {
			continue
		}
		out = append(out, l)
		prevBlank = isBlank
	}
	return out
}

// slot is an indexed line used during truncation merge.
type slot struct {
	origIdx int
	line    string
}

// truncate enforces the maxLines ceiling while never dropping lines matched by
// importanceRe. Non-important lines are kept as a head block (first ~2/3 of
// budget) and a tail block (last ~1/3), with a single marker line at the gap.
func truncate(lines []string, maxLines int) []string {
	if len(lines) <= maxLines {
		return lines
	}

	var important []slot
	var normal []slot
	for i, l := range lines {
		if importanceRe.MatchString(l) {
			important = append(important, slot{i, l})
		} else {
			normal = append(normal, slot{i, l})
		}
	}

	// If important lines alone fill or exceed the budget, return them with a
	// leading marker for the dropped normal lines (we never drop important ones).
	if len(important) >= maxLines {
		out := make([]string, 0, len(important)+1)
		trimmed := len(lines) - len(important)
		if trimmed > 0 {
			out = append(out, markerLine(trimmed))
		}
		for _, s := range important {
			out = append(out, s.line)
		}
		return out
	}

	// Budget of non-important lines we can keep.
	budget := maxLines - len(important)
	var keptNormal []slot
	trimCount := 0
	if len(normal) <= budget {
		keptNormal = normal
	} else {
		headN := budget * 2 / 3
		if headN < 1 {
			headN = 1
		}
		tailN := budget - headN
		trimCount = len(normal) - budget
		keptNormal = make([]slot, 0, budget)
		keptNormal = append(keptNormal, normal[:headN]...)
		if tailN > 0 {
			keptNormal = append(keptNormal, normal[len(normal)-tailN:]...)
		}
	}

	// Merge kept normal + important slots, sorted by original index.
	merged := make([]slot, 0, len(important)+len(keptNormal))
	merged = append(merged, important...)
	merged = append(merged, keptNormal...)
	sortSlots(merged)

	// Emit lines; insert the trimmed marker exactly once at the first index gap
	// that covers removed normal lines.
	out := make([]string, 0, len(merged)+1)
	markerInserted := false
	prevIdx := -1
	for _, s := range merged {
		if !markerInserted && trimCount > 0 && prevIdx >= 0 && s.origIdx > prevIdx+1 {
			out = append(out, markerLine(trimCount))
			markerInserted = true
		}
		out = append(out, s.line)
		prevIdx = s.origIdx
	}
	// Gap was at the tail (removed lines were at the end).
	if !markerInserted && trimCount > 0 {
		out = append(out, markerLine(trimCount))
	}
	return out
}

func markerLine(m int) string {
	return "… [stk] trimmed " + itoa(m) + " line(s) …"
}

// sortSlots sorts slots ascending by origIdx using insertion sort.
// Slices here are small (bounded by maxLines), so O(n²) is fine and avoids
// importing "sort".
func sortSlots(s []slot) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].origIdx < s[j-1].origIdx; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// itoa converts a non-negative int to a decimal string without "strconv".
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
