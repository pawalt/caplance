package client

import (
	"errors"
	"log"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/vishvananda/netlink"
)

const (
	HEALTH_RATE = 10
)

func findDevice(ip net.IP) (string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return "", err
	}
	foundDevice := ""
	for _, device := range devices {
		for _, address := range device.Addresses {
			ipNet := &net.IPNet{IP: address.IP, Mask: address.Netmask}
			if ipNet.Contains(ip) {
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
	return foundDevice, nil
}

func initPacketPool(size int) *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return &rawPacket{
				size:    0,
				payload: make([]byte, size),
			}
		},
	}
}

func (c *Client) manageBalancerConnection() {
	go c.sendHealth()
	for c.state == Active || c.state == Paused {
		message, err := c.comm.ReadLine()
		if err != nil {
			log.Println("Read timeout exceeded. Stopping")
			c.gracefulStop()
			return
		}

		tokens := strings.Split(message, " ")
		if len(tokens) < 1 {
			log.Println("Empty message received from server")
			continue
		}

		switch tokens[0] {
		case "INVALID":
			log.Println(message)

		case "DEREGISTERED":
			c.state = Deregistering
			c.gracefulStop()
			return

		case "PAUSED":
			c.state = Paused

		case "RESUMED":
			c.state = Active

		case "HEALTHACK":
			if len(tokens) < 2 {
				log.Println("HEALTHACK received from server with no status code")
			}
		default:
			log.Println("Message received from server not matching spec: " + message)
		}
	}
}

func (c *Client) sendHealth() {
	for c.state == Active || c.state == Paused {
		log.Println("sending health")
		c.comm.WriteLine("HEALTH 200")
		time.Sleep(HEALTH_RATE * time.Second)
	}
}

func (c *Client) listen() error {
	var err error
	c.dataListener, err = net.ListenPacket("udp", c.dataIP.String()+":1337")
	if err != nil {
		return err
	}

	devName, err := findDevice(c.dataIP)
	if err != nil {
		return err
	}
	link, err := netlink.LinkByName(devName)
	if err != nil {
		return err
	}

	mtu := link.Attrs().MTU
	pool := initPacketPool(mtu)

	for i := 0; i < 20; i++ {
		go c.handlePackets(pool)
	}

	c.state = Active
	for c.state == Active || c.state == Paused {
		packet := pool.Get().(*rawPacket)
		n, _, err := c.dataListener.ReadFrom(packet.payload)
		if err != nil {
			return err
		}
		packet.size = n
		c.packets <- packet
	}

	return nil
}

func (c *Client) attachVIP() error {
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return err
	}
	vipNet := &net.IPNet{IP: c.vip, Mask: net.CIDRMask(32, 32)}
	netlink.AddrAdd(lo, &netlink.Addr{IPNet: vipNet})
	return nil
}

func (c *Client) handlePackets(pool *sync.Pool) {
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

	if c.vip.To4() == nil {
		log.Panicln("vip is not ipv4")
	}

	var vipFour [4]byte
	copy(vipFour[:], c.vip[:4])

	addr := syscall.SockaddrInet4{
		Port: 0,
		Addr: vipFour,
	}

	for c.state == Active || c.state == Paused {
		packet := <-c.packets

		err := syscall.Sendto(fd, packet.payload[:packet.size], 0, &addr)
		if err != nil {
			log.Println("Failed to write packet to local vip")
		}

		pool.Put(packet)
	}
}

func (c *Client) deregister() error {
	c.state = Deregistering
	return c.comm.WriteLine("DEREGISTER " + c.name)
}

func (c *Client) gracefulStop() {
	if c.state != Deregistering {
		c.deregister()
	}
	c.comm.Close()
	c.dataListener.Close()
}
