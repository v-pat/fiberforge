package main

import (
	"fmt"
	"log"
	"strconv"
	"vpat/databases"
	"vpat/routes"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	// Add other necessary imports
)

func main() {

	//Setup env variables
	SetEnvVariables()

	// Setup database connection
	databases.ConnectToDb()

	// Create a new Fiber app
	app := fiber.New()

	// Setup routes
	routes.Routes(app)

	// Start the server
	port, err := strconv.Atoi(viper.Get("Port").(string)) // Change this to your desired port
	if err != nil {
		panic(err)
	}
	log.Fatal(app.Listen(fmt.Sprintf(":%d", port)))
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
