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

func TestIpv4Address(t *testing.T) {
	failSet := []string{
		"1",
		"1.1",
		"1.1.1",
		"00.0.0.00",
		"256.0.0.1",
		"1234.0.0.1",
		"999.9.1.0",
	}
	goodSet := []string{
		"0.0.0.0",
		"1.1.1.1",
		"11.11.11.11",
		"255.255.255.255",
	}

	for _, v := range failSet {
		p := newParser(v)
		p.next()
		if err := p.ipv4Address(); err == nil {
			t.Fatalf("ipv4Address should have failed with %s", v)
		}
	}

	for _, v := range goodSet {
		p := newParser(v)
		p.next()
		if err := p.ipv4Address(); err != nil {
			t.Fatalf("ipv4Address should succeed with '%s': %s", v, err.Error())
		}
	}
}

func TestH16(t *testing.T) {
	/// No point for a fail set since this is mainly iterating over hexDigit
	goodSet := []string{
		"1",
		"a",
		"AA",
		"AAA",
		"10E4",
		"0000",
		"00000",
	}

	for _, v := range goodSet {
		p := newParser(v)
		p.next()
		if err := p.h16(); err != nil {
			t.Fatalf("h16 should succeed with '%s': %s", v, err.Error())
		}
	}
}

func TestLs32(t *testing.T) {
	failSet := []string{
		"DDDDD:11",
	}
	goodSet := []string{
		"1:1",
		"1:1:1",
		"34De:1",
		"1:ffff",
		"12:12:222:12",
	}

	for _, v := range failSet {
		p := newParser(v)
		p.next()
		if err := p.ls32(); err == nil {
			t.Fatalf("ls32 should have failed with %s", v)
		}
	}

	for _, v := range goodSet {
		p := newParser(v)
		p.next()
		if err := p.ls32(); err != nil {
			t.Fatalf("ls32 should succeed with '%s': %s", v, err.Error())
		}
	}
}

func TestIpv6Address(t *testing.T) {
	failSet := []string{":0db8:0000:0000:0000:ff00:0042:8329"}
	goodSet := []string{"2001:0db8:0000:0000:0000:ff00:0042:8329"}

	for _, v := range failSet {
		p := newParser(v)
		p.next()
		if err := p.ipv6Address(); err == nil {
			t.Fatalf("ipv6Address should have failed with %s", v)
		}
	}

	for _, v := range goodSet {
		p := newParser(v)
		p.next()
		if err := p.ipv6Address(); err != nil {
			t.Fatalf("ipv6Address should succeed with '%s': %s", v, err.Error())
		}
	}
}

func TestIpvFutures(t *testing.T) {
	failSet := []string{"7"}
	goodSet := []string{"v7.1-2"}

	for _, v := range failSet {
		p := newParser(v)
		p.next()
		if err := p.ipvFuture(); err == nil {
			t.Fatalf("ipvFuture should have failed with %s", v)
		}
	}

	for _, v := range goodSet {
		p := newParser(v)
		p.next()
		if err := p.ipvFuture(); err != nil {
			t.Fatalf("ipvFuture should succeed with '%s': %s", v, err.Error())
		}
	}
}
