package mocks

import (
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/mock"
	"github.com/syself/hrobot-go/models"
)

func getResponsePtr(args mock.Arguments, i int) *hcloud.Response {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.(*hcloud.Response)
}

func getActionPtr(args mock.Arguments, i int) *hcloud.Action {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.(*hcloud.Action)
}

func GetLoadBalancerPtr(args mock.Arguments, i int) *hcloud.LoadBalancer {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.(*hcloud.LoadBalancer)
}

func getLoadBalancerPtrS(args mock.Arguments, i int) []*hcloud.LoadBalancer {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.([]*hcloud.LoadBalancer)
}

func getRobotServers(args mock.Arguments, i int) []models.Server {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.([]models.Server)
}

func getNetworkPtr(args mock.Arguments, i int) *hcloud.Network {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.(*hcloud.Network)
}

func getIntChan(args mock.Arguments, i int) chan int {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.(chan int)
}

func getErrChan(args mock.Arguments, i int) chan error {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.(chan error)
}

func getCertificatePtr(args mock.Arguments, i int) *hcloud.Certificate {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.(*hcloud.Certificate)
}

func getCertificatePtrS(args mock.Arguments, i int) []*hcloud.Certificate {
	v := args.Get(i)
	if v == nil {
		return nil
	}
	return v.([]*hcloud.Certificate)
}

func getCertificateCreateResult(args mock.Arguments, i int) hcloud.CertificateCreateResult {
	v := args.Get(i)
	if v == nil {
		return hcloud.CertificateCreateResult{}
	}
	return v.(hcloud.CertificateCreateResult)
}
