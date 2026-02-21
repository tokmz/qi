package openapi

import (
	"regexp"
	"strings"
)

var versionRe = regexp.MustCompile(`^v\d+$`)

// DeriveTag 从路由组 basePath 自动推导 tag 名称
//
//	"/api/v1/users" → "users"
//	"/api/v1/users/:id/posts" → "users"
//	"/admin" → "admin"
//	"/" → ""
func DeriveTag(basePath string) string {
	parts := strings.Split(strings.Trim(basePath, "/"), "/")
	for _, p := range parts {
		if p == "" || p == "api" {
			continue
		}
		if versionRe.MatchString(p) {
			continue
		}
		if strings.HasPrefix(p, ":") || strings.HasPrefix(p, "*") {
			continue
		}
		return p
	}
	return ""
}
