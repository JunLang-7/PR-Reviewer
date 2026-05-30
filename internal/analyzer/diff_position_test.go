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
