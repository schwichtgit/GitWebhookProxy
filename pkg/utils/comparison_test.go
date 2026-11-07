package utils

import "testing"

func TestInArray(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		array     interface{}
		value     interface{}
		wantFound bool
		wantIndex int
	}{
		{name: "string found", array: []string{"alice", "bob"}, value: "bob", wantFound: true, wantIndex: 1},
		{name: "string not found", array: []string{"alice", "bob"}, value: "carol", wantFound: false, wantIndex: -1},
		{name: "empty string is not matched by default", array: []string{"alice"}, value: "", wantFound: false, wantIndex: -1},
		{name: "empty slice", array: []string{}, value: "alice", wantFound: false, wantIndex: -1},
		{name: "nil slice", array: []string(nil), value: "alice", wantFound: false, wantIndex: -1},
		{name: "int found", array: []int{1, 2, 3}, value: 3, wantFound: true, wantIndex: 2},
		{name: "type mismatch", array: []int{1, 2, 3}, value: "1", wantFound: false, wantIndex: -1},
		{name: "first match wins", array: []string{"a", "b", "a"}, value: "a", wantFound: true, wantIndex: 0},
		{name: "not a slice", array: "alice", value: "alice", wantFound: false, wantIndex: -1},
		{name: "map is not searched", array: map[string]int{"alice": 1}, value: "alice", wantFound: false, wantIndex: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found, index := InArray(tt.array, tt.value)
			if found != tt.wantFound || index != tt.wantIndex {
				t.Errorf("InArray() = (%v, %d), want (%v, %d)", found, index, tt.wantFound, tt.wantIndex)
			}
		})
	}
}
