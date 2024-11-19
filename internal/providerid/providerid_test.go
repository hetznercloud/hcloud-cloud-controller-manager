package providerid

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromCloudServerID(t *testing.T) {
	tests := []struct {
		name     string
		serverID int64
		want     string
	}{
		{
			name:     "simple id",
			serverID: 1234,
			want:     "hcloud://1234",
		},
		{
			name:     "large id",
			serverID: 2251799813685247,
			want:     "hcloud://2251799813685247",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromCloudServerID(tt.serverID); got != tt.want {
				t.Errorf("FromCloudServerID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromRobotServerNumber(t *testing.T) {
	tests := []struct {
		name         string
		serverNumber int
		want         string
	}{
		{
			name:         "simple id",
			serverNumber: 4321,
			want:         "hrobot://4321",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromRobotServerNumber(tt.serverNumber); got != tt.want {
				t.Errorf("FromRobotServerNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToServerID(t *testing.T) {
	tests := []struct {
		name              string
		providerID        string
		wantID            int64
		wantIsCloudServer bool
		wantErr           error
	}{
		{
			name:              "[cloud] simple id",
			providerID:        "hcloud://1234",
			wantID:            1234,
			wantIsCloudServer: true,
			wantErr:           nil,
		},
		{
			name:              "[cloud] large id",
			providerID:        "hcloud://2251799813685247",
			wantID:            2251799813685247,
			wantIsCloudServer: true,
			wantErr:           nil,
		},
		{
			name:              "[cloud] invalid id",
			providerID:        "hcloud://my-cloud",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           errors.New("unable to parse server id: hcloud://my-cloud"),
		},
		{
			name:              "[cloud] missing id",
			providerID:        "hcloud://",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           errors.New("providerID is missing a serverID: hcloud://"),
		},
		{
			name:              "[robot] simple id",
			providerID:        "hrobot://4321",
			wantID:            4321,
			wantIsCloudServer: false,
			wantErr:           nil,
		},
		{
			name:              "[robot] invalid id",
			providerID:        "hrobot://my-robot",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           errors.New("unable to parse server id: hrobot://my-robot"),
		},
		{
			name:              "[robot] missing id",
			providerID:        "hrobot://",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           errors.New("providerID is missing a serverID: hrobot://"),
		},
		{
			name:              "[robot-syself] simple id",
			providerID:        "hcloud://bm-4321",
			wantID:            4321,
			wantIsCloudServer: false,
			wantErr:           nil,
		},
		{
			name:              "[robot-syself] invalid id",
			providerID:        "hcloud://bm-my-robot",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           errors.New("unable to parse server id: hcloud://bm-my-robot"),
		},
		{
			name:              "[robot-syself] missing id",
			providerID:        "hcloud://bm-",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           errors.New("providerID is missing a serverID: hcloud://bm-"),
		},
		{
			name:              "unknown format",
			providerID:        "foobar/321",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           &UnkownPrefixError{"foobar/321"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotIsCloudServer, err := ToServerID(tt.providerID)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ToServerID() expected error = %v, got nil", tt.wantErr)
					return
				}
				if errors.As(tt.wantErr, new(*UnkownPrefixError)) {
					assert.ErrorAsf(t, err, new(*UnkownPrefixError), "ToServerID() error = %v, wantErr %v", err, tt.wantErr)
				} else {
					assert.EqualErrorf(t, err, tt.wantErr.Error(), "ToServerID() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("ToServerID() unexpected error = %v, wantErr nil", err)
			}
			if gotID != tt.wantID {
				t.Errorf("ToServerID() gotID = %v, want %v", gotID, tt.wantID)
			}
			if gotIsCloudServer != tt.wantIsCloudServer {
				t.Errorf("ToServerID() gotIsCloudServer = %v, want %v", gotIsCloudServer, tt.wantIsCloudServer)
			}
		})
	}
}

func FuzzRoundTripCloud(f *testing.F) {
	f.Add(int64(123123123))

	f.Fuzz(func(t *testing.T, serverID int64) {
		providerID := FromCloudServerID(serverID)
		id, isCloudServer, err := ToServerID(providerID)
		if err != nil {
			t.Fatal(err)
		}
		if id != serverID {
			t.Fatalf("expected %d, got %d", serverID, id)
		}
		if !isCloudServer {
			t.Fatalf("expected %t, got %t", true, isCloudServer)
		}
	})
}

func FuzzRoundTripRobot(f *testing.F) {
	f.Add(123123123)

	f.Fuzz(func(t *testing.T, serverNumber int) {
		providerID := FromRobotServerNumber(serverNumber)
		id, isCloudServer, err := ToServerID(providerID)
		if err != nil {
			t.Fatal(err)
		}
		if int(id) != serverNumber {
			t.Fatalf("expected %d, got %d", serverNumber, id)
		}
		if isCloudServer {
			t.Fatalf("expected %t, got %t", false, isCloudServer)
		}
	})
}

func FuzzToServerId(f *testing.F) {
	f.Add("hcloud://123123123")
	f.Add("hrobot://123123123")
	f.Add("hcloud://bm-123123123")

	f.Fuzz(func(t *testing.T, providerID string) {
		_, _, err := ToServerID(providerID)
		if err != nil {
			if strings.HasPrefix(err.Error(), "providerID does not have one of the the expected prefixes") {
				return
			}
			if strings.HasPrefix(err.Error(), "providerID is missing a serverID") {
				return
			}
			if strings.HasPrefix(err.Error(), "unable to parse server id") {
				return
			}

			t.Fatal(err)
		}
	})
}
