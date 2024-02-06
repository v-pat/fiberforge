package templates

const EnvConfigTemplate = `
{
	{{ if eq .DbType "mysql" }}
	"Database" : "{{.User}}:{{.Password}}@tcp({{.Host}}:{{.Port}})/{{.Database}}?charset=utf8mb4&parseTime=True",
	{{ else if eq .DbType "postgres" }}
	"Database" : "postgres://{{.User}}:{{.Password}}@{{.Host}}:{{.Port}}/{{.Database}}?sslmode=disable",
	{{ end }}
	"Port" : "8080"
}

`
