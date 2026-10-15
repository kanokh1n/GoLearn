package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalOnly(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := localOnly(ok)

	cases := []struct {
		name    string
		method  string
		host    string
		headers map[string]string
		want    int
	}{
		{"page from localhost", "GET", "127.0.0.1:7070", nil, 200},
		{"json post from own page", "POST", "127.0.0.1:7070", map[string]string{"Content-Type": "application/json", "Origin": "http://127.0.0.1:7070"}, 200},
		{"dns rebinding host", "GET", "evil.example:7070", nil, 403},
		{"post from other site", "POST", "127.0.0.1:7070", map[string]string{"Content-Type": "application/json", "Origin": "https://evil.example"}, 403},
		{"form post without json", "POST", "127.0.0.1:7070", map[string]string{"Content-Type": "text/plain"}, 415},
		{"cross-site fetch metadata", "GET", "localhost:7070", map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/api/lesson/a/b/run", strings.NewReader("{}"))
			r.Host = tc.host
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Errorf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}
