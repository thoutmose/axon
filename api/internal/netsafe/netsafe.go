// Package netsafe makes HTTP clients for addresses people give Axon (a
// calendar to follow, a page to clip): unless the operator allows private
// networks (AXON_CALENDAR_PRIVATE=1), they refuse to connect to private,
// loopback, link-local, and CGNAT addresses, whatever the name resolves
// to, so an address cannot point Axon at its own network.
package netsafe

import (
	"errors"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
)

// ErrPrivate is returned when an address is on a private network.
var ErrPrivate = errors.New("that address is on a private network; the operator can allow it with AXON_CALENDAR_PRIVATE=1")

// AllowPrivate says whether the operator allowed private networks.
func AllowPrivate() bool { return os.Getenv("AXON_CALENDAR_PRIVATE") == "1" }

var cgnat = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// Private reports whether ip is not on the public internet.
func Private(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() || cgnat.Contains(ip)
}

// Client returns a client that checks every connection, redirects included,
// and follows at most three redirects, to HTTPS only unless private
// networks are allowed.
func Client(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: nil,
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
				Control: func(_, address string, _ syscall.RawConn) error {
					host, _, err := net.SplitHostPort(address)
					if err != nil {
						return err
					}
					if ip := net.ParseIP(host); ip != nil && Private(ip) && !AllowPrivate() {
						return ErrPrivate
					}
					return nil
				},
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "https" && !AllowPrivate() {
				return errors.New("the page redirected to an address that is not HTTPS")
			}
			return nil
		},
	}
}
