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
			t.Fatalf("pctEncoded should have failed with %s", v)
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

func TestDecOctet(t *testing.T) {
	failSet := []string{
		"00",
		"05",
		"000",
		"001",
		"256",
		"300",
	}
	goodSet := []string{
		"0",
		"9",
		"12",
		"99",
		"100",
		"255",
	}

	for _, v := range failSet {
		p := newParser(v)
		p.next()
		if err := p.decOctet(); err == nil {
			t.Fatalf("decOctet should have failed with %s", v)
		}
	}

	for _, v := range goodSet {
		p := newParser(v)
		p.next()
		if err := p.decOctet(); err != nil {
			t.Fatalf("decOctet should succeed with '%s': %s", v, err.Error())
		}
	}
}
