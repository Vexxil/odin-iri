package odin_iri

import (
	"testing"
)

func TestPctEncoded(t *testing.T) {
	failSet := []string{
		"%",
		"1",
		"%z",
		"%1",
		"%D",
	}
	// Last 3 tests have extra HEXDIGIT which should not matter.
	goodSet := []string{
		"%00",
		"%1d",
		"%C4",
		"%EE",
		"%ae",
		"%129",
		"%122",
		"%ddd",
	}

	for _, v := range failSet {
		p := newParser(v)
		p.next()
		if err := p.pctEncoded(); err == nil {
			t.Fatal(err)
		}
	}

	for _, v := range goodSet {
		p := newParser(v)
		p.next()
		if err := p.pctEncoded(); err != nil {
			t.Fatalf("pctEncoded should succeed with '%s': %s", v, err.Error())
		}
	}
}
