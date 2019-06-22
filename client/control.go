package client

import (
	"errors"
	"net"
	"strings"

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
	dataIP       net.IP
	vip          net.IP
	state        HealthState
	comm         util.Communicator
	dataListener net.PacketConn
	name         string
}

// NewClient creates a new Client object
func NewClient(vip, dataIP net.IP) *Client {
	return &Client{
		dataIP: dataIP,
		vip:    vip,
		state:  Unregistered}
}

// Start attempts to register and listen for connections
func (c *Client) Start(connectIP net.IP) error {
	c.state = Registering
	conn, err := net.Dial("tcp", connectIP.String()+":1338")
	if err != nil {
		return err
	}
	c.comm = util.NewTCPCommunicator(conn, READ_TIMEOUT, WRITE_TIMEOUT)

	err = c.comm.WriteLine("REGISTER " + c.name + " " + c.dataIP.String())
	if err != nil {
		return err
	}

	resp, err := c.comm.ReadLine()
	if err != nil {
		return err
	}

	tokens := strings.Split(resp, " ")
	if tokens[0] != "REGISTERED" {
		return errors.New(resp)
	}

	c.state = Paused

	return c.listen()
}
