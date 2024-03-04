package cmd

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/gofiber/fiber"
	"github.com/spf13/cobra"
	generationstatus "github.com/v-pat/fiberforge/generation_status"
	"github.com/v-pat/fiberforge/generators"
	"github.com/v-pat/fiberforge/model"
	"github.com/v-pat/fiberforge/utils"
)

var generateCmd = &cobra.Command{
	Use:   "generate <file>",
	Short: "a CLI to generate a simple go fiber project",
	Long:  "generate command takes configuration from a json or text file and generate code accordingly",
	Run: func(cmd *cobra.Command, args []string) {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		generationstatus.Spinner = s
		s.Start()
		err := generateCmdHandler(args, false)
		if err.ErrCode != 200 {
			log.Println(err.Message)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func generateCmdHandler(args []string, isBuildRequired bool) model.Errors {

	// Get the file name from the command-line arguments
	fileName := args[0]

	// Check the file extension
	fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	if fileExt != "json" && fileExt != "txt" {
		log.Println("Error: Unsupported file type. Only JSON or TXT files are supported.")
		return model.NewErr("Error: Unsupported file type. Only JSON or TXT files are supported.", fiber.StatusBadRequest)
	}

	// Read the content of the file
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Println("Error reading file:", err)
		return model.NewErr("Error reading file:"+err.Error(), fiber.StatusBadRequest)
	}

	// Initialize a variable to hold the parsed config
	var appJson model.AppJson

	// Parse JSON or TXT based on the file extension
	switch fileExt {
	case "json":
		err = json.Unmarshal(data, &appJson)
		if err != nil {
			log.Println("Error parsing JSON:", err)
			return model.NewErr("Error parsing JSON: "+err.Error(), fiber.StatusBadRequest)
		}
	case "txt":
		// Assuming the TXT file contains JSON data
		err = json.Unmarshal(data, &appJson)
		if err != nil {
			log.Println("Error parsing JSON from TXT:", err)
			return model.NewErr("Error parsing JSON from TXT: "+err.Error(), fiber.StatusBadRequest)
		}
	}

	dirPath := "./" + appJson.AppName

	//call this function
	if isBuildRequired {
		_, err1 := generators.GenerateAndBuild(appJson, dirPath)
		if err != nil {
			log.Println(err1.Message)
			return err1
		}
	} else {

		err = generators.GenerateApplicationCode(appJson, appJson.Database, dirPath)
		if err != nil {
			log.Println("Error while generating application code :" + err.Error())
			return model.NewErr("Error while generating application code :"+err.Error(), fiber.StatusInternalServerError)
		}
	}

	generationstatus.UpdateGenerationStatus(utils.CODE_GENERATED)

	return model.NewErr("", fiber.StatusOK)
}
