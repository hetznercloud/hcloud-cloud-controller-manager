module github.com/hetznercloud/hcloud-cloud-controller-manager

go 1.16

require (
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/golang/groupcache v0.0.0-20171101203131-84a468cf14b4 // indirect
	github.com/hetznercloud/hcloud-go v1.22.0
	github.com/stretchr/testify v1.6.1
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	gopkg.in/square/go-jose.v2 v2.3.0 // indirect
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	k8s.io/cloud-provider v0.18.8
	k8s.io/component-base v0.18.8
	k8s.io/klog/v2 v2.1.0
	k8s.io/kubernetes v1.18.3
)

replace k8s.io/api => k8s.io/api v0.18.8

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8

replace k8s.io/apimachinery => k8s.io/apimachinery v0.18.8

replace k8s.io/apiserver => k8s.io/apiserver v0.18.8

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.8

replace k8s.io/client-go => k8s.io/client-go v0.18.8

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.8

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.8

replace k8s.io/code-generator => k8s.io/code-generator v0.18.8

replace k8s.io/component-base => k8s.io/component-base v0.18.8

replace k8s.io/cri-api => k8s.io/cri-api v0.18.8

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.8

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.8

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.8

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.8

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.8

replace k8s.io/kubelet => k8s.io/kubelet v0.18.8

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.8

replace k8s.io/metrics => k8s.io/metrics v0.18.8

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.8

replace k8s.io/kubectl => k8s.io/kubectl v0.18.8
