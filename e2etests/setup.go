package e2etests

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"golang.org/x/crypto/ssh"
)

type hcloudK8sSetup struct {
	Hcloud         *hcloud.Client
	HcloudToken    string
	K8sVersion     string
	TestIdentifier string
	KeepOnFailure  bool
	privKey        string
	server         *hcloud.Server
	sshKey         *hcloud.SSHKey
	network        *hcloud.Network
}

type cloudInitTmpl struct {
	K8sVersion    string
	HcloudToken   string
	HcloudNetwork string
}

// PrepareTestEnv setups a test environment for the Cloud Controller Manager
// This includes the creation of a Network, SSH Key and Server.
// The server will be created with a Cloud Init UserData
// The template can be found under e2etests/templates/cloudinit.ixt.tpl
func (s *hcloudK8sSetup) PrepareTestEnv(ctx context.Context, additionalSSHKeys []*hcloud.SSHKey) error {
	const op = "hcloudK8sSetup/PrepareTestEnv"

	err := s.getSSHKey(ctx)
	if err != nil {
		return fmt.Errorf("%s getSSHKey: %s", op, err)
	}

	err = s.getNetwork(ctx)
	if err != nil {
		return fmt.Errorf("%s getNetwork: %s", op, err)
	}
	userData, err := s.getCloudInitConfig()
	if err != nil {
		return fmt.Errorf("%s getCloudInitConfig: %s", op, err)
	}
	sshKeys := []*hcloud.SSHKey{s.sshKey}
	for _, additionalSSHKey := range additionalSSHKeys {
		sshKeys = append(sshKeys, additionalSSHKey)
	}

	res, _, err := s.Hcloud.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       fmt.Sprintf("srv-%s", s.TestIdentifier),
		ServerType: &hcloud.ServerType{Name: "cpx21"},
		Image:      &hcloud.Image{Name: "ubuntu-20.04"},
		SSHKeys:    sshKeys,
		UserData:   userData,
		Labels:     map[string]string{"K8sVersion": s.K8sVersion, "test": s.TestIdentifier},
		Networks:   []*hcloud.Network{s.network},
	})
	if err != nil {
		return fmt.Errorf("%s Hcloud.Server.Create: %s", op, err)
	}

	_, errCh := s.Hcloud.Action.WatchProgress(ctx, res.Action)
	if err := <-errCh; err != nil {
		return fmt.Errorf("%s WatchProgress Action %s: %s", op, res.Action.Command, err)
	}

	for _, nextAction := range res.NextActions {
		_, errCh := s.Hcloud.Action.WatchProgress(ctx, nextAction)
		if err := <-errCh; err != nil {
			return fmt.Errorf("%s WatchProgress NextAction %s: %s", op, nextAction.Command, err)
		}
	}
	s.server, _, err = s.Hcloud.Server.GetByID(ctx, res.Server.ID)
	if err != nil {
		return fmt.Errorf("%s Hcloud.Server.GetByID: %s", op, err)
	}

	fmt.Printf("%s Waiting for server to be sshable:", op)
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:22", s.server.PublicNet.IPv4.IP.String()))
		if err != nil {
			fmt.Print(".")
			time.Sleep(1 * time.Second)
			continue
		}
		_ = conn.Close()
		fmt.Print("Connection successful\n")
		break
	}
	err = s.waitForCloudInit()
	if err != nil {
		return err
	}

	err = s.transferCCMDockerImage()
	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}

	fmt.Printf("%s Load Image:\n", op)
	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("docker load --input ci-hcloud-ccm.tar"))
	if err != nil {
		return fmt.Errorf("%s:  Load image %s", op, err)
	}

	return nil
}

// PrepareK8s patches an existing kubernetes cluster with a CNI and the correct
// Cloud Controller Manager version from this test run
func (s *hcloudK8sSetup) PrepareK8s(withNetworks bool) (string, error) {
	const op = "hcloudK8sSetup/PrepareK8s"

	if withNetworks {
		err := s.deployCilium()
		if err != nil {
			return "", fmt.Errorf("%s: %s", op, err)
		}
	} else {
		err := s.deployFlannel()
		if err != nil {
			return "", fmt.Errorf("%s: %s", op, err)
		}
	}

	err := s.prepareCCMDeploymentFile(withNetworks)
	if err != nil {
		return "", fmt.Errorf("%s: %s", op, err)
	}

	fmt.Printf("%s: Apply ccm deployment\n", op)
	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("KUBECONFIG=/root/.kube/config kubectl apply -f ccm.yml"))
	if err != nil {
		return "", fmt.Errorf("%s Deploy ccm: %s", op, err)
	}
	fmt.Printf("%s: Download kubeconfig\n", op)

	cmd := exec.Command("/usr/bin/scp", "-i", "ssh_key", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("root@%s:/root/.kube/config", s.server.PublicNet.IPv4.IP.String()), "kubeconfig")
	if ok := os.Getenv("TEST_DEBUG_MODE"); ok != "" {
		cmd.Stdout = os.Stdout
	}
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s download kubeconfig: %s", op, err)
	}
	return "kubeconfig", nil
}

// prepareCCMDeploymentFile patches the Cloud Controller Deployment file
// It replaces the used image and the pull policy to always use the local image
// from this test run
func (s *hcloudK8sSetup) prepareCCMDeploymentFile(networks bool) error {
	const op = "hcloudK8sSetup/prepareCCMDeploymentFile"
	fmt.Printf("%s: Read master deployment filee\n", op)
	var deploymentFilePath = "../deploy/dev-ccm.yaml"
	if networks {
		deploymentFilePath = "../deploy/dev-ccm-networks.yaml"
	}
	deploymentFile, err := ioutil.ReadFile(deploymentFilePath)
	if err != nil {
		return fmt.Errorf("%s: read ccm deployment file %s: %v", op, deploymentFilePath, err)
	}

	fmt.Printf("%s: Prepare deployment file and transfer it\n", op)
	deploymentFile = []byte(strings.ReplaceAll(string(deploymentFile), "hetznercloud/hcloud-cloud-controller-manager:latest", fmt.Sprintf("hcloud-ccm:ci_%s", s.TestIdentifier)))
	deploymentFile = []byte(strings.ReplaceAll(string(deploymentFile), " imagePullPolicy: Always", " imagePullPolicy: IfNotPresent"))

	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("echo '%s' >> ccm.yml", deploymentFile))
	if err != nil {
		return fmt.Errorf("%s: Prepare deployment file and transfer it: %s", op, err)
	}
	return nil
}

// deployFlannel deploys Flannel as CNI. Flannel is used for all tests where
// we don't use Hetzner Cloud Networks
func (s *hcloudK8sSetup) deployFlannel() error {
	const op = "hcloudK8sSetup/deployFlannel"
	fmt.Printf("%s: apply flannel deployment\n", op)
	err := RunCommandOnServer(s.privKey, s.server, "KUBECONFIG=/root/.kube/config kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml")
	if err != nil {
		return fmt.Errorf("%s: apply flannel deployment: %s", op, err)
	}
	fmt.Printf("%s: patch flannel deployment\n", op)
	err = RunCommandOnServer(s.privKey, s.server, "KUBECONFIG=/root/.kube/config kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{\"op\":\"add\",\"path\":\"/spec/template/spec/tolerations/-\",\"value\":{\"key\":\"node.cloudprovider.kubernetes.io/uninitialized\",\"value\":\"true\",\"effect\":\"NoSchedule\"}}]'")
	if err != nil {
		return fmt.Errorf("%s: patch flannel deployment: %s", op, err)
	}
	return nil
}

// deployCilium deploys Cilium as CNI. Cilium is used for all tests where
// we use Hetzner Cloud Networks as Cilium is one of the only CNIs
// that support Cloud Controllers as source for advertising routes.
func (s *hcloudK8sSetup) deployCilium() error {
	const op = "hcloudK8sSetup/deployCilium"

	deploymentFile, err := ioutil.ReadFile("templates/cilium.yml")
	if err != nil {
		return fmt.Errorf("%s: read cilium deployment file %s: %v", op, "templates/cilium.yml", err)
	}
	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("echo '%s' >> cilium.yml", deploymentFile))
	if err != nil {
		return fmt.Errorf("%s: Transfer cilium deployment: %s", op, err)
	}

	fmt.Printf("%s: apply cilium deployment\n", op)
	err = RunCommandOnServer(s.privKey, s.server, "KUBECONFIG=/root/.kube/config kubectl apply -f cilium.yml")
	if err != nil {
		return fmt.Errorf("%s: apply cilium deployment: %s", op, err)
	}

	return nil
}

// transferCCMDockerImage transfers the local build docker image tar via SCP
func (s *hcloudK8sSetup) transferCCMDockerImage() error {
	const op = "hcloudK8sSetup/transferCCMDockerImage"
	fmt.Printf("%s: Transfer docker image\n", op)
	err := WithSSHSession(s.privKey, s.server.PublicNet.IPv4.IP.String(), func(session *ssh.Session) error {
		file, err := os.Open("ci-hcloud-ccm.tar")
		if err != nil {
			return fmt.Errorf("%s read ci-hcloud-ccm.tar: %s", op, err)
		}
		defer file.Close()
		stat, err := file.Stat()
		if err != nil {
			return fmt.Errorf("%s file.Stat: %s", op, err)
		}
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			hostIn, _ := session.StdinPipe()
			defer hostIn.Close()
			fmt.Fprintf(hostIn, "C0664 %d %s\n", stat.Size(), "ci-hcloud-ccm.tar")
			io.Copy(hostIn, file)
			fmt.Fprint(hostIn, "\x00")
			wg.Done()
		}()

		err = session.Run("/usr/bin/scp -t /root")
		if err != nil {
			return fmt.Errorf("%s copy via scp: %s", op, err)
		}
		wg.Wait()
		return err
	})
	return err
}

// waitForCloudInit waits on cloud init on the server.
// when cloud init is ready we can assume that the server
// and the plain k8s installation is ready
func (s *hcloudK8sSetup) waitForCloudInit() error {
	const op = "hcloudK8sSetup/PrepareTestEnv"
	fmt.Printf("%s: Wait for cloud-init\n", op)
	err := RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("cloud-init status --wait > /dev/null"))
	if err != nil {
		return fmt.Errorf("%s: Wait for cloud-init: %s", op, err)
	}
	return nil
}

// TearDown deletes all created resources within the Hetzner Cloud
// there is no need to "shutdown" the k8s cluster before
// so we just delete all created resources
func (s *hcloudK8sSetup) TearDown(testFailed bool) error {
	const op = "hcloudK8sSetup/TearDown"

	if s.KeepOnFailure && testFailed {
		fmt.Println("Skipping tear-down for further analysis.")
		fmt.Println("Please clean-up afterwards ;-)")
		return nil
	}

	ctx := context.Background()

	_, err := s.Hcloud.Server.Delete(ctx, s.server)
	if err != nil {
		return fmt.Errorf("%s Hcloud.Server.Delete: %s", op, err)
	}
	s.server = nil
	_, err = s.Hcloud.SSHKey.Delete(ctx, s.sshKey)
	if err != nil {
		return fmt.Errorf("%s Hcloud.SSHKey.Delete: %s", err, err)
	}
	s.sshKey = nil
	_, err = s.Hcloud.Network.Delete(ctx, s.network)
	if err != nil {
		return fmt.Errorf("%s Hcloud.Network.Delete: %s", err, err)
	}
	s.network = nil
	return nil
}

// getCloudInitConfig returns the generated cloud init configuration
func (s *hcloudK8sSetup) getCloudInitConfig() (string, error) {
	const op = "hcloudK8sSetup/getCloudInitConfig"
	str, err := ioutil.ReadFile("templates/cloudinit.txt.tpl")
	if err != nil {
		return "", fmt.Errorf("%s: read template file %s: %v", "templates/cloudinit.txt.tpl", op, err)
	}
	tmpl, err := template.New("cloud_init").Parse(string(str))
	if err != nil {
		return "", fmt.Errorf("%s: parsing template file %s: %v", "templates/cloudinit.txt.tpl", op, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cloudInitTmpl{K8sVersion: s.K8sVersion, HcloudToken: s.HcloudToken, HcloudNetwork: s.network.Name}); err != nil {
		return "", fmt.Errorf("%s: execute template: %v", op, err)
	}
	return buf.String(), nil
}

//getSSHKey create and get the Hetzner Cloud SSH Key for the test
func (s *hcloudK8sSetup) getSSHKey(ctx context.Context) error {
	const op = "hcloudK8sSetup/getSSHKey"
	pubKey, privKey, err := makeSSHKeyPair()
	if err != nil {
		return err
	}
	sshKey, _, err := s.Hcloud.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      fmt.Sprintf("s-%s", s.TestIdentifier),
		PublicKey: pubKey,
		Labels:    map[string]string{"K8sVersion": s.K8sVersion, "test": s.TestIdentifier},
	})
	if err != nil {
		return fmt.Errorf("%s: creating ssh key: %v", op, err)
	}
	s.privKey = privKey
	s.sshKey = sshKey
	err = ioutil.WriteFile("ssh_key", []byte(s.privKey), 0600)
	if err != nil {
		return fmt.Errorf("%s: writing ssh key private key: %v", op, err)
	}
	return nil
}

// getNetwork create a Hetzner Cloud Network for this test
func (s *hcloudK8sSetup) getNetwork(ctx context.Context) error {
	const op = "hcloudK8sSetup/getNetwork"
	_, ipRange, _ := net.ParseCIDR("10.0.0.0/8")
	_, subnetRange, _ := net.ParseCIDR("10.0.0.0/16")
	network, _, err := s.Hcloud.Network.Create(ctx, hcloud.NetworkCreateOpts{
		Name:    fmt.Sprintf("nw-%s", s.TestIdentifier),
		IPRange: ipRange,
		Labels:  map[string]string{"K8sVersion": s.K8sVersion, "test": s.TestIdentifier},
	})
	if err != nil {
		return fmt.Errorf("%s: creating network: %v", op, err)
	}
	_, _, err = s.Hcloud.Network.AddSubnet(ctx, network, hcloud.NetworkAddSubnetOpts{
		Subnet: hcloud.NetworkSubnet{
			Type:        hcloud.NetworkSubnetTypeCloud,
			IPRange:     subnetRange,
			NetworkZone: hcloud.NetworkZoneEUCentral,
		},
	})
	if err != nil {
		return fmt.Errorf("%s: creating subnet: %v", op, err)
	}
	s.network = network
	return nil
}

// makeSSHKeyPair generate a SSH key pair
func makeSSHKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", err
	}

	// generate and write private key as PEM
	var privKeyBuf strings.Builder

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", "", err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))

	return pubKeyBuf.String(), privKeyBuf.String(), nil
}

func RunCommandOnServer(privKey string, server *hcloud.Server, command string) error {
	return WithSSHSession(privKey, server.PublicNet.IPv4.IP.String(), func(session *ssh.Session) error {
		if ok := os.Getenv("TEST_DEBUG_MODE"); ok != "" {
			session.Stdout = os.Stdout
		}
		return session.Run(command)
	})
}

func WithSSHSession(privKey string, host string, fn func(*ssh.Session) error) error {
	signer, err := ssh.ParsePrivateKey([]byte(privKey))
	if err != nil {
		return err
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(host, "22"), &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         1 * time.Second,
	})
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return fn(session)
}
