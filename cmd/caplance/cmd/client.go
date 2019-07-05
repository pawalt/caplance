package cmd

import (
	"log"
	"net"

	"github.com/pwpon500/caplance/internal/client"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start caplance in client mode",
	Long:  `Mark this host as a backend, allowing packets to be forwarded to it from the load balancer.`,
	Run: func(cmd *cobra.Command, args []string) {
		readConfig()
		vip := net.ParseIP(conf.VIP)
		if vip == nil {
			log.Fatal("Could not parse vip: " + conf.VIP)
		}
		dataIP := net.ParseIP(conf.Client.DataIP)
		if dataIP == nil {
			log.Fatal("Could not parse data ip: " + conf.Client.DataIP)
		}
		c := client.NewClient(vip, dataIP)
		connectIP := net.ParseIP(conf.Client.ConnectIP)
		if connectIP == nil {
			connectIP = net.ParseIP(conf.Server.MngIP)
			if connectIP == nil {
				log.Fatalf("Could not parse Client.ConnectIP (%v) or Server.MngIP (%v)\n", conf.Client.ConnectIP, conf.Server.MngIP)
			}
		}
		err := c.Start(connectIP)
		if err != nil {
			log.Fatal("Failed to start with error: " + err.Error())
		}
	},
}
