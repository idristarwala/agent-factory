package tools

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"
)

// Reject special-purpose ranges that IsGlobalUnicast alone would accept.
var specialNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
}

func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.Zone() != "" {
		return false
	}
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	return !slices.ContainsFunc(specialNetworks, func(p netip.Prefix) bool { return p.Contains(ip) })
}
func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	for _, ip := range addresses {
		if !publicAddress(ip) {
			return nil, fmt.Errorf("destination is outside the public network")
		}
	}
	d := net.Dialer{Timeout: 10 * time.Second}
	for _, ip := range addresses {
		// Connect to the checked address without a second name lookup.
		conn, dialErr := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
	}
	if err == nil {
		err = fmt.Errorf("destination resolved to no addresses")
	}
	return nil, err
}
func publicClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.DialContext = publicDial
	return &http.Client{Transport: t, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("HTTP tool redirects disabled") }}
}
func validateDestination(template, raw string) error {
	return validateToolURL(template, raw, true)
}

func validateToolURL(template, raw string, checkHostList bool) error {
	original, err := url.Parse(template)
	if err != nil {
		return err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host != original.Host || u.Scheme != original.Scheme {
		return fmt.Errorf("tool URL must have a fixed HTTP origin")
	}
	if hosts, configured := os.LookupEnv("FACTORY_HTTP_ALLOWED_HOSTS"); configured && checkHostList {
		allowed := strings.Split(hosts, ",")
		if !slices.ContainsFunc(allowed, func(h string) bool { return strings.EqualFold(strings.TrimSpace(h), u.Hostname()) }) {
			return fmt.Errorf("tool host is outside the configured allowlist")
		}
	}
	return nil
}
