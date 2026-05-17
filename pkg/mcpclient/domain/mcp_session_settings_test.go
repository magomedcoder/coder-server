package domain

import "testing"

func TestDefaultMCPSessionSettingsJSONCopy_IsIndependent(t *testing.T) {
	orig := append([]byte(nil), DefaultMCPSessionSettingsJSON...)
	cp := DefaultMCPSessionSettingsJSONCopy()
	if len(cp) == 0 {
		t.Fatalf("copy must not be empty")
	}

	cp[0] = '['
	if string(DefaultMCPSessionSettingsJSON) != string(orig) {
		t.Fatalf("default settings JSON must remain unchanged after copy mutation")
	}
}
