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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

const (
	readTimeout    = 5 * time.Second
	requestTimeout = 10 * time.Second
	writeTimeout   = 20 * time.Second
)

var (
	OperationCalled = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloud_controller_manager_operations_total",
		Help: "The total number of operation was called",
	}, []string{"op"})
)

var registry = prometheus.NewRegistry()

func GetRegistry() *prometheus.Registry {
	return registry
}

func GetHandler() http.Handler {
	registry.MustRegister(OperationCalled)

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}

	return promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
}

func Serve(address string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", GetHandler())

	server := &http.Server{
		Addr:         address,
		Handler:      http.TimeoutHandler(mux, requestTimeout, "timeout"),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	klog.Info("Starting metrics server at ", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		klog.ErrorS(err, "create metrics service")
	}
}
