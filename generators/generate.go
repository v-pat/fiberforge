package generators

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	generationstatus "github.com/v-pat/fiberforge/generation_status"
	"github.com/v-pat/fiberforge/model"
	"github.com/v-pat/fiberforge/utils"

	"archive/zip"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var AppName string

func Generate(appJson model.AppJson, dirPath string) (string, model.Errors) {

	AppName = appJson.AppName

	err := GenerateApplicationCode(appJson, appJson.Database, dirPath)

	if err != nil {
		log.Println("Unable to generate application code : " + err.Error())
		return "", model.NewErr("Unable to generate application code : "+err.Error(), fiber.StatusInternalServerError)
	}

	generationstatus.UpdateGenerationStatus(utils.ZIP_GEN_START)

	zipFile, err := CreateApplicationZip(appJson.AppName)

	if err != nil {
		log.Println("Unable to zip application code  : " + err.Error())
		return "", model.NewErr("Unable to zip application code  : "+err.Error(), fiber.StatusInternalServerError)
	}

	err = os.RemoveAll(dirPath)
	if err != nil {
		log.Println("Unable to clean generated directory  : " + err.Error())
		return "", model.NewErr("Unable to clean generated directory  : "+err.Error(), fiber.StatusInternalServerError)
	}

	generationstatus.UpdateGenerationStatus(utils.CODE_GENERATED)

	return zipFile, model.NewErr("Code Generated Successfull.", fiber.StatusOK)
}

func createFiles(name string, dirPath string) {

	_, err := os.Stat(dirPath)
	if err != nil && !os.IsNotExist(err) {
		panic(fmt.Sprintf("An error occurred while checking if the directory exists : %s", err.Error()))
	} else {
		err := os.RemoveAll(dirPath)

		if err != nil {
			panic(fmt.Sprintf("An error occurred while deleting existing directory : %s", err.Error()))
		}
	}
	err = os.MkdirAll(dirPath, os.ModeDir)

	if err != nil {
		panic("Unable to create generated dir")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	initCommand := exec.Command("go", "mod", "init", name)

	initCommand.Dir = dirPath + "/"
	initCommand.Stdin = os.Stdin
	initCommand.Stdout = &stdout
	initCommand.Stderr = &stderr

	err = initCommand.Run()
	if err != nil {
		log.Println("Err :" + err.Error())
		panic("Go mod init failed")
	}

	err = os.MkdirAll(name+"/databases", os.ModeDir)
	if err != nil {
		panic("Unable to create databases dir")
	}

	err = os.MkdirAll(name+"/service", os.ModeDir)

	if err != nil {
		panic("Unable to create service dir")
	}

	err = os.MkdirAll(name+"/controller", os.ModeDir)
	if err != nil {
		panic("Unable to create controller dir")
	}

	err = os.MkdirAll(name+"/model", os.ModeDir)
	if err != nil {
		panic("Unable to create model dir")
	}

	err = os.MkdirAll(name+"/routes", os.ModeDir)
	if err != nil {
		panic("Unable to create model dir")
	}
	routesFile, err := os.Create(name + "/routes/routes.go")
	if err != nil {
		panic("Unable to create routes file")
	}

	defer routesFile.Close()

	mainFile, err := os.Create(name + "/main.go")
	if err != nil {
		panic("Unable to create routes file")
	}

	defer mainFile.Close()
}

func updateModFile(dirpath string) error {

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	importCommand := exec.Command("goimports", "-l", "-w", ".")

	importCommand.Dir = dirpath + "/"
	importCommand.Stdin = os.Stdin
	importCommand.Stdout = &stdout
	importCommand.Stderr = &stderr

	err := importCommand.Run()
	if err != nil {
		log.Println("Goimports failed:" + err.Error())
		return err
	}

	getCommand := exec.Command("go", "get", "-u")

	getCommand.Dir = dirpath + "/"
	getCommand.Stdin = os.Stdin
	getCommand.Stdout = &stdout
	getCommand.Stderr = &stderr

	err = getCommand.Run()
	if err != nil {
		log.Println("go get failed:" + err.Error())
		return err
	}

	return nil
}

func CreateServices(structDefs []model.StructDefinition, database string, appName string) (fiber.Map, error) {
	for _, structDef := range structDefs {
		// Generate Go struct definition
		structCode, err := GenerateStructFromJSON(structDef.ColumnsOrFields, structDef.StructName, strings.ToLower(database))
		if err != nil {
			return fiber.Map{"error": "Failed to generate struct"}, err
		}

		// Generate CRUD methods and save in service package
		serviceFileName := fmt.Sprintf(appName+"/service/%s_service.go", strings.ToLower(structDef.StructName))
		if err := GenerateServiceFile(serviceFileName, structDef, structCode, database, appName); err != nil {
			fmt.Sprintln(err.Error())
			return fiber.Map{"error": "Failed to generate service file"}, err
		}

		// Generate controller methods and save in controller package
		controllerFileName := fmt.Sprintf(appName+"/controller/%s_controller.go", strings.ToLower(structDef.StructName))
		if err := GenerateControllerFile(controllerFileName, structDef, appName, strings.ToLower(database)); err != nil {
			return fiber.Map{"error": "Failed to generate controller file"}, err
		}

	}

	return nil, nil
}

func GenerateApplicationCode(appJson model.AppJson, database string, dirPath string) error {

	AppName = appJson.AppName
	generationstatus.AppName = appJson.AppName

	// Parse the incoming JSON data as a StructDefinition
	var structDefs []model.StructDefinition

	structDefs = appJson.Tables

	// Parse the "database" query parameter
	if database != "postgres" && database != "mysql" && database != "mongodb" {
		log.Println("Unabel to process request : Invalid database type")
		return errors.New("Only supported databases are mysql, postgres and Mongodb.")
	}

	createFiles(appJson.AppName, dirPath)

	generationstatus.UpdateGenerationStatus(utils.ENVCONFIG_GEN_START)

	err := CreateConfigJsonFile(appJson.AppName, database)
	if err != nil {
		log.Println("Unabel to generate config.json : " + err.Error())
		return err
	}

	generationstatus.UpdateGenerationStatus(utils.DB_GEN_START)

	err = CreateDatabase(database, cases.Title(language.English).String(appJson.AppName), structDefs, appJson.AppName)
	if err != nil {
		panic("Unabel to create and connect to database")
	}

	generationstatus.UpdateGenerationStatus(utils.SERVICE_GEN_START)

	//creates model,controlles and service files
	_, err = CreateServices(structDefs, database, appJson.AppName)
	if err != nil {
		log.Println("Unabel to generate services : " + err.Error())
		return err
	}

	generationstatus.UpdateGenerationStatus(utils.ROUTES_GEN_START)

	// Update routes.go to define API endpoints
	err = UpdateRoutesFile(structDefs, database, appJson.AppName)
	if err != nil {
		log.Println("Unabel to generate routes : " + err.Error())
		return err
	}

	generationstatus.UpdateGenerationStatus(utils.MAIN_GEN_START)

	err = CreateMainFile(structDefs, database, appJson.AppName)
	if err != nil {
		log.Println("Unabel to generate main.go : " + err.Error())
		return err
	}

	generationstatus.UpdateGenerationStatus(utils.PACKAGE_IMPORT_START)

	err = updateModFile(dirPath)

	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	return nil
}

func CreateApplicationZip(appName string) (string, error) {
	// Directory to zip
	dirToZip := "./generated"

	// Create a temporary zip file
	zipFile := appName + ".zip"
	zipWriter, err := os.Create(zipFile)
	if err != nil {
		return "", err
	}
	defer zipWriter.Close()

	// Create a new zip archive
	zipArchive := zip.NewWriter(zipWriter)
	defer zipArchive.Close()

	// Walk through the directory and add files to zip
	err = filepath.Walk(dirToZip, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(dirToZip, filePath)
			if err != nil {
				return err
			}
			fileToZip, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer fileToZip.Close()

			// Create a new file in the zip archive
			zipFile, err := zipArchive.Create(relPath)
			if err != nil {
				return err
			}

			// Copy the file to the zip writer
			_, err = io.Copy(zipFile, fileToZip)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return zipFile, nil
}

func GenerateAndBuild(appJson model.AppJson, dirPath string) (string, model.Errors) {

	generationstatus.UpdateGenerationStatus(utils.CODEGEN_START_MSG)

	err := GenerateApplicationCode(appJson, appJson.Database, dirPath)

	if err != nil {
		log.Println("Unable to generate application code : " + err.Error())
		return "", model.NewErr("Unable to generate application code : "+err.Error(), fiber.StatusInternalServerError)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	buildCommand := exec.Command("go", "build")

	buildCommand.Dir = dirPath + "/"
	buildCommand.Stdin = os.Stdin
	buildCommand.Stdout = &stdout
	buildCommand.Stderr = &stderr

	err = buildCommand.Run()
	if err != nil {
		log.Println("Go build generated file failed:" + err.Error())
		return "", model.NewErr("Go build generated file failed:"+err.Error(), fiber.StatusInternalServerError)
	}

	err = MoveFile(dirPath+"/"+appJson.AppName+".exe", appJson.AppName+".exe")
	if err != nil {
		log.Println("Moving exe file failed:" + err.Error())
		return "", model.NewErr("Moving exe file failed:"+err.Error(), fiber.StatusInternalServerError)
	}

	err = os.RemoveAll(dirPath)
	if err != nil {
		log.Println("Unable to clean generated directory  : " + err.Error())
		return "", model.NewErr("Unable to clean generated directory  : "+err.Error(), fiber.StatusInternalServerError)
	}

	return appJson.AppName + ".exe", model.NewErr("", fiber.StatusOK)
}

func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		log.Println("Couldn't open source file: %s", err)
		return err
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		log.Println("Couldn't open dest file: %s", err)
		return err

	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		log.Println("Writing to output file failed: %s", err)
		return err

	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		log.Println("Failed removing original file: %s", err)
		return err
	}
	return nil
}
