// Command axon-load measures Axon's latency under concurrent use (EP-21).
// It sends a realistic mix of reads and writes with a personal access token
// and prints throughput and latency percentiles per endpoint:
//
//	AXON_TOKEN=axon_... go run ./cmd/axon-load -url http://localhost:3000/api -c 20 -d 60s
//
// Run it only against an Axon you operate: it creates tasks.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"slices"
	"sort"
	"sync"
	"time"
)

type request struct {
	name, method, path string
	weight             int
	body               func() []byte
}

var mix = []request{
	{"today", "GET", "/v1/dashboard", 20, nil},
	{"tasks", "GET", "/v1/tasks?view=next", 20, nil},
	{"notes", "GET", "/v1/notes", 15, nil},
	{"search", "GET", "/v1/search?q=review", 10, nil},
	{"overview", "GET", "/v1/dashboard/overview", 10, nil},
	{"insights", "GET", "/v1/insights?weeks=12", 5, nil},
	{"habits", "GET", "/v1/habits", 10, nil},
	{"create task", "POST", "/v1/tasks", 10, func() []byte {
		return []byte(fmt.Sprintf(`{"title":"Load test %d","status":"inbox"}`, rand.IntN(1_000_000))) //nolint:gosec // load-test traffic, not security
	}},
}

type result struct {
	name string
	took time.Duration
	ok   bool
}

func main() {
	base := flag.String("url", "http://localhost:3000/api", "API base URL")
	conc := flag.Int("c", 10, "concurrent clients")
	dur := flag.Duration("d", 30*time.Second, "duration")
	flag.Parse()
	token := os.Getenv("AXON_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "set AXON_TOKEN to a write-scoped personal access token")
		os.Exit(2)
	}
	total := 0
	for _, r := range mix {
		total += r.weight
	}
	pick := func() request {
		n := rand.IntN(total) //nolint:gosec // load-test traffic, not security
		for _, r := range mix {
			if n < r.weight {
				return r
			}
			n -= r.weight
		}
		return mix[0]
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: *conc}}
	ctx, cancel := context.WithTimeout(context.Background(), *dur)
	defer cancel()
	results := make(chan result, 1024)
	var wg sync.WaitGroup
	for range *conc {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				r := pick()
				var body io.Reader
				if r.body != nil {
					body = bytes.NewReader(r.body())
				}
				req, _ := http.NewRequestWithContext(ctx, r.method, *base+r.path, body)
				req.Header.Set("Authorization", "Bearer "+token)
				if body != nil {
					req.Header.Set("Content-Type", "application/json")
				}
				start := time.Now()
				resp, err := client.Do(req)
				ok := err == nil && resp.StatusCode < 400
				if resp != nil {
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}
				if ctx.Err() != nil {
					return
				}
				results <- result{r.name, time.Since(start), ok}
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()
	by := map[string][]time.Duration{}
	errs := map[string]int{}
	var all []time.Duration
	for r := range results {
		by[r.name] = append(by[r.name], r.took)
		all = append(all, r.took)
		if !r.ok {
			errs[r.name]++
		}
	}
	pct := func(d []time.Duration, p float64) time.Duration {
		if len(d) == 0 {
			return 0
		}
		slices.Sort(d)
		return d[min(len(d)-1, int(float64(len(d))*p))]
	}
	fmt.Printf("%d clients for %v: %d requests, %.0f req/s\n\n", *conc, *dur, len(all), float64(len(all))/dur.Seconds())
	fmt.Printf("%-12s %8s %8s %8s %8s %7s\n", "endpoint", "count", "p50", "p95", "p99", "errors")
	names := make([]string, 0, len(by))
	for n := range by {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range append(names, "all") {
		d, e := by[n], errs[n]
		if n == "all" {
			d = all
			for _, v := range errs {
				e += v
			}
		}
		fmt.Printf("%-12s %8d %8v %8v %8v %7d\n", n, len(d), pct(d, .5).Round(time.Millisecond), pct(d, .95).Round(time.Millisecond), pct(d, .99).Round(time.Millisecond), e)
	}
}
