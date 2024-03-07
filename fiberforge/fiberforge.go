package fiberforge

const (
	JSON_MATERIAL int = iota
	FILE_MATERIAL
)

type materialType struct {
	material int
}

var JsonMaterial = materialType{
	material: JSON_MATERIAL,
}

var FileMaterial = materialType{
	material: FILE_MATERIAL,
}

func Forge(materialType materialType, material map[string]interface{}) error {
	if materialType.material == FILE_MATERIAL {

	}

	return nil
}

func Ignite() {

}
