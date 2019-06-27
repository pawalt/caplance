package cmd

import (
	"fmt"
	"log"
	"net/rpc"
	"os"

	"github.com/spf13/cobra"
)

const (
	SOCKADDR = "/var/sock/caplance.sock"
)

func runCommand(funcName string) {
	client, err := rpc.DialHTTP("unix", SOCKADDR)
	if err != nil {
		log.Fatal(err)
	}

	var reply string
	err = client.Call("Client."+funcName, "", &reply)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reply)
}

var rootCmd = &cobra.Command{
	Use:   "caplancectl",
	Short: "caplancectl is the controller for caplance",
	Long:  `For more information, visit https://github.com/Pwpon500/caplance`,
}

// Execute starts the cobra chain
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
