package composable_test

import (
	"testing"

	"menata.id/app/internal/composable"
)

func TestNodeIdentityStringDeterministic(t *testing.T) {
	a := composable.NodeIdentity{Kind: "dataset", Source: "mch_x", Params: map[string]string{"b": "2", "a": "1"}}
	b := composable.NodeIdentity{Kind: "dataset", Source: "mch_x", Params: map[string]string{"a": "1", "b": "2"}}
	if a.String() != b.String() {
		t.Fatalf("String() not stable across map insertion order: %q vs %q", a.String(), b.String())
	}
}

func TestNodeIdentityStringDistinguishesSource(t *testing.T) {
	a := composable.NodeIdentity{Kind: "page", Source: "mch_x"}
	b := composable.NodeIdentity{Kind: "page", Source: "mch_y"}
	if a.String() == b.String() {
		t.Fatalf("different Source produced equal identity string %q", a.String())
	}
}

func TestNodeIdentityStringDistinguishesParams(t *testing.T) {
	a := composable.NodeIdentity{Kind: "component", Source: "mch_x", Params: map[string]string{"view": "vw_1"}}
	b := composable.NodeIdentity{Kind: "component", Source: "mch_x", Params: map[string]string{"view": "vw_2"}}
	if a.String() == b.String() {
		t.Fatalf("different Params produced equal identity string %q", a.String())
	}
}
