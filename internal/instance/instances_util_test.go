package instance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hrobotmodels "github.com/syself/hrobot-go/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/events"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
)

func TestGetRobotServerByID(t *testing.T) {
	tests := []struct {
		name          string
		nodeName      string
		expectedEvent string
	}{
		{
			name:     "no diff robot and node name",
			nodeName: "foobar",
		},
		{
			name:          "diff robot and node name",
			nodeName:      "barfoo",
			expectedEvent: `Warning PossibleNodeDeletion Might be deleted by node-lifecycle-manager due to name mismatch; Node name "barfoo" differs from Robot name "foobar"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := record.NewFakeRecorder(1)

			robotClientMock := &mocks.RobotClient{}
			robotClientMock.Test(t)
			robotClientMock.On("ServerGet").Return(&hrobotmodels.Server{ServerNumber: 1, Name: "foobar"}, nil)

			inst := &instances{
				recorder:    events.NewRecorder(recorder),
				robotClient: robotClientMock,
			}

			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.nodeName,
				},
			}

			server, err := getRobotServerByID(inst, 1, node)
			require.NoError(t, err)

			if tt.expectedEvent != "" {
				require.Nil(t, server)
			} else {
				assert.Equal(t, "foobar", server.Name)
			}

			testsupport.AssertEvent(t, recorder, tt.expectedEvent)
		})
	}
}
