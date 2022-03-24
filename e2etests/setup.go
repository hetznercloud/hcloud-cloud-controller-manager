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

type K8sDistribution string

const (
	K8sDistributionK8s K8sDistribution = "k8s"
	K8sDistributionK3s K8sDistribution = "k3s"
)

var instanceType = "cpx21"

type hcloudK8sSetup struct {
	Hcloud          *hcloud.Client
	HcloudToken     string
	K8sVersion      string
	K8sDistribution K8sDistribution
	TestIdentifier  string
	ImageName       string
	KeepOnFailure   bool
	ClusterNode     *hcloud.Server
	ExtServer       *hcloud.Server
	UseNetworks     bool
	privKey         string
	sshKey          *hcloud.SSHKey
	network         *hcloud.Network
	clusterJoinCMD  string
	WorkerNodes     []*hcloud.Server
	testLabels      map[string]string
}

type cloudInitTmpl struct {
	K8sVersion      string
	HcloudToken     string
	HcloudNetwork   string
	IsClusterServer bool
	JoinCMD         string
	UseFlannel      bool
}

// PrepareTestEnv setups a test environment for the Cloud Controller Manager
// This includes the creation of a Network, SSH Key and Server.
// The server will be created with a Cloud Init UserData
// The template can be found under e2etests/templates/cloudinit_<k8s-distribution>.ixt.tpl
func (s *hcloudK8sSetup) PrepareTestEnv(ctx context.Context, additionalSSHKeys []*hcloud.SSHKey, useNetworks bool) (string, error) {
	const op = "hcloudK8sSetup/PrepareTestEnv"

	s.testLabels = map[string]string{"K8sDistribution": string(s.K8sDistribution), "K8sVersion": strings.ReplaceAll(s.K8sVersion, "+", ""), "test": s.TestIdentifier}
	err := s.getSSHKey(ctx)
	if err != nil {
		return "", fmt.Errorf("%s getSSHKey: %s", op, err)
	}

	err = s.getNetwork(ctx)
	if err != nil {
		return "", fmt.Errorf("%s getNetwork: %s", op, err)
	}
	userData, err := s.getCloudInitConfig(true)
	if err != nil {
		fmt.Printf("[cluster-node] %s getCloudInitConfig: %s", op, err)
		return "", err
	}
	srv, err := s.createServer(ctx, "cluster-node", instanceType, additionalSSHKeys, userData)
	if err != nil {
		return "", fmt.Errorf("%s: create cluster node: %v", op, err)
	}
	s.ClusterNode = srv
	s.waitUntilSSHable(srv)
	err = s.waitForCloudInit(srv)
	if err != nil {
		return "", err
	}

	joinCmd, err := s.getJoinCmd()
	if err != nil {
		return "", err
	}
	s.clusterJoinCMD = joinCmd

	err = s.transferDockerImage(s.ClusterNode)
	if err != nil {
		return "", fmt.Errorf("%s: %s", op, err)
	}

	fmt.Printf("[%s] %s: Load Image:\n", s.ClusterNode.Name, op)
	err = RunCommandOnServer(s.privKey, s.ClusterNode, "ctr -n=k8s.io image import ci-hcloud-ccm.tar")
	if err != nil {
		return "", fmt.Errorf("%s: Load image %s", op, err)
	}
	kubeconfigPath, err := s.PrepareK8s()
	if err != nil {
		return "", fmt.Errorf("%s: %s", op, err)
	}

	var workers = 1 // Change this value if you want to have more workers for the test
	var wg sync.WaitGroup
	for worker := 1; worker <= workers; worker++ {
		wg.Add(1)
		go s.createClusterWorker(ctx, additionalSSHKeys, &wg, worker)
	}
	wg.Wait()

	srv, err = s.createServer(ctx, "ext-server", instanceType, additionalSSHKeys, "")
	if err != nil {
		return "", fmt.Errorf("%s: create ext server: %v", op, err)
	}
	s.ExtServer = srv
	s.waitUntilSSHable(srv)

	return kubeconfigPath, nil
}

func (s *hcloudK8sSetup) createClusterWorker(ctx context.Context, additionalSSHKeys []*hcloud.SSHKey, wg *sync.WaitGroup, worker int) {
	const op = "hcloudK8sSetup/createClusterWorker"
	defer wg.Done()

	workerName := fmt.Sprintf("cluster-worker-%d", worker)
	fmt.Printf("[%s] %s Create worker node:\n", workerName, op)

	userData, err := s.getCloudInitConfig(false)
	if err != nil {
		fmt.Printf("[%s] %s getCloudInitConfig: %s", workerName, op, err)
		return
	}
	srv, err := s.createServer(ctx, workerName, instanceType, additionalSSHKeys, userData)
	if err != nil {
		fmt.Printf("[%s] %s createServer: %s", workerName, op, err)
		return
	}
	s.WorkerNodes = append(s.WorkerNodes, srv)

	s.waitUntilSSHable(srv)

	err = s.waitForCloudInit(srv)
	if err != nil {
		fmt.Printf("[%s] %s: wait for cloud init on worker: %v", srv.Name, op, err)
		return
	}

	err = s.transferDockerImage(srv)
	if err != nil {
		fmt.Printf("[%s] %s: transfer image on worker: %v", srv.Name, op, err)
		return
	}

	fmt.Printf("[%s] %s Load Image\n", srv.Name, op)
	err = RunCommandOnServer(s.privKey, srv, "ctr -n=k8s.io image import ci-hcloud-ccm.tar")
	if err != nil {
		fmt.Printf("[%s] %s: load image on worker: %v", srv.Name, op, err)
		return
	}
}

// waitForCloudInit waits on cloud init on the server.
// when cloud init is ready we can assume that the server
// and the plain k8s installation is ready
func (s *hcloudK8sSetup) getJoinCmd() (string, error) {
	const op = "hcloudK8sSetup/getJoinCmd"
	fmt.Printf("[%s] %s: Download join cmd\n", s.ClusterNode.Name, op)
	if s.K8sDistribution == K8sDistributionK8s {
		err := scp("ssh_key", fmt.Sprintf("root@%s:/root/join.txt", s.ClusterNode.PublicNet.IPv4.IP.String()), "join.txt")
		if err != nil {
			return "", fmt.Errorf("[%s] %s download join cmd: %s", s.ClusterNode.Name, op, err)
		}
		cmd, err := ioutil.ReadFile("join.txt")
		if err != nil {
			return "", fmt.Errorf("[%s] %s reading join cmd file: %s", s.ClusterNode.Name, op, err)
		}

		return string(cmd), nil
	}
	err := scp("ssh_key", fmt.Sprintf("root@%s:/var/lib/rancher/k3s/server/node-token", s.ClusterNode.PublicNet.IPv4.IP.String()), "join.txt")
	if err != nil {
		return "", fmt.Errorf("[%s] %s download join cmd: %s", s.ClusterNode.Name, op, err)
	}
	token, err := ioutil.ReadFile("join.txt")
	return fmt.Sprintf("K3S_URL=https://%s:6443 K3S_TOKEN=%s", s.ClusterNode.PublicNet.IPv4.IP.String(), token), nil
}

func (s *hcloudK8sSetup) waitUntilSSHable(server *hcloud.Server) {
	const op = "hcloudK8sSetup/PrepareTestEnv"
	fmt.Printf("[%s] %s: Waiting for server to be sshable:\n", server.Name, op)
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:22", server.PublicNet.IPv4.IP.String()))
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		_ = conn.Close()
		fmt.Printf("[%s] %s: SSH Connection successful\n", server.Name, op)
		break
	}
}

func (s *hcloudK8sSetup) createServer(ctx context.Context, name, typ string, additionalSSHKeys []*hcloud.SSHKey, userData string) (*hcloud.Server, error) {
	const op = "e2etest/createServer"

	sshKeys := []*hcloud.SSHKey{s.sshKey}
	for _, additionalSSHKey := range additionalSSHKeys {
		sshKeys = append(sshKeys, additionalSSHKey)
	}

	res, _, err := s.Hcloud.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       fmt.Sprintf("srv-%s-%s", name, s.TestIdentifier),
		ServerType: &hcloud.ServerType{Name: typ},
		Image:      &hcloud.Image{Name: "ubuntu-20.04"},
		SSHKeys:    sshKeys,
		UserData:   userData,
		Labels:     s.testLabels,
		Networks:   []*hcloud.Network{s.network},
	})
	if err != nil {
		return nil, fmt.Errorf("%s Hcloud.Server.Create: %s", op, err)
	}

	_, errCh := s.Hcloud.Action.WatchProgress(ctx, res.Action)
	if err := <-errCh; err != nil {
		return nil, fmt.Errorf("%s WatchProgress Action %s: %s", op, res.Action.Command, err)
	}

	for _, nextAction := range res.NextActions {
		_, errCh := s.Hcloud.Action.WatchProgress(ctx, nextAction)
		if err := <-errCh; err != nil {
			return nil, fmt.Errorf("%s WatchProgress NextAction %s: %s", op, nextAction.Command, err)
		}
	}
	srv, _, err := s.Hcloud.Server.GetByID(ctx, res.Server.ID)
	if err != nil {
		return nil, fmt.Errorf("%s Hcloud.Server.GetByID: %s", op, err)
	}
	return srv, nil
}

// PrepareK8s patches an existing kubernetes cluster with a CNI and the correct
// Cloud Controller Manager version from this test run
func (s *hcloudK8sSetup) PrepareK8s() (string, error) {
	const op = "hcloudK8sSetup/PrepareK8s"

	if s.UseNetworks {
		err := s.deployCilium()
		if err != nil {
			return "", fmt.Errorf("%s: %s", op, err)
		}
	}
	if s.K8sDistribution != K8sDistributionK3s && !s.UseNetworks {
		err := s.deployFlannel()
		if err != nil {
			return "", fmt.Errorf("%s: %s", op, err)
		}
	}

	err := s.prepareCCMDeploymentFile(s.UseNetworks)
	if err != nil {
		return "", fmt.Errorf("%s: %s", op, err)
	}

	fmt.Printf("[%s] %s: Apply ccm deployment\n", s.ClusterNode.Name, op)
	err = RunCommandOnServer(s.privKey, s.ClusterNode, "KUBECONFIG=/root/.kube/config kubectl apply -f ccm.yml")
	if err != nil {
		return "", fmt.Errorf("%s Deploy ccm: %s", op, err)
	}

	fmt.Printf("[%s] %s: Download kubeconfig\n", s.ClusterNode.Name, op)

	err = scp("ssh_key", fmt.Sprintf("root@%s:/root/.kube/config", s.ClusterNode.PublicNet.IPv4.IP.String()), "kubeconfig")
	if err != nil {
		return "", fmt.Errorf("%s download kubeconfig: %s", op, err)
	}

	fmt.Printf("[%s] %s: Ensure correct server is set\n", s.ClusterNode.Name, op)
	kubeconfigBefore, err := ioutil.ReadFile("kubeconfig")
	if err != nil {
		return "", fmt.Errorf("%s reading kubeconfig: %s", op, err)
	}
	kubeconfigAfterwards := strings.Replace(string(kubeconfigBefore), "127.0.0.1", s.ClusterNode.PublicNet.IPv4.IP.String(), -1)
	err = ioutil.WriteFile("kubeconfig", []byte(kubeconfigAfterwards), 0)
	if err != nil {
		return "", fmt.Errorf("%s writing kubeconfig: %s", op, err)
	}
	return "kubeconfig", nil
}

func scp(identityFile, src, dest string) error {
	const op = "e2etests/scp"

	err := runCmd(
		"/usr/bin/scp",
		[]string{
			"-F", "/dev/null", // ignore $HOME/.ssh/config
			"-i", identityFile,
			"-o", "IdentitiesOnly=yes", // only use the identities passed on the command line
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "StrictHostKeyChecking=no",
			src,
			dest,
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	return nil
}

func runCmd(name string, argv []string, env []string) error {
	cmd := exec.Command(name, argv...)
	if os.Getenv("TEST_DEBUG_MODE") != "" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run cmd: %s %s: %v", name, strings.Join(argv, " "), err)
	}
	return nil
}

// prepareCCMDeploymentFile patches the Cloud Controller Deployment file
// It replaces the used image and the pull policy to always use the local image
// from this test run
func (s *hcloudK8sSetup) prepareCCMDeploymentFile(networks bool) error {
	const op = "hcloudK8sSetup/prepareCCMDeploymentFile"
	fmt.Printf("%s: Read master deployment file\n", op)
	var deploymentFilePath = "../deploy/dev-ccm.yaml"
	if networks {
		deploymentFilePath = "../deploy/dev-ccm-networks.yaml"
	}
	deploymentFile, err := ioutil.ReadFile(deploymentFilePath)
	if err != nil {
		return fmt.Errorf("%s: read ccm deployment file %s: %v", op, deploymentFilePath, err)
	}

	fmt.Printf("%s: Prepare deployment file and transfer it\n", op)
	deploymentFile = []byte(strings.ReplaceAll(string(deploymentFile), "quay.io/syself/hetzner-cloud-controller-manager:latest", s.ImageName))
	deploymentFile = []byte(strings.ReplaceAll(string(deploymentFile), " imagePullPolicy: Always", " imagePullPolicy: IfNotPresent"))

	err = RunCommandOnServer(s.privKey, s.ClusterNode, fmt.Sprintf("echo '%s' >> ccm.yml", deploymentFile))
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
	err := RunCommandOnServer(s.privKey, s.ClusterNode, "KUBECONFIG=/root/.kube/config kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml")
	if err != nil {
		return fmt.Errorf("%s: apply flannel deployment: %s", op, err)
	}
	fmt.Printf("%s: patch flannel deployment\n", op)
	err = RunCommandOnServer(s.privKey, s.ClusterNode, "KUBECONFIG=/root/.kube/config kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{\"op\":\"add\",\"path\":\"/spec/template/spec/tolerations/-\",\"value\":{\"key\":\"node.cloudprovider.kubernetes.io/uninitialized\",\"value\":\"true\",\"effect\":\"NoSchedule\"}}]'")
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
	err = RunCommandOnServer(s.privKey, s.ClusterNode, fmt.Sprintf("cat <<EOF > cilium.yml\n%s\nEOF", deploymentFile))
	if err != nil {
		return fmt.Errorf("%s: Transfer cilium deployment: %s", op, err)
	}

	fmt.Printf("%s: apply cilium deployment\n", op)
	err = RunCommandOnServer(s.privKey, s.ClusterNode, "KUBECONFIG=/root/.kube/config kubectl apply -f cilium.yml")
	if err != nil {
		return fmt.Errorf("%s: apply cilium deployment: %s", op, err)
	}

	return nil
}

// transferDockerImage transfers the local build docker image tar via SCP
func (s *hcloudK8sSetup) transferDockerImage(server *hcloud.Server) error {
	const op = "hcloudK8sSetup/transferDockerImage"
	fmt.Printf("[%s] %s: Transfer docker image\n", server.Name, op)
	err := WithSSHSession(s.privKey, server.PublicNet.IPv4.IP.String(), func(session *ssh.Session) error {
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
func (s *hcloudK8sSetup) waitForCloudInit(server *hcloud.Server) error {
	const op = "hcloudK8sSetup/PrepareTestEnv"
	fmt.Printf("[%s] %s: Wait for cloud-init\n", server.Name, op)
	err := RunCommandOnServer(s.privKey, server, fmt.Sprintf("cloud-init status --wait > /dev/null"))
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

	_, err := s.Hcloud.Server.Delete(ctx, s.ClusterNode)
	if err != nil {
		return fmt.Errorf("%s Hcloud.Server.Delete: %s", op, err)
	}
	s.ClusterNode = nil

	for _, wn := range s.WorkerNodes {
		_, err := s.Hcloud.Server.Delete(ctx, wn)
		if err != nil {
			return fmt.Errorf("[%s] %s Hcloud.Server.Delete: %s", wn.Name, op, err)
		}
	}

	_, err = s.Hcloud.Server.Delete(ctx, s.ExtServer)
	if err != nil {
		return fmt.Errorf("%s Hcloud.Server.Delete: %s", op, err)
	}
	s.ExtServer = nil

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
func (s *hcloudK8sSetup) getCloudInitConfig(isClusterServer bool) (string, error) {
	const op = "hcloudK8sSetup/getCloudInitConfig"

	data := cloudInitTmpl{
		K8sVersion:      s.K8sVersion,
		HcloudToken:     s.HcloudToken,
		HcloudNetwork:   s.network.Name,
		IsClusterServer: isClusterServer,
		JoinCMD:         s.clusterJoinCMD,
		UseFlannel:      s.K8sDistribution == K8sDistributionK3s && !s.UseNetworks,
	}
	str, err := ioutil.ReadFile(fmt.Sprintf("templates/cloudinit_%s.txt.tpl", s.K8sDistribution))
	if err != nil {
		return "", fmt.Errorf("%s: read template file %s: %v", "templates/cloudinit.txt.tpl", op, err)
	}
	tmpl, err := template.New("cloud_init").Parse(string(str))
	if err != nil {
		return "", fmt.Errorf("%s: parsing template file %s: %v", "templates/cloudinit.txt.tpl", op, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
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
		Labels:    s.testLabels,
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
		Labels:  s.testLabels,
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
			session.Stderr = os.Stderr
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
