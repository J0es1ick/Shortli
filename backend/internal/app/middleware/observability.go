package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type metricKey struct {
	Method  string
	Pattern string
	Status  int
}

type MetricsRegistry struct {
	startedAt time.Time
	mu        sync.RWMutex
	requests  map[metricKey]uint64
	durations map[metricKey]time.Duration
}

type ClickQueueMetrics struct {
	Pending  int64
	Queued   int64
	Recorded int64
	Retried  int64
}

func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		startedAt: time.Now(),
		requests:  make(map[metricKey]uint64),
		durations: make(map[metricKey]time.Duration),
	}
}

func (m *MetricsRegistry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		pattern := r.Pattern
		if pattern == "" {
			pattern = "unmatched"
		}
		key := metricKey{Method: r.Method, Pattern: pattern, Status: recorder.status}
		duration := time.Since(started)
		m.mu.Lock()
		m.requests[key]++
		m.durations[key] += duration
		m.mu.Unlock()
		log.Printf(
			"request_id=%s method=%s pattern=%q status=%d duration_ms=%d remote_ip=%s",
			requestID, r.Method, pattern, recorder.status, duration.Milliseconds(),
			GetClientIP(r, false),
		)
	})
}

func (m *MetricsRegistry) Handler(token string, clickStats func() ClickQueueMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		m.mu.RLock()
		keys := make([]metricKey, 0, len(m.requests))
		for key := range m.requests {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			left := keys[i].Method + keys[i].Pattern + strconv.Itoa(keys[i].Status)
			right := keys[j].Method + keys[j].Pattern + strconv.Itoa(keys[j].Status)
			return left < right
		})
		requests := make(map[metricKey]uint64, len(m.requests))
		durations := make(map[metricKey]time.Duration, len(m.durations))
		for _, key := range keys {
			requests[key] = m.requests[key]
			durations[key] = m.durations[key]
		}
		m.mu.RUnlock()

		fmt.Fprintln(w, "# TYPE shortli_http_requests_total counter")
		for _, key := range keys {
			fmt.Fprintf(w, "shortli_http_requests_total{method=%q,route=%q,status=%q} %d\n",
				key.Method, key.Pattern, strconv.Itoa(key.Status), requests[key])
		}
		fmt.Fprintln(w, "# TYPE shortli_http_request_duration_seconds_total counter")
		for _, key := range keys {
			fmt.Fprintf(w, "shortli_http_request_duration_seconds_total{method=%q,route=%q,status=%q} %.6f\n",
				key.Method, key.Pattern, strconv.Itoa(key.Status), durations[key].Seconds())
		}

		stats := clickStats()
		fmt.Fprintln(w, "# TYPE shortli_click_spool_pending gauge")
		fmt.Fprintf(w, "shortli_click_spool_pending %d\n", stats.Pending)
		fmt.Fprintln(w, "# TYPE shortli_click_events_queued_total counter")
		fmt.Fprintf(w, "shortli_click_events_queued_total %d\n", stats.Queued)
		fmt.Fprintln(w, "# TYPE shortli_click_events_recorded_total counter")
		fmt.Fprintf(w, "shortli_click_events_recorded_total %d\n", stats.Recorded)
		fmt.Fprintln(w, "# TYPE shortli_click_events_retried_total counter")
		fmt.Fprintf(w, "shortli_click_events_retried_total %d\n", stats.Retried)
		fmt.Fprintln(w, "# TYPE shortli_process_uptime_seconds gauge")
		fmt.Fprintf(w, "shortli_process_uptime_seconds %.0f\n", time.Since(m.startedAt).Seconds())
		fmt.Fprintln(w, "# TYPE shortli_go_goroutines gauge")
		fmt.Fprintf(w, "shortli_go_goroutines %d\n", runtime.NumGoroutine())
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func newRequestID() string {
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(value[:])
}
