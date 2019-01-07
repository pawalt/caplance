package util

import (
	"errors"
	"github.com/google/gopacket/pcap"
	"github.com/vishvananda/netlink"
	"net"
)

func CreateTun(localIP, remoteIP, tunIP, remoteTunIP net.IP, name string) *netlink.Gretun {
	la := netlink.NewLinkAttrs()
	la.Name = name
	tun := &netlink.Gretun{
		LinkAttrs: la,
		Local:     localIP,
		Remote:    remoteIP,
	}
	err := netlink.LinkAdd(tun)
	handleErr(err)
	ipNet := genTunIPNet(tunIP)
	addr := &netlink.Addr{IPNet: ipNet, Peer: genTunIPNet(remoteTunIP)}
	netlink.AddrAdd(tun, addr)
	return tun
}

func genTunIPNet(ip net.IP) *net.IPNet {
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(30, 32)}
}

func AttachVIP(vip net.IP) (string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return "", err
	}
	foundDevice := ""
	for _, device := range devices {
		for _, address := range device.Addresses {
			ipNet := &net.IPNet{IP: address.IP, Mask: address.Netmask}
			if ipNet.Contains(vip) {
				if foundDevice == "" {
					foundDevice = device.Name
				} else {
					return "", errors.New("Multiple devices on the same subnet. VIP cannot be assigned.")
				}
			}
		}
	}
	if foundDevice == "" {
		return "", errors.New("No device on same subnet as VIP. VIP cannot be assigned.")
	}
	dev, err := netlink.LinkByName(foundDevice)
	if err != nil {
		return "", err
	}
	vipNet := &net.IPNet{IP: vip, Mask: net.CIDRMask(32, 32)}
	netlink.AddrAdd(dev, &netlink.Addr{IPNet: vipNet})
	return foundDevice, nil
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
