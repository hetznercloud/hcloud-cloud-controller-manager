package hcops

import (
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSameIP(t *testing.T) {
	tests := []struct {
		name     string
		addr     netip.Addr
		ip       net.IP
		expected bool
	}{
		{
			name:     "the zero value equals nothing",
			addr:     netip.Addr{},
			ip:       net.ParseIP("10.0.0.1"),
			expected: false,
		},
		{
			name:     "the zero value does not equal a missing IP either",
			addr:     netip.Addr{},
			ip:       nil,
			expected: false,
		},
		{
			name:     "same address",
			addr:     netip.MustParseAddr("10.0.0.1"),
			ip:       net.ParseIP("10.0.0.1"),
			expected: true,
		},
		{
			name:     "same address written as IPv4-in-IPv6",
			addr:     netip.MustParseAddr("::ffff:10.0.0.1"),
			ip:       net.ParseIP("10.0.0.1"),
			expected: true,
		},
		{
			name:     "different address",
			addr:     netip.MustParseAddr("10.0.0.1"),
			ip:       net.ParseIP("10.0.0.2"),
			expected: false,
		},
		{
			name:     "a missing IP equals no address",
			addr:     netip.MustParseAddr("10.0.0.1"),
			ip:       nil,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, sameIP(test.addr, test.ip))
		})
	}
}
