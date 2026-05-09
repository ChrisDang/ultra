package supabase

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// PoolerResult holds the discovered pooler host and its reachability.
type PoolerResult struct {
	Host            string `json:"host"`
	TransactionPort int    `json:"transaction_port"` // 6543
	SessionPort     int    `json:"session_port"`     // 5432
	Reachable       bool   `json:"reachable"`
}

// DiscoverPooler probes candidate pooler hosts for the given region
// in parallel and returns the first that responds on port 6543.
func DiscoverPooler(ctx context.Context, region string) (*PoolerResult, error) {
	hosts := candidateHosts(region)

	type probeResult struct {
		host string
		ok   bool
	}

	ch := make(chan probeResult, len(hosts))
	var wg sync.WaitGroup

	for _, h := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			addr := fmt.Sprintf("%s:6543", host)
			ok, _ := probeHost(ctx, addr)
			ch <- probeResult{host: host, ok: ok}
		}(h)
	}

	// Close channel after all probes finish.
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Return the first successful probe.
	for r := range ch {
		if r.ok {
			return &PoolerResult{
				Host:            r.host,
				TransactionPort: 6543,
				SessionPort:     5432,
				Reachable:       true,
			}, nil
		}
	}

	return &PoolerResult{
		Reachable: false,
	}, fmt.Errorf("no pooler host reachable for region %s (tried: %v)", region, hosts)
}

// candidateHosts returns the pooler hostnames to probe for a given region.
func candidateHosts(region string) []string {
	return []string{
		fmt.Sprintf("aws-0-%s.pooler.supabase.com", region),
		fmt.Sprintf("aws-1-%s.pooler.supabase.com", region),
	}
}

// probeHost does a TCP dial to the given address with a 3-second timeout.
func probeHost(ctx context.Context, addr string) (bool, error) {
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}
