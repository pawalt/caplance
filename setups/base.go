package setups

import (
	"fmt"
)

// RunSetup runs an appropriate testing setup
func RunSetup(setupName string, teardownFlag bool) {
	switch setupName {
	case "simple":
		pickSetup(setupSimple, teardownSimple, teardownFlag)
	default:
		fmt.Println("Setup name" + setupName + " not found")
	}
}

func pickSetup(setup, teardown func(), teardownFlag bool) {
	if teardownFlag {
		fmt.Println("tearing down setup")
		teardown()
	} else {
		fmt.Println("starting setup")
		setup()
	}
}
