package client

import (
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pwpon500/caplance/util"
)

// HealthState represents the current state of the client
type HealthState int

const (
	REGISTER_TIMEOUT = 10
	READ_TIMEOUT     = 20
	WRITE_TIMEOUT    = 5
)

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
}

// struct to hold an individual data packet recieved from lb
type rawPacket struct {
	payload []byte
	size    int
}

// NewClient creates a new Client object
func NewClient(vip, dataIP net.IP) *Client {
	return &Client{
		dataIP:   dataIP,
		vip:      vip,
		state:    Unregistered,
		packets:  make(chan *rawPacket, 100),
		stopChan: make(chan os.Signal, 5)}
}

// Start attempts to register and listen for connections
func (c *Client) Start(connectIP net.IP) error {
	c.state = Registering
	conn, err := net.Dial("tcp", connectIP.String()+":1338")
	if err != nil {
		return err
	}
	c.comm = util.NewTCPCommunicator(conn, READ_TIMEOUT, WRITE_TIMEOUT)

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
	sanityFailTime := time.Now().Add(READ_TIMEOUT * time.Second)
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
		return errors.New("failed to complete sanity check in " + strconv.Itoa(READ_TIMEOUT) + " seconds.")
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
		log.Printf("caught sig: %+v \n", sig)
		c.stopChan <- sig
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go c.manageBalancerConnection(&wg)
	go c.listen(&wg)
	wg.Wait()
	return nil
}
