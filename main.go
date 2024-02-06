package main

import (
	"encoding/json"
	"log"
	"time"

	"os"
	"path/filepath"
	"strings"

	generationstatus "github.com/v-pat/fiberforge/generation_status"
	"github.com/v-pat/fiberforge/generators"
	"github.com/v-pat/fiberforge/model"
	"golang.org/x/sys/windows"

	"github.com/gofiber/fiber/v2"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

func main() {
	err := generateCmd.Execute()
	if err != nil {
		panic(err)
	}
}

func CmdHandler(args []string, isBuildRequired bool) model.Errors {

	// Get the file name from the command-line arguments
	fileName := args[1]

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

	dirPath := "./generated"

	//call this function
	if isBuildRequired {
		_, err1 := generators.GenerateAndBuild(appJson, dirPath)
		if err != nil {
			log.Println(err1.Message)
			return err1
		}
	} else {

		_, err1 := generators.Generate(appJson, dirPath)
		if err1.ErrCode != 200 {
			log.Println(err1.Message)
			return err1
		}
	}
	return model.NewErr("", fiber.StatusOK)
}

var generateCmd = &cobra.Command{
	Use:   "generate <file>",
	Short: "generate - a CLI to generate a simple go fiber project",
	Long:  "generate - takes configuration from a json or text file and generate code accordingly",
	Run: func(cmd *cobra.Command, args []string) {
		Execute(args, cmd)
	},
}

func Execute(args []string, cmd *cobra.Command) {

	// Get the handle to the standard output console
	handle := windows.Handle(windows.Stdout)

	// Set console mode to enable virtual terminal processing
	var mode uint32
	windows.GetConsoleMode(handle, &mode)
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	windows.SetConsoleMode(handle, mode)

	//Uncoment the lines below to use as server
	// if len(args) == 0 {
	// 	server.Serve()
	// } else
	if len(args) == 2 && args[0] == "generate" {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		generationstatus.Spinner = s
		go s.Start()
		err := CmdHandler(args, false)
		s.Stop()
		if err.ErrCode != 200 {
			log.Println(err.Message)
		}
	} else if len(args) == 3 && args[0] == "generate" && args[2] == "build" {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		generationstatus.Spinner = s
		s.Start()
		err := CmdHandler(args, true)
		s.Stop()
		if err.ErrCode != 200 {
			log.Println(err.Message)
		}
	} else {
		log.Println("Allowed command is : generate <config_file>")
	}
}
