package client

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

func (c *Client) listenUnix() {
	if _, err := os.Stat(SOCKADDR); err != nil {
		log.Panicln("could not open " + SOCKADDR + ". Error: " + err.Error())
	}

	var err error
	c.unixSock, err = net.Listen("unix", SOCKADDR)
	if err != nil {
		log.Panicln(err)
	}

	rpc.Register(c)
	rpc.HandleHTTP()
	http.Serve(c.unixSock, nil)
}

// Deregister command from caplancectl
func (c *Client) Deregister(req *string, reply *string) error {
	*reply = "Deregistering and stopping..."
	c.gracefulStop()
	return nil
}

// Pause command from caplancectl
func (c *Client) Pause(req *string, reply *string) error {
	err := c.pause()
	if err == nil {
		*reply = "Pause request sent"
	} else {
		*reply = "Pause request encountered an error: " + err.Error()
	}
	return nil
}

// Resume command from caplancectl
func (c *Client) Resume(req *string, reply *string) error {
	err := c.resume()
	if err == nil {
		*reply = "Resume request sent"
	} else {
		*reply = "Resume request encountered an error: " + err.Error()
	}
	return nil
}

// GetState command from caplancectl
func (c *Client) GetState(req *string, reply *string) error {
	*reply = stateToString(c.state)
	return nil
}
