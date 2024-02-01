package server

import (
	"fmt"
	"os"

	generator "github.com/v-pat/fiberforge/generators"
	"github.com/v-pat/fiberforge/model"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func Serve() {
	app := fiber.New()

	// API endpoint to receive StructDefinition with BodyParser and Database with QueryParser
	app.Post("/generate", Handler)

	// Start the Fiber server on port 3000
	app.Listen(":3000")
}

func Handler(c *fiber.Ctx) error {
	var appJson model.AppJson
	if err := c.BodyParser(&appJson); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	dirPath := viper.Get("dirPath").(string)

	zipFile, errs := generator.Generate(appJson, dirPath)

	if errs.ErrCode != fiber.StatusOK {
		fmt.Println(errs.Message)
		return c.Status(errs.ErrCode).SendString(errs.Message)
	}

	// Set appropriate headers for download
	c.Set("Content-Disposition", "attachment; filename="+appJson.AppName+".zip")
	c.Set("Content-Type", "application/zip")

	file, err := os.ReadFile(zipFile)
	if err != nil {
		fmt.Println("Unable to read generated zip file  : " + err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Unable to read generated zip file  : " + err.Error())
	}

	// Send the zip file as response
	c.Response().SetBodyRaw(file)

	err = os.Remove(zipFile)
	if err != nil {
		fmt.Println("Unable to delete generated zip file  : " + err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
