package providerid

import (
	"strings"
	"testing"
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
		wantErr           bool
	}{
		{
			name:              "[cloud] simple id",
			providerID:        "hcloud://1234",
			wantID:            1234,
			wantIsCloudServer: true,
			wantErr:           false,
		},
		{
			name:              "[cloud] large id",
			providerID:        "hcloud://2251799813685247",
			wantID:            2251799813685247,
			wantIsCloudServer: true,
			wantErr:           false,
		},
		{
			name:              "[cloud] invalid id",
			providerID:        "hcloud://my-cloud",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           true,
		},
		{
			name:              "[cloud] missing id",
			providerID:        "hcloud://",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           true,
		},
		{
			name:              "[robot] simple id",
			providerID:        "hrobot://4321",
			wantID:            4321,
			wantIsCloudServer: false,
			wantErr:           false,
		},
		{
			name:              "[robot] invalid id",
			providerID:        "hrobot://my-robot",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           true,
		},
		{
			name:              "[robot] missing id",
			providerID:        "hrobot://",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           true,
		},
		{
			name:              "[robot-syself] simple id",
			providerID:        "hcloud://bm-4321",
			wantID:            4321,
			wantIsCloudServer: false,
			wantErr:           false,
		},
		{
			name:              "[robot-syself] invalid id",
			providerID:        "hcloud://bm-my-robot",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           true,
		},
		{
			name:              "[robot-syself] missing id",
			providerID:        "hcloud://bm-",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           true,
		},
		{
			name:              "unknown format",
			providerID:        "foobar/321",
			wantID:            0,
			wantIsCloudServer: false,
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotIsCloudServer, err := ToServerID(tt.providerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToServerID() error = %v, wantErr %v", err, tt.wantErr)
				return
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
