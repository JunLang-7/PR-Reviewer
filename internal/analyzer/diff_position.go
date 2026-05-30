package analyzer

import (
	"strconv"
	"strings"
)

// FileLineToPosition converts a file line number (in the new version) to a
// diff position within a unified diff hunk. Returns 0 if not found in this hunk.
func FileLineToPosition(hunk string, targetLine int) int {
	lines := strings.Split(hunk, "\n")
	position := 0
	currentNewLine := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			newStart := parseHunkNewStart(line)
			if newStart > 0 {
				currentNewLine = newStart - 1 // will increment on first context/new line
			}
			continue
		}

		// Empty line or end
		if line == "" {
			continue
		}

		position++

		// Context and added lines count as new file lines
		if !strings.HasPrefix(line, "-") {
			currentNewLine++
			if currentNewLine == targetLine {
				return position
			}
		}
		// Deleted lines (-) don't increment currentNewLine
	}

	return 0
}

// parseHunkNewStart extracts the new file start line from a hunk header.
// Format: @@ -old_start,old_count +new_start,new_count @@
func parseHunkNewStart(header string) int {
	// Find "+" after the first "@@"
	plusIdx := strings.Index(header, "+")
	if plusIdx < 0 {
		return 0
	}

	rest := header[plusIdx+1:]
	// Extract number before comma or space
	commaIdx := strings.Index(rest, ",")
	endIdx := strings.Index(rest, " ")
	if commaIdx >= 0 {
		endIdx = commaIdx
	}
	if endIdx < 0 {
		endIdx = len(rest)
	}

	numStr := rest[:endIdx]
	n, err := strconv.Atoi(strings.TrimSpace(numStr))
	if err != nil {
		return 0
	}
	return n
}
