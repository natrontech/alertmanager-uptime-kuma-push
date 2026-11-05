package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
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
	pushBudget = 300 * time.Millisecond
	return gets
}

func answering(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }
}

func send(t *testing.T, method, body string) int {
	t.Helper()
	req := httptest.NewRequest(method, "/push", strings.NewReader(body))
	resp, err := newApp().Test(req, fiber.TestConfig{Timeout: 0})
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

func TestPushStaysInsideBudget(t *testing.T) {
	const budget = time.Second
	for _, tc := range []struct {
		name     string
		handler  http.HandlerFunc
		min, max time.Duration
	}{
		// No wait after the last attempt, so two of the three shares
		{"failing fast", answering(500), budget * 2 / 3, budget},
		{"never answering", func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() }, budget, budget * 3 / 2},
	} {
		gets := fakeKuma(t, 3, tc.handler)
		pushBudget = budget

		start := time.Now()
		got := send(t, http.MethodPost, watchdogBody)
		elapsed := time.Since(start)

		if got != 500 {
			t.Errorf("%s: answered %d, want 500", tc.name, got)
		}
		if gets.Load() != 3 {
			t.Errorf("%s: %d GETs, want 3", tc.name, gets.Load())
		}
		if elapsed < tc.min || elapsed > tc.max {
			t.Errorf("%s: took %v, want between %v and %v", tc.name, elapsed, tc.min, tc.max)
		}
	}
}

func TestPushOnlyForFiringWatchdog(t *testing.T) {
	for _, tc := range []struct {
		name, method, body string
		want               int
		gets               int32
	}{
		{"firing watchdog", http.MethodPost, watchdogBody, 200, 1},
		{"watchdog among others", http.MethodPost, `{"alerts":[{"status":"firing","labels":{"alertname":"Other"}},{"status":"firing","labels":{"alertname":"Watchdog"}}]}`, 200, 1},
		{"resolved watchdog", http.MethodPost, `{"alerts":[{"status":"resolved","labels":{"alertname":"Watchdog"}}]}`, 200, 0},
		{"other alert", http.MethodPost, `{"alerts":[{"status":"firing","labels":{"alertname":"Other"}}]}`, 200, 0},
		{"no alerts", http.MethodPost, `{"alerts":[]}`, 200, 0},
		{"invalid body", http.MethodPost, "not json", 400, 0},
		{"get", http.MethodGet, "", 405, 0},
	} {
		gets := fakeKuma(t, 1, answering(200))
		if got := send(t, tc.method, tc.body); got != tc.want {
			t.Errorf("%s: answered %d, want %d", tc.name, got, tc.want)
		}
		if got := gets.Load(); got != tc.gets {
			t.Errorf("%s: %d GETs, want %d", tc.name, got, tc.gets)
		}
	}
}

func TestPushTokenNotLogged(t *testing.T) {
	const token = "secret-push-token"
	fakeKuma(t, 1, func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	util.UptimeKumaURL += "/api/push/" + token

	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	if got := send(t, http.MethodPost, watchdogBody); got != 500 {
		t.Errorf("answered %d, want 500", got)
	}
	if strings.Contains(logs.String(), token) {
		t.Errorf("push token in logs:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "Client.Timeout") {
		t.Errorf("failure cause missing from logs:\n%s", logs.String())
	}
}
