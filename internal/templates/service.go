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

{{if .IsOwned}}
// List{{.Name}}s returns all {{.Name}} rows owned by userID.
func List{{.Name}}s(userID uint) ([]model.{{.Name}}, error) {
	var items []model.{{.Name}}
	if err := databases.DB.Where("user_id = ?", userID).Find(&items).Error; err != nil {
		return nil, errors.New("unable to list {{.Name}}s: " + err.Error())
	}
	return items, nil
}

// Get{{.Name}}ByID returns a single {{.Name}} by primary key and userID.
func Get{{.Name}}ByID(id uint, userID uint) (*model.{{.Name}}, error) {
	var m model.{{.Name}}
	if err := databases.DB.Where("id = ? AND user_id = ?", id, userID).First(&m).Error; err != nil {
		return nil, errors.New("{{.Name}} not found")
	}
	return &m, nil
}

// Update{{.Name}} updates an existing {{.Name}} by primary key and userID using validated input DTO.
func Update{{.Name}}(id uint, userID uint, patch *model.Update{{.Name}}Input) error {
	var m model.{{.Name}}
	if err := databases.DB.Where("id = ? AND user_id = ?", id, userID).First(&m).Error; err != nil {
		return errors.New("{{.Name}} not found")
	}
{{range .WritableFields}}	if patch.{{.GoName}} != nil {
		m.{{.GoName}} = *patch.{{.GoName}}
	}
{{end}}	if err := databases.DB.Save(&m).Error; err != nil {
		return errors.New("unable to update {{.Name}}: " + err.Error())
	}
	return nil
}

// Delete{{.Name}}ByID deletes a {{.Name}} by primary key and userID.
func Delete{{.Name}}ByID(id uint, userID uint) error {
	res := databases.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.{{.Name}}{})
	if res.Error != nil {
		return errors.New("unable to delete {{.Name}}: " + res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return errors.New("{{.Name}} not found")
	}
	return nil
}
{{else}}
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

// Update{{.Name}} updates an existing {{.Name}} by primary key using validated input DTO.
func Update{{.Name}}(id uint, patch *model.Update{{.Name}}Input) error {
	var m model.{{.Name}}
	if err := databases.DB.First(&m, id).Error; err != nil {
		return errors.New("{{.Name}} not found")
	}
{{range .WritableFields}}	if patch.{{.GoName}} != nil {
		m.{{.GoName}} = *patch.{{.GoName}}
	}
{{end}}	if err := databases.DB.Save(&m).Error; err != nil {
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
{{end}}
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

{{if .IsOwned}}
// List{{.Name}}s returns all {{.Name}} documents for the given user.
func List{{.Name}}s(userID string) ([]model.{{.Name}}, error) {
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	var items []model.{{.Name}}
	cursor, err := mgm.Coll(&model.{{.Name}}{}).Find(mgm.Ctx(), bson.M{"userId": oid})
	if err != nil {
		return nil, errors.New("unable to list {{.Name}}s: " + err.Error())
	}
	if err := cursor.All(mgm.Ctx(), &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Get{{.Name}}ByID returns a single {{.Name}} by id scoped to the owner.
func Get{{.Name}}ByID(id string, userID string) (*model.{{.Name}}, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid id")
	}
	userOid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	var m model.{{.Name}}
	if err := mgm.Coll(&model.{{.Name}}{}).First(bson.M{"_id": oid, "userId": userOid}, &m); err != nil {
		return nil, errors.New("{{.Name}} not found")
	}
	return &m, nil
}

// Update{{.Name}} updates an existing {{.Name}} by id scoped to the owner using validated input DTO.
func Update{{.Name}}(id string, userID string, patch *model.Update{{.Name}}Input) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid id")
	}
	userOid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user id")
	}
	var m model.{{.Name}}
	if err := mgm.Coll(&model.{{.Name}}{}).First(bson.M{"_id": oid, "userId": userOid}, &m); err != nil {
		return errors.New("{{.Name}} not found")
	}
{{range .WritableFields}}	if patch.{{.GoName}} != nil {
		m.{{.GoName}} = *patch.{{.GoName}}
	}
{{end}}	if err := mgm.Coll(&m).Update(&m); err != nil {
		return errors.New("unable to update {{.Name}}: " + err.Error())
	}
	return nil
}

// Delete{{.Name}}ByID deletes a {{.Name}} by id scoped to the owner.
func Delete{{.Name}}ByID(id string, userID string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid id")
	}
	userOid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user id")
	}
	res, err := mgm.Coll(&model.{{.Name}}{}).DeleteOne(mgm.Ctx(), bson.M{"_id": oid, "userId": userOid})
	if err != nil {
		return errors.New("unable to delete {{.Name}}: " + err.Error())
	}
	if res.DeletedCount == 0 {
		return errors.New("{{.Name}} not found")
	}
	return nil
}
{{else}}
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

// Update{{.Name}} updates an existing {{.Name}} by id using validated input DTO.
func Update{{.Name}}(id string, patch *model.Update{{.Name}}Input) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid id")
	}
	var m model.{{.Name}}
	if err := mgm.Coll(&model.{{.Name}}{}).FindByID(oid, &m); err != nil {
		return errors.New("{{.Name}} not found")
	}
{{range .WritableFields}}	if patch.{{.GoName}} != nil {
		m.{{.GoName}} = *patch.{{.GoName}}
	}
{{end}}	if err := mgm.Coll(&m).Update(&m); err != nil {
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
{{end}}
`
