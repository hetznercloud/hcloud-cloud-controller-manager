package legacydatacenter

import "testing"

func TestNameFromLocation(t *testing.T) {
	tests := []struct {
		name     string
		location string
		want     string
	}{
		{
			name:     "existing location returns dc name (ngb1)",
			location: "nbg1",
			want:     "nbg1-dc3",
		},
		{
			name:     "existing location returns dc name (hel1)",
			location: "hel1",
			want:     "hel1-dc2",
		},
		{
			name:     "existing location returns dc name (fsn1)",
			location: "fsn1",
			want:     "fsn1-dc14",
		},
		{
			name:     "existing location returns dc name (ash)",
			location: "ash",
			want:     "ash-dc1",
		},
		{
			name:     "existing location returns dc name (hil)",
			location: "hil",
			want:     "hil-dc1",
		},
		{
			name:     "existing location returns dc name (sin)",
			location: "sin",
			want:     "sin-dc1",
		},
		{
			name:     "unknown location returns location name",
			location: "mars",
			want:     "mars",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NameFromLocation(tt.location); got != tt.want {
				t.Errorf("NameFromLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}
