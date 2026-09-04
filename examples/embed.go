package examples

import (
	"embed"
	"fmt"
)

//go:embed *.yaml
var Files embed.FS

// Get returns the content of an example schema by name (e.g. "blog").
func Get(name string) ([]byte, error) {
	content, err := Files.ReadFile(name + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("template '%s' not found", name)
	}
	return content, nil
}
