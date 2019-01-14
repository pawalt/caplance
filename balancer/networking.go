package balancer

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/pwpon500/caplance/balancer/backend"
	"github.com/vishvananda/netlink"
	"net"
	"time"
)

type Balancer struct {
	backends *backend.BackendHandler
	vip      net.IP
	packets  chan gopacket.Packet
}

func New(startVIP net.IP, capacity int64) (*Balancer, error) {
	back, err := backend.New(capacity)
	if err != nil {
		return nil, err
	}
	return &Balancer{backends: back, vip: startVIP, packets: make(chan gopacket.Packet)}, nil
}

func (b *Balancer) Add(name string, ip net.IP) error {
	return b.backends.Add(name, ip)
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
	for i := 0; i < 20; i++ {
		go b.handlePacket()
	}
	for packet := range packetSource.Packets() {
		b.packets <- packet
	}
}

func (b *Balancer) handlePacket() {
	for {
		packet := <-b.packets
		hostPort, ipLayer := getPacketDetails(packet)
		if hostPort == "" {
			continue
		}
		backend, err := b.backends.Get(hostPort)
		if err != nil {
			fmt.Println("Packet received with no backends. Packet dropped.")
			continue
		}
		fmt.Println(backend)
		toWrite := []gopacket.SerializableLayer{
			&layers.Ethernet{
				SrcMAC:       net.HardwareAddr{142, 122, 18, 195, 169, 113},
				DstMAC:       net.HardwareAddr{58, 86, 107, 105, 89, 94},
				EthernetType: layers.EthernetTypeIPv4,
			},
			&layers.IPv4{
				Version:  4,
				SrcIP:    net.IP{192, 168, 1, 1},
				DstIP:    net.IP{192, 168, 1, 2},
				Protocol: layers.IPProtocolGRE,
				TTL:      64,
				IHL:      5,
			},
			&layers.GRE{
				Protocol: layers.EthernetTypeIPv4,
			},
			ipLayer,
		}
		fmt.Println(toWrite)
	}
}

func getPacketDetails(packet gopacket.Packet) (string, *layers.IPv4) {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return "", nil
	}
	ip, _ := ipLayer.(*layers.IPv4)

	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		udpLayer := packet.Layer(layers.LayerTypeUDP)
		if udpLayer == nil {
			return "", nil
		}
		udp, _ := udpLayer.(*layers.UDP)

		return ip.SrcIP.String() + ":" + string(udp.SrcPort), ip
	}
	tcp, _ := tcpLayer.(*layers.TCP)

	return ip.SrcIP.String() + ":" + string(tcp.SrcPort), ip
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
				} else if foundDevice != device.Name {
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