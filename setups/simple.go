package setups

import (
	"net"

	"github.com/pwpon500/caplance/balancer"
)

func setupSimple() {
	vip := net.ParseIP("10.0.0.50")
	b, err := balancer.New(vip, net.ParseIP("10.0.0.1"), 53)
	if err != nil {
		panic(err)
	}
	// Add not currently supported
	/*err = b.Add("b1", net.ParseIP("10.0.0.2"))
	if err != nil {
		panic(err)
	}*/
	b.Start()
}

func teardownSimple() {

}
