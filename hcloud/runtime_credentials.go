package hcloud

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/http/httpguts"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
)

const invalidAuthorizationTokenError = "authorization token contains invalid characters"

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newHCloudHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: newHCloudCredentialReloader(nil),
	}
}

func newRobotHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: newRobotCredentialReloader(nil),
	}
}

func newHCloudCredentialReloader(next http.RoundTripper) http.RoundTripper {
	next = transportOrDefault(next)

	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		token, err := config.LookupHCloudToken()
		if err != nil {
			return nil, err
		}
		if token != "" && !httpguts.ValidHeaderFieldValue(token) {
			return nil, fmt.Errorf(invalidAuthorizationTokenError)
		}

		cloned := cloneRequest(req)
		if token == "" {
			cloned.Header.Del("Authorization")
		} else {
			cloned.Header.Set("Authorization", "Bearer "+token)
		}
		return next.RoundTrip(cloned)
	})
}

func newRobotCredentialReloader(next http.RoundTripper) http.RoundTripper {
	next = transportOrDefault(next)

	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		user, password, err := config.LookupRobotCredentials()
		if err != nil {
			return nil, err
		}

		cloned := cloneRequest(req)
		if user == "" && password == "" {
			cloned.Header.Del("Authorization")
		} else {
			cloned.SetBasicAuth(user, password)
		}
		return next.RoundTrip(cloned)
	})
}

func cloneRequest(req *http.Request) *http.Request {
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	return cloned
}

func transportOrDefault(next http.RoundTripper) http.RoundTripper {
	if next != nil {
		return next
	}
	return http.DefaultTransport
}
