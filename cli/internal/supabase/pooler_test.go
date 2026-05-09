package supabase

import (
	"context"
	"net"
	"testing"
)

func TestCandidateHosts(t *testing.T) {
	hosts := candidateHosts("us-east-1")
	if len(hosts) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(hosts))
	}
	if hosts[0] != "aws-0-us-east-1.pooler.supabase.com" {
		t.Errorf("hosts[0] = %q, want %q", hosts[0], "aws-0-us-east-1.pooler.supabase.com")
	}
	if hosts[1] != "aws-1-us-east-1.pooler.supabase.com" {
		t.Errorf("hosts[1] = %q, want %q", hosts[1], "aws-1-us-east-1.pooler.supabase.com")
	}
}

func TestCandidateHosts_EURegion(t *testing.T) {
	hosts := candidateHosts("eu-west-1")
	if hosts[0] != "aws-0-eu-west-1.pooler.supabase.com" {
		t.Errorf("hosts[0] = %q, want %q", hosts[0], "aws-0-eu-west-1.pooler.supabase.com")
	}
}

func TestDiscoverPooler_WithLocalListener(t *testing.T) {
	// Start a local TCP listener to simulate a pooler.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	ctx := context.Background()

	// Probe the local listener directly.
	result, err := probeHost(ctx, addr)
	if err != nil {
		t.Fatalf("probeHost() error: %v", err)
	}
	if !result {
		t.Error("probeHost() = false, want true for listening address")
	}
}

func TestDiscoverPooler_UnreachableHost(t *testing.T) {
	ctx := context.Background()
	// RFC 5737 TEST-NET address — guaranteed unreachable.
	result, err := probeHost(ctx, "192.0.2.1:6543")
	if err == nil && result {
		t.Error("probeHost() should fail for unreachable address")
	}
}
