/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
