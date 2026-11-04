package util

import "testing"

func TestLoadEnvRetries(t *testing.T) {
	t.Setenv("UPTIME_KUMA_URL", "https://uptime-kuma.example.com/api/push/x")
	for _, tc := range []struct {
		value   string
		want    int
		wantErr bool
	}{
		{value: "", want: 3},
		{value: "1", want: 1},
		{value: "0", wantErr: true},
		{value: "-1", wantErr: true},
		{value: "x", wantErr: true},
	} {
		t.Setenv("GET_RETRIES", tc.value)
		err := LoadEnv()
		if (err != nil) != tc.wantErr {
			t.Errorf("GET_RETRIES=%q: err = %v, wantErr %v", tc.value, err, tc.wantErr)
		}
		if err == nil && Retries != tc.want {
			t.Errorf("GET_RETRIES=%q: Retries = %d, want %d", tc.value, Retries, tc.want)
		}
	}
}
