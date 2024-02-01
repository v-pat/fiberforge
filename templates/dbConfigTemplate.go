package templates

const DatabaseConnectionTemplate = `
package databases

import (
	"fmt"
	{{if eq .DatabaseDriverName "postgres"}}
	"gorm.io/driver/postgres"
	{{else}}
    "gorm.io/driver/mysql"
	{{end}}
    "gorm.io/gorm"
	"{{.AppName}}/model"

	_ "{{.DatabaseDriver}}" // Import the database driver package
)

var  Database *gorm.DB



func ConnectToDb() {
	
	// Construct the database connection URL
	dbURL := viper.Get("Database").(string)


    var err error

    Database, err = gorm.Open({{.DatabaseDriverName}}.Open(dbURL), &gorm.Config{
        SkipDefaultTransaction: true,
        PrepareStmt:            true,
    })

    if err != nil {
        panic(err)
    }

    Database.AutoMigrate(
		{{range .StructNames}}
		&model.{{.StructName}}{}, 
	{{end}}

	)

}


`

/*
func ConnectToDatabase() (*sql.DB, error) {
	// Define the database connection parameters
	dbHost := "{{.DBHost}}"
	dbPort := "{{.DBPort}}"
	dbName := "{{.DBName}}"
	dbUser := "{{.DBUser}}"
	dbPassword := "{{.DBPassword}}"

	// Construct the database connection URL
	dbURL := fmt.Sprintf("{{.DBURLFormat}}", dbUser, dbPassword, dbHost, dbPort, dbName)

	// Open a database connection
	db, err := sql.Open("{{.DatabaseDriverName}}", dbURL)
	if err != nil {
		return nil, err
	}

	// Test the database connection
	if err = db.Ping(); err != nil {
		return nil, err
	}

	{{.DBName}} = db



	return db, nil
}
*/

// func main() {
// 	// Create the database
// 	err := createDatabase()
// 	if err != nil {
// 		fmt.Println("Failed to create the database:", err)
// 		return
// 	}
// 	fmt.Println("Database created successfully.")

// 	// Create and connect to the database
// 	db, err := connectToDatabase()
// 	if err != nil {
// 		fmt.Println("Failed to connect to the database:", err)
// 		return
// 	}
// 	defer db.Close()

// 	fmt.Println("Connected to the database successfully")

// 	// Now, you can use 'db' to perform database operations
// }
