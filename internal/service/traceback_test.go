package service

import "testing"

func TestParseTracebackRust(t *testing.T) {
	stderr := `   Compiling foo v0.1.0
error[E0425]: cannot find value 'x' in this scope
 --> src/main.rs:2:5
`
	got := ParseTraceback(stderr)
	if got == "" || !contains(got, "E0425") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestParseTracebackPython(t *testing.T) {
	stderr := `Traceback (most recent call last):
  File "main.py", line 3, in <module>
    foo()
ZeroDivisionError: division by zero
`
	got := ParseTraceback(stderr)
	if got == "" {
		t.Fatal("expected python error")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
