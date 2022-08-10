package docs

//go:generate swag init -d ../internal/web/controllers --parseDependency -g api_v1.go -o ./

//go:generate swag init -d ../internal/web/controllers --parseDependency -g api_tool.go -o ./
