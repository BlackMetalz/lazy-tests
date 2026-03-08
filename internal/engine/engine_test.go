package engine

import "testing"

func TestDrivers(t *testing.T) {
	eng := New()
	got := eng.Drivers()
	want := []string{"mysql", "postgres", "redis", "tcp"}
	if len(got) != len(want) {
		t.Fatalf("unexpected driver list length: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected driver at index %d: got %q want %q", i, got[i], want[i])
		}
	}
}
