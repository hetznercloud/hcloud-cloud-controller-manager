package hcloud

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/http/httpguts"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
)

const invalidAuthorizationTokenError = "authorization token contains invalid characters"
const credentialsReloadDebounce = 100 * time.Millisecond

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type runtimeCredentials struct {
	mu sync.RWMutex

	hcloudToken string
	robotUser   string
	robotPass   string

	hcloudTokenPath string
	robotUserPath   string
	robotPassPath   string

	watcher   *fsnotify.Watcher
	closeOnce sync.Once
}

func newRuntimeCredentials() (*runtimeCredentials, error) {
	credentials := &runtimeCredentials{}

	if err := credentials.loadInitial(); err != nil {
		return nil, err
	}

	files := config.LookupRuntimeCredentialFiles()
	credentials.hcloudTokenPath = files.HCloudToken
	credentials.robotUserPath = files.RobotUser
	credentials.robotPassPath = files.RobotPassword

	if !files.HasAnyFilePaths() {
		return credentials, nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, dir := range files.Directories() {
		if err := watcher.Add(dir); err != nil {
			addErr := fmt.Errorf("watch credentials directory %q: %w", dir, err)
			if closeErr := watcher.Close(); closeErr != nil {
				return nil, errors.Join(addErr, fmt.Errorf("close credentials watcher: %w", closeErr))
			}
			return nil, addErr
		}
	}

	credentials.watcher = watcher
	go credentials.watch()
	return credentials, nil
}

func (c *runtimeCredentials) loadInitial() error {
	token, err := config.LookupHCloudToken()
	if err != nil {
		return err
	}
	if token != "" && !httpguts.ValidHeaderFieldValue(token) {
		return errors.New(invalidAuthorizationTokenError)
	}

	user, password, err := config.LookupRobotCredentials()
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.hcloudToken = token
	c.robotUser = user
	c.robotPass = password
	c.mu.Unlock()

	return nil
}

func (c *runtimeCredentials) watch() {
	var debounceTimer *time.Timer
	var debounceC <-chan time.Time

	stopDebounce := func() {
		if debounceTimer == nil {
			return
		}
		if !debounceTimer.Stop() {
			select {
			case <-debounceTimer.C:
			default:
			}
		}
	}

	defer stopDebounce()

	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				// !ok means the watcher closed the events channel.
				return
			}
			if !shouldReload(event) {
				continue
			}
			if debounceTimer == nil {
				debounceTimer = time.NewTimer(credentialsReloadDebounce)
				debounceC = debounceTimer.C
				continue
			}
			stopDebounce()
			debounceTimer.Reset(credentialsReloadDebounce)
		case err, ok := <-c.watcher.Errors:
			if !ok {
				// !ok means the watcher closed the errors channel.
				return
			}
			klog.ErrorS(err, "watching mounted credential files")
		case <-debounceC:
			debounceC = nil
			c.reload()
		}
	}
}

func shouldReload(event fsnotify.Event) bool {
	return event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename|fsnotify.Remove|fsnotify.Chmod) != 0
}

func (c *runtimeCredentials) reload() {
	if c.hcloudTokenPath != "" {
		token, err := config.ReadCredentialFile(c.hcloudTokenPath)
		switch {
		case err != nil:
			klog.ErrorS(err, "reloading HCLOUD_TOKEN from mounted secret")
		case token != "" && !httpguts.ValidHeaderFieldValue(token):
			klog.ErrorS(errors.New(invalidAuthorizationTokenError), "reloading HCLOUD_TOKEN from mounted secret")
		default:
			c.mu.Lock()
			c.hcloudToken = token
			c.mu.Unlock()
		}
	}

	if c.robotUserPath != "" || c.robotPassPath != "" {
		user, password, err := c.loadRobotCredentials()
		if err != nil {
			klog.ErrorS(err, "reloading Robot credentials from mounted secret")
			return
		}
		c.mu.Lock()
		c.robotUser = user
		c.robotPass = password
		c.mu.Unlock()
	}
}

func (c *runtimeCredentials) loadRobotCredentials() (string, string, error) {
	c.mu.RLock()
	user := c.robotUser
	password := c.robotPass
	c.mu.RUnlock()

	var err error
	if c.robotUserPath != "" {
		user, err = config.ReadCredentialFile(c.robotUserPath)
		if err != nil {
			return "", "", err
		}
	}
	if c.robotPassPath != "" {
		password, err = config.ReadCredentialFile(c.robotPassPath)
		if err != nil {
			return "", "", err
		}
	}
	if (user == "") != (password == "") {
		return "", "", fmt.Errorf("both %q and %q must be provided, or neither", "ROBOT_USER", "ROBOT_PASSWORD")
	}
	return user, password, nil
}

func (c *runtimeCredentials) close() error {
	var err error
	c.closeOnce.Do(func() {
		if c.watcher != nil {
			err = c.watcher.Close()
		}
	})
	return err
}

func (c *runtimeCredentials) hcloudAuthorization() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.hcloudToken == "" {
		return ""
	}
	return "Bearer " + c.hcloudToken
}

func (c *runtimeCredentials) robotCredentials() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.robotUser, c.robotPass
}

func newHCloudHTTPClient(timeout time.Duration, credentials *runtimeCredentials) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: newHCloudCredentialReloader(credentials, nil),
	}
}

func newRobotHTTPClient(timeout time.Duration, credentials *runtimeCredentials) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: newRobotCredentialReloader(credentials, nil),
	}
}

func newHCloudCredentialReloader(credentials *runtimeCredentials, next http.RoundTripper) http.RoundTripper {
	next = transportOrDefault(next)

	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		cloned := cloneRequest(req)
		auth := credentials.hcloudAuthorization()
		if auth == "" {
			cloned.Header.Del("Authorization")
		} else {
			cloned.Header.Set("Authorization", auth)
		}
		return next.RoundTrip(cloned)
	})
}

func newRobotCredentialReloader(credentials *runtimeCredentials, next http.RoundTripper) http.RoundTripper {
	next = transportOrDefault(next)

	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		cloned := cloneRequest(req)
		user, password := credentials.robotCredentials()
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
