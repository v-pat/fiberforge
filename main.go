package main

import (
	"encoding/json"
	"fmt"
	"time"

	"os"
	"path/filepath"
	"strings"

	generationstatus "github.com/v-pat/fiberforge/generation_status"
	"github.com/v-pat/fiberforge/generators"
	"github.com/v-pat/fiberforge/model"
	"github.com/v-pat/fiberforge/server"

	"github.com/gofiber/fiber/v2"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	SetEnvVariables()
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

func CmdHandler(args []string) model.Errors {

	// Get the file name from the command-line arguments
	fileName := args[1]

	// Check the file extension
	fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	if fileExt != "json" && fileExt != "txt" {
		fmt.Println("Error: Unsupported file type. Only JSON or TXT files are supported.")
		return model.NewErr("Error: Unsupported file type. Only JSON or TXT files are supported.", fiber.StatusBadRequest)
	}

	// Read the content of the file
	data, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return model.NewErr("Error reading file:"+err.Error(), fiber.StatusBadRequest)
	}

	// Initialize a variable to hold the parsed config
	var appJson model.AppJson

	// Parse JSON or TXT based on the file extension
	switch fileExt {
	case "json":
		err = json.Unmarshal(data, &appJson)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			return model.NewErr("Error parsing JSON: "+err.Error(), fiber.StatusBadRequest)
		}
	case "txt":
		// Assuming the TXT file contains JSON data
		err = json.Unmarshal(data, &appJson)
		if err != nil {
			fmt.Println("Error parsing JSON from TXT:", err)
			return model.NewErr("Error parsing JSON from TXT: "+err.Error(), fiber.StatusBadRequest)
		}
	}

	dirPath := viper.Get("dirPath").(string)

	//call this function
	_, err1 := generators.Generate(appJson, dirPath)
	if err1.ErrCode != 200 {
		fmt.Println(err1.Message)
		return err1
	}
	return model.NewErr("", fiber.StatusOK)
}

func SetEnvVariables() {
	viper.SetConfigFile("config.json")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

}

var rootCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate - a CLI to generate a simple go fiber project",
	Long:  "generate - takes configuration from a json or text file and generate code accordingly",
	Run: func(cmd *cobra.Command, args []string) {
		Execute(args)
	},
}

func Execute(args []string) {
	if len(args) == 0 {
		server.Serve()
	} else if len(args) == 2 && args[0] == "generate" {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		generationstatus.Spinner = s
		s.Start()
		err := CmdHandler(args)
		s.Stop()
		if err.ErrCode != 200 {
			fmt.Println(err.Message)
		}
	} else {
		fmt.Println("Allowed command is : generate <config_file>")
	}
}
