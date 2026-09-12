// Command loadtest (composable-runtime-roadmap.md 17j) is a minimal,
// dependency-free concurrent HTTP load driver -- this repo has no
// wrk/vegeta/k6/hey anywhere (confirmed before writing this), and adding
// one would be a new external tool dependency this project's own "Infer
// Before Configure" principle would rather avoid without a forcing case.
// stdlib net/http, a worker pool, and a sorted-latency-slice percentile
// calculation are enough for this benchmark's own honest scope: real
// per-request latency against a real (throwaway) server, not a
// general-purpose load-testing tool.
//
// Usage:
//
//	go run ./scripts/loadtest -url http://localhost:4099/ws_default/mch_approval_document \
//	    -cookiejar /path/to/alice.jar -concurrency 10 -requests 200
//
// -cookiejar reads a Netscape-format cookie jar (exactly what `curl -c`
// already writes, and what conformance/lib.sh's own session_for already
// produces via the same login flow) into a real net/http/cookiejar.Jar,
// so every real cookie the login response set (menata_session and any
// other) is sent back correctly -- not a single hand-extracted
// name=value guess.
//
// Prints one JSON object to stdout: request count, status-code counts,
// and min/p50/p95/p99/max latency in milliseconds.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type result struct {
	statusCode int
	latencyMs  float64
}

func main() {
	targetURL := flag.String("url", "", "URL to load-test (required)")
	cookieJarPath := flag.String("cookiejar", "", "path to a Netscape-format cookie jar file (as written by curl -c)")
	concurrency := flag.Int("concurrency", 10, "number of concurrent worker goroutines")
	requests := flag.Int("requests", 100, "total number of requests to send")
	rate := flag.Float64("rate", 0, "max aggregate requests/second across all workers (0 = unlimited, fires as fast as possible)")
	flag.Parse()

	if *targetURL == "" {
		fmt.Fprintln(os.Stderr, "loadtest: -url is required")
		os.Exit(1)
	}

	jar, err := loadCookieJar(*cookieJarPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadtest: loading cookie jar: %v\n", err)
		os.Exit(1)
	}
	client := &http.Client{Timeout: 10 * time.Second, Jar: jar}

	// -rate (composable-runtime-roadmap.md 17j): this server's own
	// per-IP rate limiter (cmd/server/ratelimit.go, 30 req/s burst 120)
	// otherwise dominates the measured latency with 429s that never ran
	// the composable pipeline at all -- pacing dispatch, not just firing
	// as fast as possible, is what a real client population actually
	// looks like anyway.
	var ticker *time.Ticker
	if *rate > 0 {
		ticker = time.NewTicker(time.Duration(float64(time.Second) / *rate))
		defer ticker.Stop()
	}

	jobs := make(chan struct{}, *requests)
	for i := 0; i < *requests; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	results := make([]result, 0, *requests)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				if ticker != nil {
					<-ticker.C
				}
				res := doRequest(client, *targetURL)
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	printSummary(results)
}

// loadCookieJar parses a Netscape-format cookie file (curl's own -c/-b
// format: tab-separated domain/flag/path/secure/expiration/name/value,
// an httponly cookie's domain prefixed with "#HttpOnly_", other lines
// starting with "#" are comments) into a real cookiejar.Jar keyed by
// domain, so http.Client sends every real cookie back automatically.
// Empty path returns an empty (but non-nil) jar -- an anonymous request.
func loadCookieJar(path string) (http.CookieJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return jar, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	byDomain := make(map[string][]*http.Cookie)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#HttpOnly_") {
			line = strings.TrimPrefix(line, "#HttpOnly_")
		} else if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			continue
		}
		domain := strings.TrimPrefix(fields[0], ".")
		name, value := fields[5], fields[6]
		byDomain[domain] = append(byDomain[domain], &http.Cookie{Name: name, Value: value})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for domain, cookies := range byDomain {
		if u, err := url.Parse("http://" + domain); err == nil {
			jar.SetCookies(u, cookies)
		}
	}
	return jar, nil
}

func doRequest(client *http.Client, url string) result {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return result{statusCode: 0, latencyMs: 0}
	}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return result{statusCode: 0, latencyMs: float64(latency.Microseconds()) / 1000}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return result{statusCode: resp.StatusCode, latencyMs: float64(latency.Microseconds()) / 1000}
}

// summary is the one JSON object printed to stdout -- percentiles
// computed over a sorted copy of every real measured latency, not
// estimated or fabricated. LatencyMsAll mixes every response regardless
// of status; LatencyMs2xx isolates successful responses only, since a
// rejected request (a 429 from the server's own per-IP rate limiter,
// short-circuited before any real work, or a 403 permission deny) has a
// fundamentally different latency shape than one that actually ran the
// composable pipeline -- mixing them would misrepresent "how fast is a
// real request," not just report it imprecisely.
type summary struct {
	Requests     int                `json:"requests"`
	StatusCodes  map[string]int     `json:"status_codes"`
	LatencyMsAll map[string]float64 `json:"latency_ms_all"`
	LatencyMs2xx map[string]float64 `json:"latency_ms_2xx"`
}

func printSummary(results []result) {
	statusCodes := make(map[string]int)
	all := make([]float64, 0, len(results))
	ok2xx := make([]float64, 0, len(results))
	for _, r := range results {
		statusCodes[fmt.Sprintf("%d", r.statusCode)]++
		all = append(all, r.latencyMs)
		if r.statusCode >= 200 && r.statusCode < 300 {
			ok2xx = append(ok2xx, r.latencyMs)
		}
	}
	sort.Float64s(all)
	sort.Float64s(ok2xx)

	s := summary{
		Requests:     len(results),
		StatusCodes:  statusCodes,
		LatencyMsAll: percentiles(all),
		LatencyMs2xx: percentiles(ok2xx),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(s)
}

func percentiles(sorted []float64) map[string]float64 {
	return map[string]float64{
		"min": percentile(sorted, 0),
		"p50": percentile(sorted, 50),
		"p95": percentile(sorted, 95),
		"p99": percentile(sorted, 99),
		"max": percentile(sorted, 100),
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p / 100 * float64(len(sorted)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
