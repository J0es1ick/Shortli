package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestCORSConcurrentRequests(t *testing.T) {
	h := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }), "https://shortli.test/")
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/api/me", nil)
			r.Header.Set("Origin", "https://shortli.test")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "https://shortli.test" {
				t.Error("CORS response changed")
			}
		}()
	}
	wg.Wait()
}
