package openapi

import "regexp"

var (
	ginParamRe    = regexp.MustCompile(`:(\w+)`)
	ginWildcardRe = regexp.MustCompile(`\*(\w+)`)
)

// GinPathToOpenAPI 将 Gin 路径格式转换为 OpenAPI 路径格式
//
//	":id"       → "{id}"
//	"*filepath" → "{filepath}"
func GinPathToOpenAPI(ginPath string) string {
	result := ginParamRe.ReplaceAllString(ginPath, `{$1}`)
	result = ginWildcardRe.ReplaceAllString(result, `{$1}`)
	return result
}
