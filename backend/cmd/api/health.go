package main

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	adminuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/admin"
)

// s3Pinger is the optional health surface of the object store (real *Client has
// it; the Disabled stub does not, so S3 reports "not configured").
type s3Pinger interface {
	Ping(ctx context.Context) error
}

// depChecker implements adminuc.DependencyChecker over the real backing clients.
// It lives in cmd/api (the composition root) where every client is in scope.
// Each probe is bounded and they run concurrently; unconfigured ones are skipped.
type depChecker struct {
	store     *storage.Store
	redis     *redis.Client // nil when REDIS_ADDR is empty
	objs      s3Pinger      // nil when object storage is the Disabled stub
	speechkit string        // host:port, "" unless STT_PROVIDER_KIND=speechkit
	smtp      string        // host:port
}

func (d *depChecker) Check(ctx context.Context) []adminuc.DepStatus {
	type probe struct {
		name       string
		configured bool
		run        func(context.Context) error // nil error == up
	}
	probes := []probe{
		{"Postgres", true, func(c context.Context) error { return d.store.Ping(c) }},
		{"Redis", d.redis != nil, func(c context.Context) error { return d.redis.Ping(c).Err() }},
		{"Object Storage", d.objs != nil, func(c context.Context) error { return d.objs.Ping(c) }},
		{"SpeechKit", d.speechkit != "", dialProbe(d.speechkit)},
		{"SMTP", d.smtp != "", dialProbe(d.smtp)},
	}

	out := make([]adminuc.DepStatus, len(probes))
	var wg sync.WaitGroup
	for i := range probes {
		wg.Add(1)
		go func(i int, p probe) {
			defer wg.Done()
			st := adminuc.DepStatus{Name: p.name, Configured: p.configured}
			if !p.configured {
				st.Detail = "не настроено"
				out[i] = st
				return
			}
			cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			start := time.Now()
			err := p.run(cctx)
			cancel()
			st.LatencyMS = time.Since(start).Milliseconds()
			st.Up = err == nil
			if err != nil {
				st.Detail = err.Error()
			}
			out[i] = st
		}(i, probes[i])
	}
	wg.Wait()
	return out
}

// dialProbe returns a TCP-reachability check for addr (host:port). Used for
// SpeechKit and SMTP, where a real protocol round-trip would need credentials.
func dialProbe(addr string) func(context.Context) error {
	return func(ctx context.Context) error {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return err
		}
		return conn.Close()
	}
}
