package templates

// MigrateCmdTemplate renders a small migration runner command using
// golang-migrate with embedded FS (SQL databases only).
const MigrateCmdTemplate = `// cmd/migrate/main.go
package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/{{.Driver}}"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
	"gorm.io/driver/{{.Driver}}"
)

//go:embed migrations/*.sql
var fs embed.FS

func main() {
	dir := flag.String("dir", "", "sqlmigrations directory (default embedded)")
	flag.Parse()

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"),
	)

	{{if .UsePostgres}}dsn = fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"),
	)
	{{end}}db, err := gorm.Open({{.Driver}}.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, _ := db.DB()

	driver, err := {{.Driver}}driver.WithInstance(sqlDB, &{{.Driver}}driver.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var m *migrate.Migrate
	if *dir != "" {
		m, err = migrate.New("file://"+*dir, "", migrate.WithDatabase(driver))
	} else {
		src, _ := iofs.New(fs, "migrations")
		m, err = migrate.NewWithInstance("iofs", src, "", driver)
	}
	if err != nil {
		log.Fatal(err)
	}

	cmd := flag.Arg(0)
	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "version":
		v, dirty, verr := m.Version()
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
		err = verr
	default:
		log.Fatalf("usage: migrate up|down|version")
	}
	if err != nil {
		log.Fatal(err)
	}
}
`
