package balancer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/pwpon500/caplance/balancer/backend"
	"github.com/vishvananda/netlink"
)

// Balancer is the main data struct for the load balancer
type Balancer struct {
	backends     *backend.BackendHandler // maglev hashtable of backends to their GRE device names
	vip          net.IP                  // VIP the balancer is operating off of
	connectIP    net.IP                  // IP for the RPC between backends and balancer
	packets      chan gopacket.Packet    // channel of queued up packets
	stopChan     chan os.Signal          // channel to listen for graceful stop
	listenHandle *pcap.Handle            // handle for the vip listener
}

// New creates new new Balancer. Throws error if capacity is not prime
func New(startVIP, toConnect net.IP, capacity int64) (*Balancer, error) {
	back, err := backend.New(capacity)
	if err != nil {
		return nil, err
	}

	return &Balancer{
		backends:  back,
		vip:       startVIP,
		connectIP: toConnect,
		packets:   make(chan gopacket.Packet),
		stopChan:  make(chan os.Signal)}, nil
}

// Add adds a new backend and creates a tunnel between said backend and the LB
func (b *Balancer) Add(name string, ip net.IP) error {
	if devName, err := b.backends.Get(name); err != nil && linkExists(devName) {
		fmt.Println(devName)
		return nil
	}
	ind := 0
	for linkExists("gre" + strconv.Itoa(ind)) {
		ind++
	}
	srcIP, dstIP, err := ipsFromIndex(ind)
	if err != nil {
		return err
	}
	fmt.Println(name)
	fmt.Println(strconv.Itoa(ind))
	tun := createTun(b.connectIP, ip, srcIP, dstIP, "gre"+strconv.Itoa(ind))
	return b.backends.Add(name, tun.Name)
}

// Start attaches the VIP and starts the load balancer
func (b *Balancer) Start() error {
	dev, err := attachVIP(b.vip)
	if err != nil {
		return err
	}
	signal.Notify(b.stopChan, syscall.SIGTERM)
	signal.Notify(b.stopChan, syscall.SIGINT)
	go func() {
		defer func() {
			if b.listenHandle != nil {
				b.listenHandle.Close()
			}
			os.Exit(0)
		}()
		sig := <-b.stopChan
		fmt.Printf("caught sig: %+v", sig)
	}()
	b.listen(dev)
	return nil
}

// stops the currently running lb by appending onto `stop`
func (b *Balancer) Stop() error {
	if b.listenHandle == nil {
		return errors.New("unstarted balancer cannot be stopped")
	}

	b.stopChan <- syscall.SIGTERM

	return nil
}

func ipsFromIndex(index int) (net.IP, net.IP, error) {
	if index >= 256*64 {
		return nil, nil, errors.New("More backends than can fit into consecutive 192.168.0.0/30 subnets")
	}
	src := net.IPv4(192, 168, byte(index/64), byte((4*index+1)%256))
	dest := net.IPv4(192, 168, byte(index/64), byte((4*index+2)%256))
	return src, dest, nil
}

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
	var err error
	b.listenHandle, err = pcap.OpenLive(deviceName, 1024, false, 30*time.Second)
	handleErr(err)

	filter := "dst host " + b.vip.String()
	err = b.listenHandle.SetBPFFilter(filter)
	handleErr(err)

	packetSource := gopacket.NewPacketSource(b.listenHandle, b.listenHandle.LinkType())
	for i := 0; i < 20; i++ {
		go b.handlePacket()
	}
	for packet := range packetSource.Packets() {
		b.packets <- packet
	}
}

func (b *Balancer) handlePacket() {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
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
		handle, err := pcap.OpenLive(backend, 1600, false, 30*time.Second)
		if err != nil {
			fmt.Println("Error opening requested device " + backend)
			continue
		}
		ipLayer.SerializeTo(buf, opts)
		handle.WritePacketData(buf.Bytes())
		handle.Close()
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
