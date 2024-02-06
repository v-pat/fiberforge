package generationstatus

import (
	"fmt"

	"github.com/briandowns/spinner"
)

var Spinner *spinner.Spinner
var LastMessage string = ""

func UpdateGenerationStatus(message string) {
	if LastMessage != "" {
		Spinner.Disable()
		fmt.Println("\x1b[32m[✔]\x1b[0m " + LastMessage)
		Spinner.Enable()
	}
	Spinner.Suffix = " " + message
	Spinner.Restart()
	LastMessage = message
}
