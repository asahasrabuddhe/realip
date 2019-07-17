package realip

import (
	"fmt"
	"net/http"
	"testing"
)

func TestIsPrivateAddr(t *testing.T) {
	testData := map[string]bool{
		"127.0.0.0":   true,
		"10.0.0.0":    true,
		"169.254.0.0": true,
		"192.168.0.0": true,
		"::1":         true,
		"fc00::":      true,

		"172.15.0.0": false,
		"172.16.0.0": true,
		"172.31.0.0": true,
		"172.32.0.0": false,

		"147.12.56.11": false,
	}

	for addr, isLocal := range testData {
		isPrivate, err := isPrivateAddress(addr)
		if err != nil {
			t.Errorf("fail processing %s: %v", addr, err)
		}

		if isPrivate != isLocal {
			format := "%s should "
			if !isLocal {
				format += "not "
			}
			format += "be local address"

			t.Errorf(format, addr)
		}
	}
}

func TestRealIP(t *testing.T) {
	// Create type and function for testing
	type testIP struct {
		name     string
		request  *http.Request
		expected string
	}

	newRequest := func(remoteAddr, xRealIP string, forwarded bool, xForwardedFor ...string) *http.Request {
		h := http.Header{}
		if xRealIP != "" {
			h.Set("X-Real-IP", xRealIP)
		}
		if forwarded {
			for _, address := range xForwardedFor {
				h.Add("Forwarded", address)
			}
		} else {
			for _, address := range xForwardedFor {
				h.Add("X-Forwarded-For", address)
			}
		}
		return &http.Request{
			RemoteAddr: remoteAddr,
			Header:     h,
		}
	}

	// Create test data
	publicAddr1 := "144.12.54.87"
	publicAddr2 := "119.14.55.11"
	publicAddr3 := "13.182.55.11:8080"
	publicAddr3Wop := "13.182.55.11"
	localAddr := "127.0.0.0"

	testData := []testIP{
		{
			name:     "No header",
			request:  newRequest(publicAddr3, "", false),
			expected: publicAddr3Wop,
		}, {
			name:     "No header with port",
			request:  newRequest(publicAddr1, "", false),
			expected: publicAddr1,
		}, {
			name:     "Has X-Forwarded-For",
			request:  newRequest("", "", false, publicAddr1),
			expected: publicAddr1,
		}, {
			name:     "Has X-Forwarded-For multiple IPs (comma separated)",
			request:  newRequest("", "", false, fmt.Sprintf("%s,%s", localAddr, publicAddr1)),
			expected: publicAddr1,
		}, {
			name:     "Has X-Forwarded-For multiple IPs (comma and then space)",
			request:  newRequest("", "", false, fmt.Sprintf("%s, %s", localAddr, publicAddr1)),
			expected: publicAddr1,
		}, {
			name:     "Has multiple X-Forwarded-For",
			request:  newRequest("", "", false, localAddr, publicAddr1, publicAddr2),
			expected: publicAddr1,
		}, {
			name:     "Has multiple address for X-Forwarded-For",
			request:  newRequest("", "", false, localAddr, fmt.Sprintf("%s,%s", publicAddr1, publicAddr2)),
			expected: publicAddr1,
		}, {
			name:     "Has X-Real-IP",
			request:  newRequest("", "", false, publicAddr1),
			expected: publicAddr1,
		}, {
			name:     "Has Forwarded",
			request:  newRequest("", "", true, fmt.Sprintf("for=%s", publicAddr1)),
			expected: publicAddr1,
		}, {
			name:     "Has multiple addresses for Forwarded (comma separated)",
			request:  newRequest("", "", true, fmt.Sprintf("for=%s,for=%s", localAddr, publicAddr1)),
			expected: publicAddr1,
		}, {
			name:     "Has multiple addresses for Forwarded (comma and then space)",
			request:  newRequest("", "", true, fmt.Sprintf("for=%s, for=%s", localAddr, publicAddr2)),
			expected: publicAddr2,
		},
	}

	// Run test
	for _, v := range testData {
		if actual := FromRequest(v.request); v.expected != actual {
			t.Errorf("%s: expected %s but get %s", v.name, v.expected, actual)
		}
	}
}
