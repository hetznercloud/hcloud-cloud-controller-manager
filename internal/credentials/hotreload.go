package credentials

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	fsnotify "github.com/fsnotify/fsnotify"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	robotclient "github.com/syself/hetzner-cloud-controller-manager/internal/robot/client"
	"k8s.io/klog/v2"
)

var (
	// fsnotify creates several events for a single update of a mounted secret.
	// To avoid multiple reloads, we store the old values and only reload when
	// the values have changed.
	oldRobotUser     string
	oldRobotPassword string
	oldHcloudToken   string

	// robotReloadCounter gets incremented when the credentials get reloaded.
	// Mosty used for testing.
	robotReloadCounter uint64

	// hcloudTokenReloadCounter gets incremented when the credentials get reloaded.
	// Mosty used for testing.
	hcloudTokenReloadCounter uint64

	hcloudMutex sync.Mutex
	robotMutex  sync.Mutex
)

// GetRobotReloadCounter returns the number of times the robot credentials have been reloaded.
// Mostly used for testing.
func GetRobotReloadCounter() uint64 {
	robotMutex.Lock()
	defer robotMutex.Unlock()
	return robotReloadCounter
}

// GetHcloudReloadCounter returns the number of times the hcloud credentials have been reloaded.
// Mostly used for testing.
func GetHcloudReloadCounter() uint64 {
	hcloudMutex.Lock()
	defer hcloudMutex.Unlock()
	return hcloudTokenReloadCounter
}

// Watch the mounted secrets. Reload the credentials, when the files get updated. The robotClient can be nil.
func Watch(credentialsDir string, hcloudClient *hcloud.Client, robotClient robotclient.Client) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Fatal(err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if !isValidEvent(event) {
					continue
				}

				// get last element of path. Example: /etc/hetzner-secret/robot-user -> robot-user
				baseName := filepath.Base(event.Name)

				var err error
				switch baseName {
				case "robot-user":
					err = loadRobotCredentials(credentialsDir, robotClient)
				case "robot-password":
					err = loadRobotCredentials(credentialsDir, robotClient)
				case "hcloud":
					err = loadHcloudCredentials(credentialsDir, hcloudClient)
				case "..data":
					// The files (for example hcloud) are symlinks to ..data/.
					// For example the file "hcloud" is a symlink to ../data/hcloud
					// This means the files/symlinks don't change. When the secrets get changed, then
					// a new ..data directory gets created. This is done by Kubernetes to make the
					// update of all files atomic.
					err = loadHcloudCredentials(credentialsDir, hcloudClient)
					if robotClient != nil {
						err = errors.Join(err, loadRobotCredentials(credentialsDir, robotClient))
					}
				default:
					klog.Infof("Ignoring fsnotify event for file %q: %s", baseName, event.String())
				}
				if err != nil {
					klog.Errorf("error processing fsnotify event: %s", err.Error())
					continue
				}

			case err := <-watcher.Errors:
				klog.Infof("error: %s", err)
			}
		}
	}()

	err = watcher.Add(credentialsDir)
	if err != nil {
		return fmt.Errorf("watcher.Add: %w", err)
	}
	return nil
}

func isValidEvent(event fsnotify.Event) bool {
	baseName := filepath.Base(event.Name)
	if strings.HasPrefix(baseName, "..") && baseName != "..data" {
		// Skip ..data_tmp and ..YYYY_MM_DD...
		return false
	}
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
		return true
	}
	return false
}

func loadRobotCredentials(credentialsDir string, robotClient robotclient.Client) error {
	robotMutex.Lock()
	defer robotMutex.Unlock()

	username, password, err := readRobotCredentials(credentialsDir)
	if err != nil {
		return fmt.Errorf("reading robot credentials from secret failed: %w", err)
	}

	if username == oldRobotUser && password == oldRobotPassword {
		return nil
	}

	// Update global variables
	oldRobotUser = username
	oldRobotPassword = password
	robotReloadCounter++

	err = robotClient.SetCredentials(username, password)
	if err != nil {
		return fmt.Errorf("SetCredentials: %w", err)
	}

	klog.Infof("Hetzner Robot credentials updated to new value: %q %s...", username, password[:3])
	return nil
}

func GetInitialRobotCredentials(credentialsDir string) (username, password string, err error) {
	u, p, err := readRobotCredentials(credentialsDir)
	if err != nil {
		return "", "", fmt.Errorf("readRobotCredentials: %w", err)
	}

	// Update global variables
	oldRobotUser = u
	oldRobotPassword = p

	return u, p, nil
}

func readRobotCredentials(credentialsDir string) (username, password string, err error) {
	robotUserNameFile := filepath.Join(credentialsDir, "robot-user")
	robotPasswordFile := filepath.Join(credentialsDir, "robot-password")

	u, err := os.ReadFile(robotUserNameFile)
	if err != nil {
		return "", "", fmt.Errorf("reading robot user name from %q failed: %w", robotUserNameFile, err)
	}

	p, err := os.ReadFile(robotPasswordFile)
	if err != nil {
		return "", "", fmt.Errorf("reading robot password from %q failed: %w", robotPasswordFile, err)
	}

	return strings.TrimSpace(string(u)), strings.TrimSpace(string(p)), nil
}

func loadHcloudCredentials(credentialsDir string, hcloudClient *hcloud.Client) error {
	hcloudMutex.Lock()
	defer hcloudMutex.Unlock()

	token, err := readHcloudCredentials(credentialsDir)
	if err != nil {
		return err
	}

	if len(token) != 64 {
		return fmt.Errorf("loadHcloudCredentials: entered token (%s...) is invalid (must be exactly 64 characters long)",
			token[:5])
	}

	if token == oldHcloudToken {
		return nil
	}

	// Update global variables
	oldHcloudToken = token
	hcloudTokenReloadCounter++

	// Update credentials of hcloudClient
	hcloud.WithToken(token)(hcloudClient)

	klog.Infof("Hetzner Cloud token updated to new value: %s...", token[:5])
	return nil
}

func GetInitialHcloudCredentialsFromDirectory(credentialsDir string) (string, error) {
	token, err := readHcloudCredentials(credentialsDir)
	if err != nil {
		return "", fmt.Errorf("readHcloudCredentials: %w", err)
	}

	// Update global variable
	oldHcloudToken = token

	return token, nil
}

func readHcloudCredentials(credentialsDir string) (string, error) {
	hcloudTokenFile := filepath.Join(credentialsDir, "hcloud")
	data, err := os.ReadFile(hcloudTokenFile)
	if err != nil {
		return "", fmt.Errorf("reading hcloud token from %q failed: %w", hcloudTokenFile, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// GetDirectory returns the directory where the credentials are stored.
// The credentials are stored in the directory etc/hetzner-secret.
func GetDirectory(rootDir string) string {
	return filepath.Join(rootDir, "etc", "hetzner-secret")
}
