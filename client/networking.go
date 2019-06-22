package client

import (
	"errors"
	"log"
	"net"
	"strings"
	"sync"
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

func initBufPool(size int) *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return make([]byte, size)
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
	pool := initBufPool(mtu)

	c.state = Active
	for {
		buf := pool.Get().([]byte)
		n, _, err := c.dataListener.ReadFrom(buf)
		if err != nil {
			return err
		}
		log.Println(string(buf[:n]))
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
