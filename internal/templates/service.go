package templates

// ServiceTemplate renders the SQL (GORM) service for a model.
const ServiceTemplate = `package service

import (
	"errors"

	"{{.AppName}}/databases"
	"{{.AppName}}/model"
)

// Create{{.Name}} inserts a new {{.Name}} into the database.
func Create{{.Name}}(m *model.{{.Name}}) error {
	if err := databases.DB.Create(m).Error; err != nil {
		return errors.New("unable to create {{.Name}}: " + err.Error())
	}
	return nil
}

// List{{.Name}}s returns all {{.Name}} rows.
func List{{.Name}}s() ([]model.{{.Name}}, error) {
	var items []model.{{.Name}}
	if err := databases.DB.Find(&items).Error; err != nil {
		return nil, errors.New("unable to list {{.Name}}s: " + err.Error())
	}
	return items, nil
}

// Get{{.Name}}ByID returns a single {{.Name}} by primary key.
func Get{{.Name}}ByID(id uint) (*model.{{.Name}}, error) {
	var m model.{{.Name}}
	if err := databases.DB.First(&m, id).Error; err != nil {
		return nil, errors.New("{{.Name}} not found")
	}
	return &m, nil
}

// Update{{.Name}} updates an existing {{.Name}} by primary key.
func Update{{.Name}}(id uint, patch *model.{{.Name}}) error {
	var m model.{{.Name}}
	if err := databases.DB.First(&m, id).Error; err != nil {
		return errors.New("{{.Name}} not found")
	}
	if err := databases.DB.Model(&m).Updates(patch).Error; err != nil {
		return errors.New("unable to update {{.Name}}: " + err.Error())
	}
	return nil
}

// Delete{{.Name}}ByID deletes a {{.Name}} by primary key.
func Delete{{.Name}}ByID(id uint) error {
	res := databases.DB.Delete(&model.{{.Name}}{}, id)
	if res.Error != nil {
		return errors.New("unable to delete {{.Name}}: " + res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return errors.New("{{.Name}} not found")
	}
	return nil
}
`

// MongoServiceTemplate renders the Mongo service for a model.
const MongoServiceTemplate = `package service

import (
	"errors"

	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"{{.AppName}}/model"
)

// Create{{.Name}} inserts a new {{.Name}} into the collection.
func Create{{.Name}}(m *model.{{.Name}}) error {
	if err := mgm.Coll(&model.{{.Name}}{}).Create(m); err != nil {
		return errors.New("unable to create {{.Name}}: " + err.Error())
	}
	return nil
}

// List{{.Name}}s returns all {{.Name}} documents.
func List{{.Name}}s() ([]model.{{.Name}}, error) {
	var items []model.{{.Name}}
	cursor, err := mgm.Coll(&model.{{.Name}}{}).Find(mgm.Ctx(), bson.M{})
	if err != nil {
		return nil, errors.New("unable to list {{.Name}}s: " + err.Error())
	}
	if err := cursor.All(mgm.Ctx(), &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Get{{.Name}}ByID returns a single {{.Name}} by id.
func Get{{.Name}}ByID(id string) (*model.{{.Name}}, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid id")
	}
	var m model.{{.Name}}
	if err := mgm.Coll(&model.{{.Name}}{}).FindByID(oid, &m); err != nil {
		return nil, errors.New("{{.Name}} not found")
	}
	return &m, nil
}

// Update{{.Name}} updates an existing {{.Name}} by id.
func Update{{.Name}}(id string, patch *model.{{.Name}}) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid id")
	}
	patch.SetID(oid)
	if err := mgm.Coll(patch).Update(patch); err != nil {
		return errors.New("unable to update {{.Name}}: " + err.Error())
	}
	return nil
}

// Delete{{.Name}}ByID deletes a {{.Name}} by id.
func Delete{{.Name}}ByID(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid id")
	}
	if _, err := mgm.Coll(&model.{{.Name}}{}).DeleteOne(mgm.Ctx(), bson.M{"_id": oid}); err != nil {
		return errors.New("unable to delete {{.Name}}: " + err.Error())
	}
	return nil
}
`
