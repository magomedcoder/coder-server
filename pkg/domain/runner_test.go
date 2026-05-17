package domain

import "testing"

func TestParseRunnerHostOrHostPort(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		fallback int32
		wantHost string
		wantPort int32
		wantErr  bool
	}{
		{"10.0.0.1", 50052, "10.0.0.1", 50052, false},
		{"10.0.0.1:9", 1, "10.0.0.1", 9, false},
		{"example.org:443", 1, "example.org", 443, false},
		{"[::1]:50052", 1, "::1", 50052, false},
		{"[::1]", 50052, "::1", 50052, false},
		{"localhost", 0, "", 0, true},
		{"", 50052, "", 0, true},
	}
	for _, tc := range cases {
		h, p, err := ParseRunnerHostOrHostPort(tc.in, tc.fallback)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q: expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if h != tc.wantHost || p != tc.wantPort {
			t.Fatalf("%q: got %q:%d want %q:%d", tc.in, h, p, tc.wantHost, tc.wantPort)
		}
	}
}
