package main

import (
	"fmt"
	"github.com/pwpon500/caplance/balancer"
	"net"
)

func main() {
	vip := net.ParseIP("10.0.0.50")
	b, err := balancer.New(vip, 53)
	if err != nil {
		panic(err)
	}
	err = b.Add("b1", net.ParseIP("192.168.1.1"))
	if err != nil {
		panic(err)
	}
	fmt.Println("starting")
	b.Start()
}
