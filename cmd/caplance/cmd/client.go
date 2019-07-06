package cmd

import (
	"net"

	"github.com/pwpon500/caplance/internal/client"
	log "github.com/sirupsen/logrus"
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
		log.Infoln("Reading in config file")
		readConfig()
		vip := net.ParseIP(conf.VIP)
		if vip == nil {
			log.Fatalln("Could not parse vip: " + conf.VIP)
		}
		dataIP := net.ParseIP(conf.Client.DataIP)
		if dataIP == nil {
			log.Fatalln("Could not parse data ip: " + conf.Client.DataIP)
		}
		if conf.Client.Name == "" {
			log.Fatalln("Please provide a client name")
		}
		c := client.NewClient(conf.Client.Name, vip, dataIP, conf.ReadTimeout, conf.WriteTimeout, conf.HealthRate, conf.Sockaddr)
		connectIP := net.ParseIP(conf.Client.ConnectIP)
		if connectIP == nil {
			connectIP = net.ParseIP(conf.Server.MngIP)
			if connectIP == nil {
				log.Fatalf("Could not parse Client.ConnectIP (%v) or Server.MngIP (%v)\n", conf.Client.ConnectIP, conf.Server.MngIP)
			}
		}
		log.Infoln("Starting client")
		err := c.Start(connectIP)
		if err != nil {
			log.Fatalln("Failed to start with error: " + err.Error())
		}
	},
}
