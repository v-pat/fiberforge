package generationstatus

import "github.com/briandowns/spinner"

var Spinner *spinner.Spinner

func UpdateGenerationStatus(message string) {
	Spinner.Suffix = message
	Spinner.Restart()
}
