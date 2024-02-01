package templates

const ServiceTemplate = `package service

import (
	"errors"
    "{{.AppName}}/databases"
    "{{.AppName}}/model"
)

{{.StructCode}}

// Create{{.StructName}} inserts a new {{.StructName}} record into the database.
func  Create{{.StructName}}({{.StructName}} model.{{.StructNameTitlecase}}) error {
    result := databases.Database.Create(&{{.StructName}})
    if result.RowsAffected == 0 || result.Error != nil {
        return errors.New("Unable to create {{.StructName}}. Please try again.")
    }
    return nil
}

// Get{{.StructName}} retrieves a {{.StructName}} record from the database by ID.
func Get{{.StructName}}ByID(id int) (model.{{.StructNameTitlecase}}, error) {

    var {{.StructName}} model.{{.StructNameTitlecase}}
    result := databases.Database.Find(&{{.StructName}}, id)

    if result.RowsAffected == 0 || result.Error != nil {
        return {{.StructName}},errors.New("{{.StructName}} not found.")
    }

    return {{.StructName}},nil
}

// Update{{.StructName}} updates an existing {{.StructName}} record in the database.
func Update{{.StructName}}({{.StructName}} model.{{.StructNameTitlecase}}, id string) error {
    result := databases.Database.Where("id = ?", id).Updates(&{{.StructName}})
    if result.RowsAffected == 0 || result.Error != nil {
        return errors.New("Unable to update {{.StructName}}. Please try again.")
    }
    return nil
}

// Delete{{.StructName}} deletes a {{.StructName}} record from the database by ID.
func Delete{{.StructName}}ByID(id int) error {
    var {{.StructName}} model.{{.StructNameTitlecase}}

    result := databases.Database.Delete(&{{.StructName}}, id)

    if result.RowsAffected == 0 || result.Error != nil {
        return errors.New("Unable to delete {{.StructName}}. Please try again.")
    }

    return nil
}

`
