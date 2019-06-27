package setups

import (
	"fmt"
)

// RunSetup runs an appropriate testing setup
func RunSetup(setupName string, teardownFlag bool) {
	// partially apply pickSetup with teardownFlag
	pickPartial := func(s, t func()) { pickSetup(s, t, teardownFlag) }
	switch setupName {
	case "simple":
		pickPartial(setupSimple, teardownSimple)
	case "simpleClient":
		pickPartial(setupClientSimple, teardownClientSimple)
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
