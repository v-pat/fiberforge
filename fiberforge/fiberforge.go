package fiberforge

import (
	"errors"
	"path/filepath"

	"github.com/gofiber/fiber"
	"github.com/v-pat/fiberforge/generators"
	"github.com/v-pat/fiberforge/model"
)

const (
	JSON_MATERIAL int = iota
	FILE_MATERIAL
)

type materialType struct {
	material int
}

var JsonMaterial = materialType{
	material: JSON_MATERIAL,
}

var FileMaterial = materialType{
	material: FILE_MATERIAL,
}

type Config struct {
	AppName  string                   `json:"appName"`
	Tables   []model.StructDefinition `json:"tables"`
	Database string                   `json:"database"`
}

// custom errors
type Errors struct {
	ErrCode int
	Message string
}

func newErr(msg string, errCode int) Errors {
	return Errors{
		ErrCode: errCode,
		Message: msg,
	}
}

func Forge(material Config) error {

	configStruct := model.AppJson{
		AppName:  material.AppName,
		Tables:   material.Tables,
		Database: material.Database,
	}

	if err := gen(configStruct); err.ErrCode != 200 {
		return errors.New(err.Message)
	}

	return nil
}

func gen(material model.AppJson) Errors {
	err := generators.GenerateApplicationCode(material, material.Database, filepath.Join(".", material.AppName))
	if err != nil {
		return newErr("Error while generating application code :"+err.Error(), fiber.StatusInternalServerError)
	}
	return newErr("Code geneerated successfull.", 200)
}

func Ignite(name string, database string) error {
	configStruct := model.AppJson{
		AppName:  name,
		Tables:   make([]model.StructDefinition, 0),
		Database: database,
	}

	if err := gen(configStruct); err.ErrCode != 200 {
		return errors.New(err.Message)
	}

	return nil
}
