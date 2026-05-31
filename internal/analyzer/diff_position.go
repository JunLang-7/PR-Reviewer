package analyzer

import (
	"strconv"
	"strings"
)

// FileLineToPosition converts a file line number (in the new version) to a
// diff position within a unified diff patch. Returns 0 if not found.
// Position counts ALL lines in the diff except @@ headers, matching GitHub's
// definition: "number of lines down from the first @@ hunk header."
func FileLineToPosition(patch string, targetLine int) int {
	if targetLine <= 0 {
		return 0
	}

	lines := strings.Split(patch, "\n")
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

		// GitHub counts every line in the diff (including empty separator
		// lines and "\ No newline" markers) toward position.
		position++

		// Only new-side lines advance the file line counter.
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
