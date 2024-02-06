package templates

const ServiceTemplate = `package service



{{if eq .DbType "mongodb"}}

import (
    "errors"
    "{{.AppName}}/databases"
    "{{.AppName}}/model"
    "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
    "github.com/kamva/mgm/v3"
)


// Create{{.StructName}} inserts a new {{.StructName}} record into the database.
func  Create{{.StructName}}({{.StructName}} model.{{.StructNameTitlecase}}) error {

    _,err := mgm.Coll(&model.{{.StructNameTitlecase}}{}).InsertOne(mgm.Ctx(),{{.StructName}})

    if err!=nil{
        return errors.New("Unable to create {{.StructName}}. Please try again.")
    }

    return nil
}

// GetAll{{.StructNameTitlecase}}s retrieves all rows in {{.StructName}} table.
func GetAll{{.StructNameTitlecase}}s() ([]model.{{.StructNameTitlecase}},error) {

	var {{.StructName}}s []model.{{.StructNameTitlecase}}

    cursor,err := mgm.Coll(&model.{{.StructNameTitlecase}}{}).Find(mgm.Ctx(),bson.M{})

    if err!=nil{
        return nil,errors.New("Unable to create {{.StructName}}. Please try again.")
    }

    // Query for return all {{.StructName}}s.
	err = cursor.All(mgm.Ctx(), &{{.StructName}}s)
	if err != nil {
		return nil, err
	}

    return {{.StructName}}s,nil
}

// Get{{.StructName}} retrieves a {{.StructName}} record from the database by ID.
func Get{{.StructName}}ByID(id string) (model.{{.StructNameTitlecase}}, error) {

    var {{.StructName}} model.{{.StructNameTitlecase}}

    err := mgm.Coll(&model.{{.StructNameTitlecase}}{}).FindByID(id,&{{.StructName}})

    if err != nil {
        return {{.StructName}},errors.New("{{.StructName}} not found.")
    }

    return {{.StructName}},nil
}

// Update{{.StructName}} updates an existing {{.StructName}} record in the database.
func Update{{.StructName}}({{.StructName}} model.{{.StructNameTitlecase}}, id string) error {

    doc_id,err := primitive.ObjectIDFromHex(id)
    if err != nil{
        return errors.New("Provide id is not valid. Please try again.")
    }
    _,err = mgm.Coll(& model.{{.StructNameTitlecase}}{}).UpdateByID(mgm.Ctx(),doc_id,bson.M{"$set":{{.StructName}}})
    if err != nil {
        return errors.New("Unable to update {{.StructName}}. Please try again.")
    }
    return nil
}

// Delete{{.StructName}} deletes a {{.StructName}} record from the database by ID.
func Delete{{.StructName}}ByID(id string) error {

    doc_id,err := primitive.ObjectIDFromHex(id)
    if err != nil{
        return errors.New("Provide id is not valid. Please try again.")
    }

    _,err = mgm.Coll(& model.{{.StructNameTitlecase}}{}).DeleteOne(mgm.Ctx(),bson.M{"_id":doc_id})


    if err != nil {
        return errors.New("Unable to delete {{.StructName}}. Please try again.")
    }

    return nil
}




{{else}}

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

// GetAll{{.StructNameTitlecase}}s retrieves all rows in {{.StructName}} table.
func GetAll{{.StructNameTitlecase}}s() ([]model.{{.StructNameTitlecase}},error) {
	var {{.StructName}}s []model.{{.StructNameTitlecase}}

    result := databases.Database.Find(&{{.StructName}}s)

    if result.RowsAffected == 0 || result.Error != nil {
        return {{.StructName}}s,errors.New("{{.StructName}} table not found.")
    }

	return {{.StructName}}s,nil
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

{{end}}

`
