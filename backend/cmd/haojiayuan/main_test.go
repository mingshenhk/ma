package main

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestTrustedProxies(t *testing.T) {
	got, err := trustedProxies(" 172.30.0.2/32, 10.0.0.7/24, , 2001:db8::1/64 ")
	if err != nil {
		t.Fatalf("trustedProxies returned error: %v", err)
	}
	want := []netip.Prefix{
		netip.MustParsePrefix("172.30.0.2/32"),
		netip.MustParsePrefix("10.0.0.0/24"),
		netip.MustParsePrefix("2001:db8::/64"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("trustedProxies() = %v, want %v", got, want)
	}
}

func TestTrustedProxiesEmpty(t *testing.T) {
	got, err := trustedProxies(" , ")
	if err != nil {
		t.Fatalf("trustedProxies returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("trustedProxies() = %v, want no trusted proxies", got)
	}
}

func TestTrustedProxiesRejectsInvalidPrefix(t *testing.T) {
	if _, err := trustedProxies("172.30.0.2, not-a-prefix"); err == nil {
		t.Fatal("trustedProxies accepted an invalid prefix")
	}
}
