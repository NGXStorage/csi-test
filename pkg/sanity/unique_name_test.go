package sanity

import (
	"strings"
	"testing"
)

func TestUniqueStringGeneratesDistinctNames(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name := UniqueString("sanity-list-volume")
		if !strings.HasPrefix(name, "sanity-list-volume-") || seen[name] {
			t.Fatalf("invalid or duplicate name %q", name)
		}
		seen[name] = true
	}
}

func TestUniqueStringWithLength(t *testing.T) {
	first := UniqueStringWithLength("snapshot", MaxNameLength)
	second := UniqueStringWithLength("snapshot", MaxNameLength)
	if len(first) != MaxNameLength || len(second) != MaxNameLength || first == second {
		t.Fatal("expected distinct names with the requested length")
	}
}
