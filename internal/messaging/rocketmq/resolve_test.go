package rocketmq

import (
	"net"
	"strings"
	"testing"
)

// TestResolveAddrs_LiteralIPPassesThroughUnchanged covers the case that
// motivated this function: the official RocketMQ Go client's nameserver
// resolver rejects hostnames outright with ErrIllegalIP, so
// resolveAddrs must turn "rocketmq-namesrv:9876" (a Docker Compose
// service name) into a literal IP before it ever reaches that client.
func TestResolveAddrs_LiteralIPPassesThroughUnchanged(t *testing.T) {
	got, err := resolveAddrs([]string{"127.0.0.1:9876"})
	if err != nil {
		t.Fatalf("resolveAddrs failed: %v", err)
	}
	if len(got) != 1 || got[0] != "127.0.0.1:9876" {
		t.Errorf("expected a literal IP to pass through unchanged, got %v", got)
	}
}

func TestResolveAddrs_ResolvesHostname(t *testing.T) {
	got, err := resolveAddrs([]string{"localhost:9876"})
	if err != nil {
		t.Fatalf("resolveAddrs failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly one resolved address, got %v", got)
	}
	host, port, err := net.SplitHostPort(got[0])
	if err != nil {
		t.Fatalf("resolved address %q isn't a valid host:port: %v", got[0], err)
	}
	if net.ParseIP(host) == nil {
		t.Errorf("expected the hostname to resolve to a literal IP, got %q", host)
	}
	if port != "9876" {
		t.Errorf("expected the port to be preserved, got %q", port)
	}
}

func TestResolveAddrs_MultipleAddresses(t *testing.T) {
	got, err := resolveAddrs([]string{"127.0.0.1:9876", "localhost:9877"})
	if err != nil {
		t.Fatalf("resolveAddrs failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 resolved addresses, got %v", got)
	}
}

func TestResolveAddrs_MissingPort(t *testing.T) {
	_, err := resolveAddrs([]string{"127.0.0.1"})
	if err == nil {
		t.Error("expected an error for an address with no port")
	}
}

func TestResolveAddrs_UnresolvableHostname(t *testing.T) {
	_, err := resolveAddrs([]string{"this-host-should-not-resolve.invalid:9876"})
	if err == nil {
		t.Error("expected an error for an unresolvable hostname")
	}
	if !strings.Contains(err.Error(), "resolve rocketmq host") {
		t.Errorf("expected a descriptive resolve error, got %v", err)
	}
}

func TestResolveAddrs_EmptyInput(t *testing.T) {
	got, err := resolveAddrs(nil)
	if err != nil {
		t.Fatalf("resolveAddrs(nil) failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected an empty result for empty input, got %v", got)
	}
}
