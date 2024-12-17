/*
Copyright 2018 Hetzner Cloud GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

var OperationCalled = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "cloud_controller_manager_operations_total",
	Help: "The total number of operation was called",
}, []string{"op"})

var registry = prometheus.NewRegistry()

func GetRegistry() *prometheus.Registry {
	return registry
}

func Serve(address string) {
	klog.Info("Starting metrics server at ", address)

	registry.MustRegister(OperationCalled)

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}

	http.Handle("/metrics", promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{}))
	// TODO: Setup proper timeouts for metrics server and remove nolint:gosec
	if err := http.ListenAndServe(address, nil); err != nil { //nolint:gosec
		klog.ErrorS(err, "create metrics service")
	}
}
