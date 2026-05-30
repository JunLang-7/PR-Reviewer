package analyzer

import (
	"strconv"
	"strings"
)

// FileLineToPosition converts a file line number (in the new version) to a
// diff position within a single unified diff hunk. Returns 0 if not found.
func FileLineToPosition(hunk string, targetLine int) int {
	if targetLine <= 0 {
		return 0
	}

	lines := strings.Split(hunk, "\n")
	position := 0
	currentNewLine := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			newStart := parseHunkNewStart(line)
			if newStart > 0 {
				currentNewLine = newStart - 1
			}
			continue
		}

		// Skip empty lines and git metadata markers (e.g. "\ No newline at end of file")
		if line == "" || strings.HasPrefix(line, "\\") {
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
	}

	return 0
}

// parseHunkNewStart extracts the new file start line from a hunk header.
// Format: @@ -old_start,old_count +new_start,new_count @@
func parseHunkNewStart(header string) int {
	plusIdx := strings.Index(header, "+")
	if plusIdx < 0 {
		return 0
	}

	rest := header[plusIdx+1:]
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
