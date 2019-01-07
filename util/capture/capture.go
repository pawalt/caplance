package capture

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"net"
	"time"
)

func Listen(deviceName string, vip net.IP) {
	handle, err := pcap.OpenLive(deviceName, 1024, false, 30*time.Second)
	handleErr(err)
	defer handle.Close()

	filter := "dst host " + vip.String()
	err = handle.SetBPFFilter(filter)
	handleErr(err)

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		fmt.Println(packet)
	}
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
