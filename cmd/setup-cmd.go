package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/gofiber/fiber"
	"github.com/jwalton/gchalk"
	"github.com/spf13/cobra"
	generationstatus "github.com/v-pat/fiberforge/generation_status"
	"github.com/v-pat/fiberforge/generators"
	"github.com/v-pat/fiberforge/model"
	"github.com/v-pat/fiberforge/utils"
)

var setUpCmd = &cobra.Command{
	Use:   "setup",
	Short: "a CLI to create a boilerpate code for go fiber project",
	Long:  "setup command takes database and name of project terminal and create setup of your project",
	Run: func(cmd *cobra.Command, args []string) {
		setupCmdHandler(cmd)
	},
}

func init() {
	setUpCmd.PersistentFlags().StringP("name", "n", "", "Name of project")
	setUpCmd.PersistentFlags().StringP("db", "d", "", "Database for your project. Availabel options are mongodb, mysql and postgres.")
	rootCmd.AddCommand(setUpCmd)
}

func setupCmdHandler(cmd *cobra.Command) model.Errors {

	scanner := bufio.NewScanner(os.Stdin)

	appName, _ := cmd.Flags().GetString("name")
	db, _ := cmd.Flags().GetString("db")

	if appName == "" {

		// fmt.Println("\x1b[32m> \x1b[34mWhat is name of your application\x1b[0m?")
		// fmt.Print("\x1b[34m> \x1b[0m")

		fmt.Println(gchalk.Green("> ") + gchalk.Blue("What is name of your application") + "?")
		fmt.Print(gchalk.Blue("> "))

		scanner.Scan()

		appName = scanner.Text()
	}

	if db == "" {

		fmt.Println(gchalk.Green("> ") + gchalk.Blue("Which database you want to use") + "? available options are" + gchalk.Green(" mongodb, mysql and postgres."))
		fmt.Print(gchalk.Blue("> "))

		scanner.Scan()

		db = scanner.Text()
	}

	appJson := model.AppJson{
		AppName:  appName,
		Database: db,
		Tables:   make([]model.StructDefinition, 0),
	}

	dirPath := "./" + appName

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	generationstatus.Spinner = s
	s.Start()

	err := generators.GenerateApplicationCode(appJson, appJson.Database, dirPath)

	if err != nil {
		log.Println("Unable to generate application code : " + err.Error())
		return model.NewErr("Unable to generate application code : "+err.Error(), fiber.StatusInternalServerError)
	}

	generationstatus.UpdateGenerationStatus(utils.CODE_GENERATED)

	return model.NewErr("Code Generated Successfull.", fiber.StatusOK)
}
