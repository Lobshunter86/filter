package filter

import (
	"net/http"
	"testing"
	"time"

	"github.com/caddyserver/caddy"
)

func TestSetup(t *testing.T) {
	c := caddy.NewTestController("dns", `filter`)
	if err := setup(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `filter hello`)
	if err := setup(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}
}

func TestIp2Interval(t *testing.T) {
	ip, mask := "1.0.2.0", "23"

	ipInfo := IP2IPInterval(ip, mask)
	t.Log(ipInfo)
}

func TestUpdateLocalIP(t *testing.T) {
	// negligible data race
	duration := 2 * time.Second
	f := Filter{}

	t.Log(f.localIP)

	go f.updateLocalIP(duration)
	time.Sleep(duration * 2)

	t.Log(f.localIP)
}

func TestDebug(t *testing.T) {
	resp, err := http.Get("https://api.ipify.org?format=json")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}
