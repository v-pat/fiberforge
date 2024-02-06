package fiberforge

const (
	JSON_MATERIAL int = iota
	FILE_MATERIAL
)

type MaterialType struct {
	Material int
}

var jsonMaterial = MaterialType{
	Material: JSON_MATERIAL,
}

var fileMaterial = MaterialType{
	Material: FILE_MATERIAL,
}

func JsonMaterial() MaterialType {
	return jsonMaterial
}

func FileMaterial() MaterialType {
	return fileMaterial
}

func Forge(materialType MaterialType, material map[string]interface{}) error {
	if materialType.Material == FILE_MATERIAL {

	}

	return nil
}
