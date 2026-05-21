package runnerresolve

import "testing"

func TestResolve_prefersHealthyOverUnhealthyCandidate(t *testing.T) {
	entries := []Entry{
		{
			Address:   "a:1",
			Enabled:   true,
			Probed:    true,
			Connected: false,
		},
		{
			Address:   "b:2",
			Name:      "B",
			Enabled:   true,
			Probed:    true,
			Connected: true,
		},
	}
	available := AvailableAddresses(entries)

	sel := Resolve(available, entries, "a:1")
	if sel.Address != "b:2" {
		t.Fatalf("address: got %q want b:2", sel.Address)
	}

	if !sel.Connected {
		t.Fatal("expected connected runner")
	}
}

func TestResolve_skipsDisabledCandidate(t *testing.T) {
	entries := []Entry{
		{
			Address:   "a:1",
			Enabled:   false,
			Probed:    true,
			Connected: false,
		},
		{
			Address:   "b:2",
			Enabled:   true,
			Probed:    true,
			Connected: true,
		},
	}
	available := AvailableAddresses(entries)

	sel := Resolve(available, entries, "a:1")
	if sel.Address != "b:2" {
		t.Fatalf("address: got %q want b:2", sel.Address)
	}
}

func TestIsRunnable_withoutProbe(t *testing.T) {
	entries := []Entry{
		{
			Address: "a:1",
			Enabled: true,
			Probed:  false,
		},
	}

	if !IsRunnable("a:1", entries) {
		t.Fatal("enabled without probe should be runnable")
	}
}
