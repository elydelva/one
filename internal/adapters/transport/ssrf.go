package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// ErrBlockedHost is returned when a request targets a host that resolves to a
// blocked (private/loopback/link-local) IP.
type ErrBlockedHost struct {
	Host   string
	Reason string
}

func (e ErrBlockedHost) Error() string {
	return fmt.Sprintf("blocked host %q: %s", e.Host, e.Reason)
}

// guardingDialer wraps the dial step to refuse private/loopback/link-local IPs.
// Allowed hosts (typically "127.0.0.1" for tests) bypass the check.
type guardingDialer struct {
	inner    *net.Dialer
	allowed  map[string]struct{}
	resolver func(ctx context.Context, host string) ([]net.IPAddr, error)
}

func newGuardingDialer(allowed []string) *guardingDialer {
	set := map[string]struct{}{}
	for _, h := range allowed {
		set[h] = struct{}{}
	}
	d := &net.Dialer{}
	return &guardingDialer{
		inner:    d,
		allowed:  set,
		resolver: (&net.Resolver{}).LookupIPAddr,
	}
}

func (g *guardingDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if err := g.checkHost(ctx, host); err != nil {
		return nil, err
	}
	return g.inner.DialContext(ctx, network, addr)
}

func (g *guardingDialer) checkHost(ctx context.Context, host string) error {
	if _, ok := g.allowed[host]; ok {
		return nil
	}
	// Fast path: host is already a literal IP.
	if ip := net.ParseIP(host); ip != nil {
		return guardIP(host, ip)
	}
	addrs, err := g.resolver(ctx, host)
	if err != nil {
		return err
	}
	for _, a := range addrs {
		if err := guardIP(host, a.IP); err != nil {
			return err
		}
	}
	return nil
}

func guardIP(host string, ip net.IP) error {
	if ip.IsLoopback() {
		return ErrBlockedHost{Host: host, Reason: "loopback IP " + ip.String()}
	}
	if ip.IsPrivate() {
		return ErrBlockedHost{Host: host, Reason: "private IP " + ip.String()}
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return ErrBlockedHost{Host: host, Reason: "link-local IP " + ip.String()}
	}
	if ip.IsUnspecified() {
		return ErrBlockedHost{Host: host, Reason: "unspecified IP"}
	}
	// AWS instance metadata (covered by IsLinkLocal, but explicit for clarity).
	if ip.Equal(net.IPv4(169, 254, 169, 254)) {
		return ErrBlockedHost{Host: host, Reason: "cloud metadata endpoint"}
	}
	return nil
}

// checkRedirect returns a CheckRedirect function that caps redirects and
// re-validates each Location.
func checkRedirect(cap int, allowHTTP bool) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= cap {
			return errors.New("too many redirects (cap " + fmt.Sprint(cap) + ")")
		}
		if req.URL.Scheme != "https" && !allowHTTP {
			return errors.New("redirect to non-HTTPS URL refused: " + req.URL.String())
		}
		return nil
	}
}
