package cmd

import (
	"log"
	"net"
	"strconv"

	"github.com/pwpon500/caplance/internal/balancer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start caplance in server mode",
	Long:  `Mark this host as the load balancer, forwarding packets to a set of backends.`,
	Run: func(cmd *cobra.Command, args []string) {
		readConfig()
		vip := net.ParseIP(conf.VIP)
		if vip == nil {
			log.Fatal("Could not parse vip: " + conf.VIP)
		}
		mngIP := net.ParseIP(conf.Server.MngIP)
		if mngIP == nil {
			log.Fatal("Could not parse management ip: " + conf.Server.MngIP)
		}
		if conf.Server.BackendCapacity <= 0 {
			log.Fatal("Backend capacity " + strconv.Itoa(conf.Server.BackendCapacity) + " must be postive.")
		}
		b, err := balancer.New(vip, mngIP, conf.Server.BackendCapacity)
		if err != nil {
			log.Fatal("Error when creating balancer: " + err.Error())
		}
		b.Start()
	},
}
