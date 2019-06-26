package setups

import (
	"log"
	"net"

	"github.com/pwpon500/caplance/internal/client"
)

func setupClientSimple() {
	vip := net.ParseIP("10.0.0.50")
	dataIP := net.ParseIP("10.0.0.2")
	c := client.NewClient(vip, dataIP)
	connectIP := net.ParseIP("10.0.0.1")
	err := c.Start(connectIP)
	if err != nil {
		log.Panicln(err)
	}
}

func teardownClientSimple() {

}
