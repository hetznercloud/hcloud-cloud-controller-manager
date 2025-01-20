package metrics_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
)

func TestMetrics(t *testing.T) {
	t.Parallel()

	client := &http.Client{}
	ts := httptest.NewServer(metrics.GetHandler())
	ctx := context.Background()

	metrics.OperationCalled.WithLabelValues("test").Inc()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if m := `cloud_controller_manager_operations_total{op="test"} 1`; !strings.Contains(string(body), m) {
		t.Fatalf("no metric %s found", m)
	}

	if m := `kubernetes_build_info`; strings.Contains(string(body), m) {
		t.Fatal("kubernetes_build_info included in our metrics", m)
	}
}
