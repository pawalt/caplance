package balancer

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/pwpon500/caplance/balancer/backend"
	"github.com/vishvananda/netlink"
	"net"
	"time"
)

type Balancer struct {
	backends *backend.BackendHandler
	vip      net.IP
}

func New(startVIP net.IP, capacity int64) (*Balancer, error) {
	back, err := backend.New(capacity)
	if err != nil {
		return nil, err
	}
	return &Balancer{backends: back, vip: startVIP}, nil
}

func (b *Balancer) Start() {
	dev, err := attachVIP(b.vip)
	handleErr(err)
	b.listen(dev)
}

func createTun(localIP, remoteIP, tunIP, remoteTunIP net.IP, name string) *netlink.Gretun {
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

func (b *Balancer) listen(deviceName string) {
	handle, err := pcap.OpenLive(deviceName, 1024, false, 30*time.Second)
	handleErr(err)
	defer handle.Close()

	filter := "dst host " + b.vip.String()
	err = handle.SetBPFFilter(filter)
	handleErr(err)

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		go b.handlePacket(packet)
	}
}

func (b *Balancer) handlePacket(packet gopacket.Packet) {
	_, dst := packet.NetworkLayer().NetworkFlow().Endpoints()
	backend, err := b.backends.Get(dst.String())
	if err != nil {
		fmt.Println("Packet received with no backends. Packet dropped.")
		return
	}
}

func genTunIPNet(ip net.IP) *net.IPNet {
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(30, 32)}
}

func attachVIP(vip net.IP) (string, error) {
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
