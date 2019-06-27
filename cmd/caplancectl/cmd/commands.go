package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(deregister)
	rootCmd.AddCommand(pause)
	rootCmd.AddCommand(resume)
	rootCmd.AddCommand(getstate)
}

var deregister = &cobra.Command{
	Use:   "deregister",
	Short: "Deregister from the load balancer",
	Long: `Send a deregistration request to the server, and gracefully
	stop this instance of caplance`,
	Run: func(cmd *cobra.Command, args []string) {
		runCommand("Deregister")
	},
}

var pause = &cobra.Command{
	Use:   "pause",
	Short: "Pause the client",
	Long: `Send a pause request to the load balancer. Pause will not go into effect
	until the client gets a confirmation from the load balancer.`,
	Run: func(cmd *cobra.Command, args []string) {
		runCommand("Pause")
	},
}

var resume = &cobra.Command{
	Use:   "resume",
	Short: "resume the client",
	Long: `Send a resume request to the load balancer. Resume will not go into effect
	until the client gets a confirmation from the load balancer.`,
	Run: func(cmd *cobra.Command, args []string) {
		runCommand("Resume")
	},
}

var getstate = &cobra.Command{
	Use:   "getstate",
	Short: "Get the client state",
	Run: func(cmd *cobra.Command, args []string) {
		runCommand("GetState")
	},
}
