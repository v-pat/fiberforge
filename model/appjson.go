package model

type AppJson struct {
	AppName  string             `json:"appName"`
	Tables   []StructDefinition `json:"tables"`
	Database string             `json:"database"`
	// Language string             `json:"language"`
}

// StructDefinition represents the data required for generating CRUD methods.
type StructDefinition struct {
	StructName  string                 `json:"name"`
	JSONExample map[string]interface{} `json:"columns"`
	Endpoint    string                 `json:"endpoint"`
}

// custom errors
type Errors struct {
	ErrCode int
	Message string
}

func NewErr(msg string, errCode int) Errors {
	return Errors{
		ErrCode: errCode,
		Message: msg,
	}
}
