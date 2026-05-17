package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAction_Kind(t *testing.T) {
	cases := map[Action]string{
		ActionServerRead:   "server",
		ActionApiKeyDelete: "api_key",
		Action("noversionsplit"): "noversionsplit",
	}
	for a, want := range cases {
		if got := a.Kind(); got != want {
			t.Errorf("Action(%q).Kind() = %q, want %q", a, got, want)
		}
	}
}

func TestScopeSet_HasAction(t *testing.T) {
	tests := []struct {
		name   string
		set    ScopeSet
		action Action
		want   bool
	}{
		{"wildcard matches anything", NewScopeSet("*"), ActionServerDelete, true},
		{"kind wildcard matches same kind", NewScopeSet("server:*"), ActionServerWrite, true},
		{"kind wildcard does not match other kind", NewScopeSet("server:*"), ActionApiKeyRead, false},
		{"exact match", NewScopeSet(string(ActionServerRead)), ActionServerRead, true},
		{"empty set authorizes nothing", NewScopeSet(), ActionServerRead, false},
		{"unrelated scope", NewScopeSet(string(ActionApiKeyRead)), ActionServerRead, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.set.HasAction(tc.action); got != tc.want {
				t.Errorf("HasAction(%q) = %v, want %v", tc.action, got, tc.want)
			}
		})
	}
}

func TestScopeSet_Intersect(t *testing.T) {
	a := NewScopeSet("server:read", "server:write", "api_key:read")
	b := NewScopeSet("server:read", "api_key:read")

	got := a.Intersect(b)
	want := NewScopeSet("server:read", "api_key:read")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Intersect = %v, want %v", got.Sorted(), want.Sorted())
	}
}

func TestScopeSet_Intersect_WildcardInOther(t *testing.T) {
	a := NewScopeSet("server:read", "api_key:write")
	b := NewScopeSet("*")
	got := a.Intersect(b)
	if !reflect.DeepEqual(got, a) {
		t.Errorf("intersect with wildcard should return original: got %v", got.Sorted())
	}
}

func TestScopeSet_Intersect_WildcardInSelf(t *testing.T) {
	a := NewScopeSet("*")
	b := NewScopeSet("server:read")
	got := a.Intersect(b)
	if _, ok := got["*"]; !ok {
		t.Errorf("intersect should preserve wildcard from self: %v", got.Sorted())
	}
}

func TestScopeSet_IsSubsetOf(t *testing.T) {
	tests := []struct {
		name  string
		s     ScopeSet
		other ScopeSet
		want  bool
	}{
		{"subset of wildcard", NewScopeSet("server:read"), NewScopeSet("*"), true},
		{"subset by kind wildcard", NewScopeSet("server:read", "server:write"), NewScopeSet("server:*"), true},
		{"not subset", NewScopeSet("server:read", "api_key:read"), NewScopeSet("server:*"), false},
		{"wildcard in self not subset of narrower", NewScopeSet("*"), NewScopeSet("server:*"), false},
		{"wildcard subset of wildcard", NewScopeSet("*"), NewScopeSet("*"), true},
		{"empty is subset", NewScopeSet(), NewScopeSet("server:read"), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.IsSubsetOf(tc.other); got != tc.want {
				t.Errorf("IsSubsetOf = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScopeSet_SortedAndEmpty(t *testing.T) {
	s := NewScopeSet("b", "a", "c")
	got := s.Sorted()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sorted = %v, want %v", got, want)
	}
	if s.Empty() {
		t.Error("non-empty set reported Empty()")
	}
	if !NewScopeSet().Empty() {
		t.Error("empty set should report Empty()")
	}
}

func TestScopeSet_ValueAndScan_Roundtrip(t *testing.T) {
	orig := NewScopeSet("server:read", "server:write")

	v, err := orig.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v != "server:read,server:write" {
		t.Errorf("Value = %q, want sorted CSV", v)
	}

	var dst ScopeSet
	if err := dst.Scan(v); err != nil {
		t.Fatalf("Scan(string): %v", err)
	}
	if !reflect.DeepEqual(dst, orig) {
		t.Errorf("Scan roundtrip mismatch: %v vs %v", dst.Sorted(), orig.Sorted())
	}

	var dst2 ScopeSet
	if err := dst2.Scan([]byte("a,b,c")); err != nil {
		t.Fatalf("Scan([]byte): %v", err)
	}
	if !reflect.DeepEqual(dst2, NewScopeSet("a", "b", "c")) {
		t.Errorf("Scan([]byte) mismatch: %v", dst2.Sorted())
	}

	var dst3 ScopeSet
	if err := dst3.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if dst3 == nil || len(dst3) != 0 {
		t.Errorf("Scan(nil) should yield non-nil empty set, got %v", dst3)
	}

	var dst4 ScopeSet
	if err := dst4.Scan(123); err == nil {
		t.Error("Scan(int) should error")
	}
}

func TestScopeSet_Value_Nil(t *testing.T) {
	var s ScopeSet
	v, err := s.Value()
	if err != nil {
		t.Fatalf("Value on nil: %v", err)
	}
	if v != "" {
		t.Errorf("nil Value should be empty string, got %q", v)
	}
}

func TestScopeSet_JSON_Roundtrip(t *testing.T) {
	orig := NewScopeSet("c", "a", "b")
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(b) != `["a","b","c"]` {
		t.Errorf("Marshal = %s, want sorted array", b)
	}

	var got ScopeSet
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("JSON roundtrip mismatch: %v vs %v", got.Sorted(), orig.Sorted())
	}

	// nil marshals to []
	var nilSet ScopeSet
	b, _ = json.Marshal(nilSet)
	if string(b) != "[]" {
		t.Errorf("nil marshal = %s, want []", b)
	}

	// blank entries trimmed
	var trimmed ScopeSet
	if err := json.Unmarshal([]byte(`["", "a", "  "]`), &trimmed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(trimmed, NewScopeSet("a")) {
		t.Errorf("Blank entries not trimmed: %v", trimmed.Sorted())
	}
}
