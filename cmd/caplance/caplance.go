package main

import (
	"flag"
	"os"

	"github.com/pwpon500/caplance/pkg/setups"
)

var (
	setupName    = flag.String("setup", "", "name of test setup to create")
	teardownFlag = flag.Bool("teardown", false, "run teardown instead of build setup")
)

func main() {
	flag.Parse()
	if *setupName != "" {
		setups.RunSetup(*setupName, *teardownFlag)
		os.Exit(0)
	}
}
