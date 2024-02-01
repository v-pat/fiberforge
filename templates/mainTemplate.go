package templates

const MainTemplate = `
package main

import (
	"fmt"
	"log"
	"{{.AppName}}/routes"
	"{{.AppName}}/databases"
	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
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
	port,err := strconv.Atoi(viper.Get("Port").(string)) // Change this to your desired port
	if err!= nil{
		panic(err)
	}
	log.Fatal(app.Listen(fmt.Sprintf(":%d", port)))
}


func SetEnvVariables(){
	viper.SetConfigFile("config.json")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}


`
