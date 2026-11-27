package tools

import (
	"net/netip"
	"testing"
)

func TestPublicAddressPolicy(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.0.0.8", "169.254.1.2", "100.64.1.1", "::1", "fc00::1", "::ffff:192.168.1.1", "64:ff9b::1"} {
		if publicAddress(netip.MustParseAddr(s)) {
			t.Errorf("accepted %s", s)
		}
	}
	if !publicAddress(netip.MustParseAddr("8.8.8.8")) {
		t.Error("rejected public address")
	}
}
func TestConfiguredToolOrigin(t *testing.T) {
	t.Setenv("FACTORY_HTTP_ALLOWED_HOSTS", "api.example.com")
	if err := validateDestination("https://api.example.com/items/{id}", "https://api.example.com/items/42"); err != nil {
		t.Fatal(err)
	}
	if err := validateDestination("https://other.example/items", "https://other.example/items"); err == nil {
		t.Fatal("accepted unlisted host")
	}
}
