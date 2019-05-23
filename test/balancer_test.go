package test

import (
	"net"
	"testing"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/pwpon500/caplance/balancer"
)

func TestBalancerCreation(t *testing.T) {
	vip := net.ParseIP("10.0.0.50")
	connectIP := net.ParseIP("10.0.0.1")
	_, err := balancer.New(vip, connectIP, 53)
	ok(t, err)
}

func TestVIPAttachDetach(t *testing.T) {
	vip := net.ParseIP("10.0.0.50")
	connectIP := net.ParseIP("10.0.0.1")
	bal, err := balancer.NewTest(vip, connectIP, 53)
	go bal.Start()
	time.Sleep(10 * time.Millisecond) // sleep long enough to ensure Start() gets mutex lock
	bal.WaitForUnlock()

	devices, err := pcap.FindAllDevs()
	ok(t, err)
	foundDevice := ""
	for _, device := range devices {
		if device.Flags != 6 {
			continue
		}
		for _, address := range device.Addresses {
			if address.IP.Equal(vip) {
				if foundDevice == "" {
					foundDevice = device.Name
				}
			}
		}
	}
	equals(t, "h1-eth0", foundDevice)

	bal.Stop()
	time.Sleep(10 * time.Millisecond) // sleep long enough to ensure Start() gets mutex lock
	bal.WaitForUnlock()

	devices, err = pcap.FindAllDevs()
	ok(t, err)
	foundDevice = ""
	for _, device := range devices {
		if device.Flags != 6 {
			continue
		}
		for _, address := range device.Addresses {
			if address.IP.Equal(vip) {
				if foundDevice == "" {
					foundDevice = device.Name
				}
			}
		}
	}
	equals(t, "", foundDevice)
}
