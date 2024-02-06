package generationstatus

import (
	"time"

	"github.com/briandowns/spinner"
)

var Spinner *spinner.Spinner

func UpdateGenerationStatus(message string) {
	Spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	Spinner.Suffix = message
	Spinner.Restart()
}
