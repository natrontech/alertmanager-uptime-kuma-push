package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/natrontech/alertmanager-uptime-kuma-push/internal/util"
)

const watchdogBody = `{"status":"firing","alerts":[{"status":"firing","labels":{"alertname":"Watchdog"}}]}`

// fakeKuma points the pusher at a server that runs handler and counts GETs.
func fakeKuma(t *testing.T, retries int, handler http.HandlerFunc) *atomic.Int32 {
	t.Helper()
	gets := new(atomic.Int32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	util.UptimeKumaURL = srv.URL
	util.Retries = retries
	return gets
}

func answering(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }
}

func send(t *testing.T, method, body string) int {
	t.Helper()
	req := httptest.NewRequest(method, "/push", strings.NewReader(body))
	resp, err := newApp().Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestPushStatusFromKuma(t *testing.T) {
	for _, tc := range []struct {
		kuma, want int
		gets       int32
	}{
		{kuma: 200, want: 200, gets: 1},
		{kuma: 404, want: 500, gets: 2},
		{kuma: 500, want: 500, gets: 2},
	} {
		gets := fakeKuma(t, 2, answering(tc.kuma))
		if got := send(t, http.MethodPost, watchdogBody); got != tc.want {
			t.Errorf("kuma %d: answered %d, want %d", tc.kuma, got, tc.want)
		}
		if got := gets.Load(); got != tc.gets {
			t.Errorf("kuma %d: %d GETs, want %d", tc.kuma, got, tc.gets)
		}
	}
}
