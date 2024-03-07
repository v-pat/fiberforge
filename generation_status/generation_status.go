package generationstatus

import (
	"fmt"

	"github.com/briandowns/spinner"
	"github.com/jwalton/gchalk"
	"github.com/v-pat/fiberforge/utils"
)

var Spinner *spinner.Spinner
var LastMessage string = ""
var AppName string = ""

func UpdateGenerationStatus(message string) {
	if LastMessage != "" {
		//[✔]
		Spinner.Disable()
		gchalk.Green("[√]")
		// fmt.Println("\x1b[32m" + "[√]" + "\x1b[0m" + " " + LastMessage)
		fmt.Println(gchalk.Green("[√]") + " " + LastMessage)
		Spinner.Enable()
	}
	if LastMessage == utils.PACKAGE_IMPORT_START {
		// fmt.Println("\x1b[32m" + "[√]" + "\x1b[0m " + "\x1b[34mCode generated successfully. Please check newly created \x1b[0m" + AppName + "\x1b[34m folder in this directory.\x1b[0m")
		fmt.Println(gchalk.Green("[√] ") + gchalk.Blue("Code generated successfully. Please check newly created ") + AppName + gchalk.Blue(" folder in this directory."))
		Spinner.Stop()
	} else {
		Spinner.Suffix = " " + message
		Spinner.Restart()
		LastMessage = message
	}

}
