package analyzer

import "testing"

func TestFileLineToPosition_Basic(t *testing.T) {
	hunk := `@@ -1,5 +1,6 @@
 package main
+import "fmt"
 func main() {
-    x := 1
+    x := 2
     fmt.Println(x)
 }`

	tests := []struct {
		targetLine int
		wantPos    int
	}{
		// line 1: package main → position 1
		{1, 1},
		// line 2: import "fmt" (new line) → position 2
		{2, 2},
		// line 3: func main() → position 3
		{3, 3},
		// line 4: x := 2 (new line, old x := 1 was line 4) → position 5
		{4, 5},
		// line 5: fmt.Println → position 6
		{5, 6},
		// line 6: } → position 7
		{6, 7},
	}

	for _, tt := range tests {
		got := FileLineToPosition(hunk, tt.targetLine)
		if got != tt.wantPos {
			t.Errorf("targetLine=%d: want position %d, got %d", tt.targetLine, tt.wantPos, got)
		}
	}
}

func TestFileLineToPosition_NotFound(t *testing.T) {
	hunk := `@@ -1,3 +1,3 @@
 line1
 line2
 line3`

	pos := FileLineToPosition(hunk, 99)
	if pos != 0 {
		t.Errorf("expected 0 for non-existent line, got %d", pos)
	}
}

func TestFileLineToPosition_FirstLine(t *testing.T) {
	hunk := `@@ -10,3 +10,3 @@
 line10
 line11
 line12`

	pos := FileLineToPosition(hunk, 10)
	if pos != 1 {
		t.Errorf("expected position 1 for first line, got %d", pos)
	}
}

func TestFileLineToPosition_NoNewlineMarker(t *testing.T) {
	patch := `@@ -1,3 +1,4 @@
 line1
 line2
+line3
 line4
\ No newline at end of file
@@ -10,3 +10,4 @@
 line10
+line11
 line12
 line13`

	// line 11 is in the second hunk. The "\ No newline" marker at position 5
	// should count toward position but NOT advance currentNewLine.
	// position sequence: line1(1) line2(2) +line3(3) line4(4) \(5) line10(6) +line11(7)
	pos := FileLineToPosition(patch, 11)
	if pos != 7 {
		t.Errorf("expected position 7 for line 11 with no-newline marker, got %d", pos)
	}

	// line 4: position should be 4 (marker doesn't affect position of this line)
	pos = FileLineToPosition(patch, 4)
	if pos != 4 {
		t.Errorf("expected position 4 for line 4, got %d", pos)
	}
}

func TestParseHunkNewStart(t *testing.T) {
	tests := []struct {
		header string
		want   int
	}{
		{"@@ -1,5 +1,6 @@", 1},
		{"@@ -10,3 +10,3 @@", 10},
		{"@@ -100 +200 @@", 200},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := parseHunkNewStart(tt.header)
		if got != tt.want {
			t.Errorf("parseHunkNewStart(%q) = %d, want %d", tt.header, got, tt.want)
		}
	}
}
