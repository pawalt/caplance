package client

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/pwpon500/caplance/pkg/util"
)

// HealthState represents the current state of the client
type HealthState int

const (
	// Unregistered represents the client state before registration
	Unregistered HealthState = 0
	// Registering represents state during registration
	Registering HealthState = 1
	// Active represents state when packets are being forwarded to the client
	Active HealthState = 2
	// Paused represents state when client is registered but packets are not being forwarded
	Paused HealthState = 3
	// Deregistering represents state when client is deregistering
	Deregistering HealthState = 4
)

// Client holds the current state and configuration for a backend
type Client struct {
	dataIP       net.IP            // ip for the lb to forward packets to
	vip          net.IP            // vip for the cluster
	state        HealthState       // state of backend as described by the above consts
	comm         util.Communicator // communicator between backend and lb
	dataListener net.PacketConn    // listener for packets forwarded from lb
	name         string            // name of backend
	packets      chan *rawPacket   // channel of packets to process
	stopChan     chan os.Signal    // channel to capture SIGTERM and SIGINT for graceful stop
	unixSock     net.Listener      // unix sock for communicating with caplancectl
	readTimeout  int
	writeTimeout int
	healthRate   int
	sockaddr     string
}

// struct to hold an individual data packet recieved from lb
type rawPacket struct {
	payload []byte
	size    int
}

// NewClient creates a new Client object
func NewClient(name string, vip, dataIP net.IP, readTimeout, writeTimeout, healthRate int, sockaddr string) *Client {
	return &Client{
		dataIP:       dataIP,
		vip:          vip,
		state:        Unregistered,
		name:         name,
		packets:      make(chan *rawPacket, 100),
		stopChan:     make(chan os.Signal, 5),
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		healthRate:   healthRate,
		sockaddr:     sockaddr}
}

// Start attempts to register and listen for connections
func (c *Client) Start(connectIP net.IP) error {
	c.state = Registering
	conn, err := net.Dial("tcp", connectIP.String()+":1338")
	if err != nil {
		return err
	}
	c.comm = util.NewTCPCommunicator(conn, c.readTimeout, c.writeTimeout)

	ender := func() { c.comm.Close() }

	err = c.comm.WriteLine("REGISTER " + c.name + " " + c.dataIP.String())
	if err != nil {
		ender()
		return err
	}

	c.dataListener, err = net.ListenPacket("udp", c.dataIP.String()+":1337")
	if err != nil {
		ender()
		return err
	}

	ender = func() {
		c.comm.Close()
		c.dataListener.Close()
	}

	mtu, err := c.getMTU()
	if err != nil {
		ender()
		return err
	}

	sanityString := ""
	buf := make([]byte, mtu)
	sanityFailTime := time.Now().Add(time.Duration(c.readTimeout) * time.Second)
	for !strings.HasPrefix(sanityString, "SANITY") && !time.Now().After(sanityFailTime) {
		n, _, err := c.dataListener.ReadFrom(buf)
		if err != nil {
			ender()
			return err
		}
		sanityString = string(buf[:n])
	}

	sanitySplit := strings.Split(sanityString, " ")
	if sanitySplit[0] != "SANITY" || len(sanitySplit) < 2 {
		ender()
		return errors.New("failed to complete sanity check in " + strconv.Itoa(c.readTimeout) + " seconds.")
	}

	c.comm.WriteLine("SANE " + sanitySplit[1])

	resp, err := c.comm.ReadLine()
	if err != nil {
		ender()
		return err
	}

	tokens := strings.Split(resp, " ")
	if tokens[0] != "REGISTERED" {
		ender()
		return errors.New(resp)
	}

	c.state = Paused

	err = c.attachVIP()
	if err != nil {
		ender()
		return err
	}

	signal.Notify(c.stopChan, syscall.SIGTERM)
	signal.Notify(c.stopChan, syscall.SIGINT)
	go func() {
		defer c.gracefulStop()
		sig := <-c.stopChan
		log.Infof("caught sig: %+v \n", sig)
		c.stopChan <- sig
	}()

	var wg sync.WaitGroup
	wg.Add(3)
	go c.manageBalancerConnection(&wg)
	go c.listen(&wg)
	go c.listenUnix()
	wg.Wait()
	return nil
}

func stateToString(health HealthState) string {
	switch health {
	case Unregistered:
		return "Unregistered"
	case Registering:
		return "Registering"
	case Active:
		return "Active"
	case Paused:
		return "Paused"
	case Deregistering:
		return "Deregistering"
	}
	return "State not found"
}
