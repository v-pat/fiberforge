package databases

import (
	"gorm.io/driver/postgres"

	"gorm.io/gorm"

	_ "github.com/lib/pq" // Import the database driver package
	"github.com/spf13/viper"
)

var Database *gorm.DB

func ConnectToDb() {

	// Construct the database connection URL
	dbURL := viper.Get("Database").(string)

	var err error

	Database, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})

	if err != nil {
		panic(err)
	}

	Database.AutoMigrate()

}
