package templates

// SqlDBTemplate renders the GORM database connection (postgres/mysql).
const SqlDBTemplate = `package databases

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/{{.Driver}}"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"{{.AppName}}/model"
)

// DB is the shared GORM handle.
var DB *gorm.DB

// Connect opens the database connection, configures the pool and migrates.
func Connect() error {
	dsn := buildDSN()
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{SlowThreshold: time.Second, LogLevel: logger.Warn},
	)

	db, err := gorm.Open({{.Driver}}.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	DB = db
	return migrate(db)
}

// Ping verifies database connectivity.
func Ping() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
{{range .Models}}		&model.{{.}}{},
{{end}}	)
}

func buildDSN() string {
	return fmt.Sprintf(
		"{{.Format}}",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
}
`

// MongoDBTemplate renders the mgm MongoDB connection.
const MongoDBTemplate = `package databases

import (
	"fmt"
	"os"

	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect opens the MongoDB connection via mgm.
func Connect() error {
	uri := os.Getenv("DB_URI")
	name := os.Getenv("DB_NAME")
	if uri == "" || name == "" {
		return fmt.Errorf("DB_URI and DB_NAME must be set")
	}
	return mgm.SetDefaultConfig(nil, name, options.Client().ApplyURI(uri))
}

// Ping verifies database connectivity.
func Ping() error {
	_, client, _, err := mgm.DefaultConfigs()
	if err != nil || client == nil {
		return fmt.Errorf("mongodb connection not initialized")
	}
	return client.Ping(nil, nil)
}
`
