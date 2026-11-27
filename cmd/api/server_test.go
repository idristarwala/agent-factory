package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestAPIServerTimeoutSettings(t *testing.T) {
	for _, k := range []string{"FACTORY_API_READ_TIMEOUT", "FACTORY_API_WRITE_TIMEOUT", "FACTORY_API_IDLE_TIMEOUT"} {
		t.Setenv(k, "")
	}
	s, err := apiServer(nil)
	if err != nil || s.ReadTimeout != 15*time.Second || s.WriteTimeout != 30*time.Second || s.IdleTimeout != time.Minute {
		t.Fatal(s, err)
	}
	t.Setenv("FACTORY_API_READ_TIMEOUT", "0")
	t.Setenv("FACTORY_API_IDLE_TIMEOUT", "1m30s")
	s, err = apiServer(nil)
	if err != nil || s.ReadTimeout != 0 || s.IdleTimeout != 90*time.Second {
		t.Fatal(s, err)
	}
	for _, invalid := range []string{"-1s", "later", "5"} {
		t.Setenv("FACTORY_API_WRITE_TIMEOUT", invalid)
		if _, err := apiServer(nil); err == nil {
			t.Fatalf("accepted timeout %q", invalid)
		}
	}
}

func TestResponseHonorsConfiguredWriteTimeout(t *testing.T) {
	for _, tc := range []struct {
		timeout string
		success bool
	}{{"10ms", false}, {"2s", true}} {
		t.Run(tc.timeout, func(t *testing.T) {
			t.Setenv("FACTORY_API_WRITE_TIMEOUT", tc.timeout)
			srv, err := apiServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(60 * time.Millisecond)
				fmt.Fprint(w, "review ready")
			}))
			if err != nil {
				t.Fatal(err)
			}
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer srv.Close()
			go srv.Serve(ln)
			client := http.Client{Timeout: 3 * time.Second}
			res, err := client.Get("http://" + ln.Addr().String())
			if err != nil {
				if tc.success {
					t.Fatal(err)
				}
				return
			}
			defer res.Body.Close()
			data, err := io.ReadAll(res.Body)
			if (err == nil && string(data) == "review ready") != tc.success {
				t.Fatalf("response %q, error %v", data, err)
			}
		})
	}
}
