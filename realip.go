package realip

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

// Should use canonical format of the header key s
// https://golang.org/pkg/net/http/#CanonicalHeaderKey
var xForwardedForHeader = http.CanonicalHeaderKey("X-Forwarded-For")
var xRealIpHeader = http.CanonicalHeaderKey("X-Real-IP")

// RFC7239 defines a new "Forwarded: " header designed to replace the
// existing use of X-Forwarded-* headers.
// e.g. Forwarded: for=192.0.2.60;proto=https;by=203.0.113.43
var forwardedHeader = http.CanonicalHeaderKey("Forwarded")

var cidrs []*net.IPNet

func init() {
	maxCidrBlocks := []string{
		"127.0.0.1/8",    // localhost
		"10.0.0.0/8",     // 24-bit block
		"172.16.0.0/12",  // 20-bit block
		"192.168.0.0/16", // 16-bit block
		"169.254.0.0/16", // link local address
		"::1/128",        // localhost IPv6
		"fc00::/7",       // unique local address IPv6
		"fe80::/10",      // link local address IPv6
	}

	cidrs = make([]*net.IPNet, len(maxCidrBlocks))
	for i, maxCidrBlock := range maxCidrBlocks {
		_, cidr, _ := net.ParseCIDR(maxCidrBlock)
		cidrs[i] = cidr
	}
}

// isLocalAddress works by checking if the address is under private CIDR blocks.
// List of private CIDR blocks can be seen on :
//
// https://en.wikipedia.org/wiki/Private_network
//
// https://en.wikipedia.org/wiki/Link-local_address
func isPrivateAddress(address string) (bool, error) {
	ipAddress := net.ParseIP(address)
	if ipAddress == nil {
		return false, errors.New("address is not valid")
	}

	for i := range cidrs {
		if cidrs[i].Contains(ipAddress) {
			return true, nil
		}
	}

	return false, nil
}

// FromRequest returns client's real public IP address from http request headers.
func FromRequest(r *http.Request) string {
	// Fetch header value
	xRealIP := r.Header.Get(xRealIpHeader)
	xForwardedFor := r.Header[xForwardedForHeader]
	forwarded := r.Header.Get(forwardedHeader)

	// If both empty, return IP from remote address
	if xRealIP == "" && len(xForwardedFor) == 0 && forwarded == "" {
		var remoteIP string

		// If there are colon in remote address, remove the port number
		// otherwise, return remote address as is
		if strings.ContainsRune(r.RemoteAddr, ':') {
			remoteIP, _, _ = net.SplitHostPort(r.RemoteAddr)
		} else {
			remoteIP = r.RemoteAddr
		}

		return remoteIP
	}

	// Check list of IP in X-Forwarded-For and return the first global address
	for _, a := range xForwardedFor {
		for _, b := range strings.Split(a, ",") {
			address := strings.TrimSpace(b)
			isPrivate, err := isPrivateAddress(address)
			if !isPrivate && err == nil {
				return address
			}
		}
	}

	// Check list of IPs in the new Forwarded header and return the first global address
	for _, a := range strings.Split(forwarded, ";") {
		for _, b := range strings.Split(a, ",") {
			if strings.Contains(b, "for") {
				c := strings.Split(b, "=")
				if len(c) == 2 {
					address := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(c[1]), `"[`), `]"`)
					isPrivate, err := isPrivateAddress(address)
					if !isPrivate && err == nil {
						return address
					}
				}
			}
		}
	}

	// If nothing succeed, return X-Real-IP
	return xRealIP
}

// RealIP return client's real public IP address from http request headers.
//
// Deprecated: Use FromRequest instead.
func RealIP(r *http.Request) string {
	return FromRequest(r)
}
