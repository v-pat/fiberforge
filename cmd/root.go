package cmd

import (
	"github.com/spf13/cobra"
	"golang.org/x/sys/windows"
)

var rootCmd = &cobra.Command{
	Use:   "fiberforge [command]",
	Short: "A CLI to get started with go fiber app instantly",
	Long:  "Fiberforge will create a backend setup or based on your input, even generate CRUD APIs for your tables/collections.",
}

func init() {
	// Get the handle to the standard output console
	handle := windows.Handle(windows.Stdout)

	// Set console mode to enable virtual terminal processing
	var mode uint32
	windows.GetConsoleMode(handle, &mode)
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	windows.SetConsoleMode(handle, mode)
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
