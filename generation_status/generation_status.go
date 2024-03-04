package generationstatus

import (
	"fmt"

	"github.com/briandowns/spinner"
	"github.com/v-pat/fiberforge/utils"
)

var Spinner *spinner.Spinner
var LastMessage string = ""
var AppName string = ""

func UpdateGenerationStatus(message string) {
	if LastMessage != "" {
		//[✔]
		Spinner.Disable()
		fmt.Println("\x1b[32m" + "[√]" + "\x1b[0m" + " " + LastMessage)
		Spinner.Enable()
	}
	if LastMessage == utils.PACKAGE_IMPORT_START {
		fmt.Println("\x1b[32m" + "[√]" + "\x1b[0m " + "\x1b[34mCode generated successfully. Please check newly created \x1b[0m" + AppName + "\x1b[34m folder in this directory.\x1b[0m")
		Spinner.Stop()
	} else {
		Spinner.Suffix = " " + message
		Spinner.Restart()
		LastMessage = message
	}

}
