package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fiberforge [command]",
	Short: "A CLI to get started with go fiber app instantly",
	Long:  "Fiberforge will create a backend setup or based on your input, even generate CRUD APIs for your tables/collections.",
}

func init() {
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
