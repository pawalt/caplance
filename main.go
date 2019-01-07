package main

import (
	"fmt"
	"github.com/pwpon500/caplance/util"
	"github.com/pwpon500/caplance/util/capture"
	"net"
)

func main() {
	vip := net.ParseIP("10.0.0.50")
	dev, err := util.AttachVIP(vip)
	if err != nil {
		panic(err)
	}
	fmt.Println("listening on ", vip.String())
	capture.Listen(dev, vip)
}
