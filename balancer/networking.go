package balancer

import (
	"errors"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/vishvananda/netlink"
)

func linkExists(name string) bool {
	if name == "" {
		return false
	}
	interfaces, err := net.Interfaces()
	handleErr(err)
	for i := range interfaces {
		if strings.Contains(interfaces[i].Name, name) {
			return true
		}
	}
	return false
}

func initBufPool(size int) *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return make([]byte, size)
		},
	}
}

// I could be convinced to listen on more than tcp and udp, but it would have
// to be a very convincing argument. As it sits, I don't see any reason for
// listening on more than tcp and upd. AFAIK, almost all applications that could
// benefit from load balancing are over tcp or udp.
func (b *Balancer) listen(link netlink.Link) {
	toListen, err := net.ResolveIPAddr("ip4", b.vip.String())
	handleErr(err)
	tcpConn, err := net.ListenIP("ip4:tcp", toListen)
	handleErr(err)
	udpConn, err := net.ListenIP("ip4:udp", toListen)
	handleErr(err)

	mtu := link.Attrs().MTU

	pool := initBufPool(mtu)

	for i := 0; i < 20; i++ {
		go b.handlePacket(pool)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go b.listenWithConn(tcpConn, pool)
	go b.listenWithConn(udpConn, pool)
	wg.Wait()
}

func (b *Balancer) listenWithConn(conn *net.IPConn, pool *sync.Pool) {
	for {
		buf := pool.Get().([]byte)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("could not read from connection")
			continue
		}
		toSend := rawPacket{buf, n}
		b.packets <- toSend
	}
}

func (b *Balancer) handlePacket(pool *sync.Pool) {
	for {
		raw := <-b.packets
		clippedLoad := raw.payload[:raw.size]
		packet := gopacket.NewPacket(clippedLoad, layers.LayerTypeIPv4, gopacket.Lazy)

		hostPort, err := getPacketDetails(packet)
		if err != nil {
			log.Println(err)
			continue
		}
		backend, err := b.backendManager.Get(hostPort)
		if err != nil {
			log.Println("Packet received with no backends. Packet dropped.")
			continue
		}
		backend.Writer.SendData(clippedLoad)
		pool.Put(raw.payload)
	}
}

func getPacketDetails(packet gopacket.Packet) (string, error) {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return "", errors.New("couldn't find ip layer in packet")
	}
	ip, _ := ipLayer.(*layers.IPv4)

	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		udpLayer := packet.Layer(layers.LayerTypeUDP)
		if udpLayer == nil {
			return "", errors.New("couldn't find tcp or udp layer in packet")
		}
		udp, _ := udpLayer.(*layers.UDP)

		return ip.SrcIP.String() + ":" + udp.SrcPort.String(), nil
	}
	tcp, _ := tcpLayer.(*layers.TCP)

	return ip.SrcIP.String() + ":" + tcp.SrcPort.String(), nil
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
					return "", errors.New("multiple devices on the same subnet. VIP cannot be assigned")
				}
			}
		}
	}
	if foundDevice == "" {
		return "", errors.New("no device on same subnet as VIP. VIP cannot be assigned")
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
