package test

import (
	"net"
	"testing"

	"github.com/pwpon500/caplance/balancer"
)

func TestBalancerCreation(t *testing.T) {
	vip := net.ParseIP("10.0.0.50")
	connectIP := net.ParseIP("10.0.0.1")
	bal, err := balancer.New(vip, connectIP, 53)
	ok(t, err)
}
