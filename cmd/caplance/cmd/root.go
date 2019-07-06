package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type config struct {
	Client struct {
		ConnectIP string
		DataIP    string
		Name      string
	}
	Server struct {
		MngIP           string
		BackendCapacity int
	}
	VIP  string
	Test bool

	HealthRate   int
	ReadTimeout  int
	WriteTimeout int

	Sockaddr string
}

var (
	conf           *config
	configLocation string
)

// Execute starts the cobra chain
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/caplance/")
	viper.SetConfigName("caplance.cfg")

	viper.SetDefault("Test", false)
	viper.SetDefault("HealthRate", 20)
	viper.SetDefault("RegisterTimeout", 10)
	viper.SetDefault("ReadTimeout", 30)
	viper.SetDefault("WriteTimeout", 10)
	viper.SetDefault("Sockaddr", "/var/run/caplance.sock")

	rootCmd.PersistentFlags().StringVarP(&configLocation, "file", "f", "", "choose a non-standard config location")

}

func readConfig() {
	if configLocation != "" {
		viper.SetConfigFile(configLocation)
	}

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("Failed to read in config: " + err.Error())
	}

	conf = &config{}
	err = viper.Unmarshal(conf)
	if err != nil {
		log.Fatal("Failed to unmarshal config into struct: " + err.Error())
	}
}

var rootCmd = &cobra.Command{
	Use:   "caplancectl",
	Short: "caplancectl is the controller for caplance",
	Long:  `For more information, visit https://github.com/Pwpon500/caplance`,
}
