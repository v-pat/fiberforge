package templates

const EnvConfigTemplate = `
{
	"Database" : "{{.User}}:{{.Password}}@tcp({{.Host}}:{{.Port}})/{{.Database}}?charset=utf8mb4&parseTime=True",
	"Port" : "8080"
}

`
