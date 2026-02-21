package qi

import (
	"reflect"
	"strings"

	"github.com/tokmz/qi/pkg/openapi"
)

// ============ Full: Handle[Req, Resp] — 有请求 + 有响应 ============

// GET 注册 GET 路由（泛型：有请求 + 有响应），自动收集 OpenAPI 文档
func GET[Req any, Resp any](rg *RouterGroup, path string, handler func(*Context, *Req) (*Resp, error), doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "GET", path, doc, openapi.RouteTypeFull, reflect.TypeOf((*Req)(nil)).Elem(), reflect.TypeOf((*Resp)(nil)).Elem())
	Handle[Req, Resp](rg.GET, path, handler, middlewares...)
}

// POST 注册 POST 路由（泛型：有请求 + 有响应），自动收集 OpenAPI 文档
func POST[Req any, Resp any](rg *RouterGroup, path string, handler func(*Context, *Req) (*Resp, error), doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "POST", path, doc, openapi.RouteTypeFull, reflect.TypeOf((*Req)(nil)).Elem(), reflect.TypeOf((*Resp)(nil)).Elem())
	Handle[Req, Resp](rg.POST, path, handler, middlewares...)
}

// PUT 注册 PUT 路由（泛型：有请求 + 有响应），自动收集 OpenAPI 文档
func PUT[Req any, Resp any](rg *RouterGroup, path string, handler func(*Context, *Req) (*Resp, error), doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "PUT", path, doc, openapi.RouteTypeFull, reflect.TypeOf((*Req)(nil)).Elem(), reflect.TypeOf((*Resp)(nil)).Elem())
	Handle[Req, Resp](rg.PUT, path, handler, middlewares...)
}

// PATCH 注册 PATCH 路由（泛型：有请求 + 有响应），自动收集 OpenAPI 文档
func PATCH[Req any, Resp any](rg *RouterGroup, path string, handler func(*Context, *Req) (*Resp, error), doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "PATCH", path, doc, openapi.RouteTypeFull, reflect.TypeOf((*Req)(nil)).Elem(), reflect.TypeOf((*Resp)(nil)).Elem())
	Handle[Req, Resp](rg.PATCH, path, handler, middlewares...)
}

// DELETE 注册 DELETE 路由（泛型：有请求 + 有响应），自动收集 OpenAPI 文档
func DELETE[Req any, Resp any](rg *RouterGroup, path string, handler func(*Context, *Req) (*Resp, error), doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "DELETE", path, doc, openapi.RouteTypeFull, reflect.TypeOf((*Req)(nil)).Elem(), reflect.TypeOf((*Resp)(nil)).Elem())
	Handle[Req, Resp](rg.DELETE, path, handler, middlewares...)
}

// ============ Request-only: Handle0[Req] — 有请求，无响应数据 ============

// GET0 注册 GET 路由（泛型：有请求，无响应数据），自动收集 OpenAPI 文档
func GET0[Req any](rg *RouterGroup, path string, handler func(*Context, *Req) error, doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "GET", path, doc, openapi.RouteTypeRequestOnly, reflect.TypeOf((*Req)(nil)).Elem(), nil)
	Handle0[Req](rg.GET, path, handler, middlewares...)
}

// POST0 注册 POST 路由（泛型：有请求，无响应数据），自动收集 OpenAPI 文档
func POST0[Req any](rg *RouterGroup, path string, handler func(*Context, *Req) error, doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "POST", path, doc, openapi.RouteTypeRequestOnly, reflect.TypeOf((*Req)(nil)).Elem(), nil)
	Handle0[Req](rg.POST, path, handler, middlewares...)
}

// PUT0 注册 PUT 路由（泛型：有请求，无响应数据），自动收集 OpenAPI 文档
func PUT0[Req any](rg *RouterGroup, path string, handler func(*Context, *Req) error, doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "PUT", path, doc, openapi.RouteTypeRequestOnly, reflect.TypeOf((*Req)(nil)).Elem(), nil)
	Handle0[Req](rg.PUT, path, handler, middlewares...)
}

// PATCH0 注册 PATCH 路由（泛型：有请求，无响应数据），自动收集 OpenAPI 文档
func PATCH0[Req any](rg *RouterGroup, path string, handler func(*Context, *Req) error, doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "PATCH", path, doc, openapi.RouteTypeRequestOnly, reflect.TypeOf((*Req)(nil)).Elem(), nil)
	Handle0[Req](rg.PATCH, path, handler, middlewares...)
}

// DELETE0 注册 DELETE 路由（泛型：有请求，无响应数据），自动收集 OpenAPI 文档
func DELETE0[Req any](rg *RouterGroup, path string, handler func(*Context, *Req) error, doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "DELETE", path, doc, openapi.RouteTypeRequestOnly, reflect.TypeOf((*Req)(nil)).Elem(), nil)
	Handle0[Req](rg.DELETE, path, handler, middlewares...)
}

// ============ Response-only: HandleOnly[Resp] — 无请求，有响应 ============

// GETOnly 注册 GET 路由（泛型：无请求，有响应），自动收集 OpenAPI 文档
func GETOnly[Resp any](rg *RouterGroup, path string, handler func(*Context) (*Resp, error), doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "GET", path, doc, openapi.RouteTypeResponseOnly, nil, reflect.TypeOf((*Resp)(nil)).Elem())
	HandleOnly[Resp](rg.GET, path, handler, middlewares...)
}

// POSTOnly 注册 POST 路由（泛型：无请求，有响应），自动收集 OpenAPI 文档
func POSTOnly[Resp any](rg *RouterGroup, path string, handler func(*Context) (*Resp, error), doc *openapi.DocOption, middlewares ...HandlerFunc) {
	registerGeneric(rg, "POST", path, doc, openapi.RouteTypeResponseOnly, nil, reflect.TypeOf((*Resp)(nil)).Elem())
	HandleOnly[Resp](rg.POST, path, handler, middlewares...)
}

// ============ 内部注册函数 ============

// registerGeneric 将泛型路由元数据注册到 registry
func registerGeneric(rg *RouterGroup, method, path string, doc *openapi.DocOption, routeType openapi.RouteType, reqType, respType reflect.Type) {
	if rg.registry == nil {
		return
	}

	fullPath := strings.TrimRight(rg.group.BasePath(), "/") + path

	rg.registry.Add(openapi.RouteEntry{
		Method:          method,
		Path:            fullPath,
		BasePath:        rg.group.BasePath(),
		Type:            routeType,
		ReqType:         reqType,
		RespType:        respType,
		Doc:             doc,
		DefaultTag:      rg.defaultTag,
		DefaultTagDesc:  rg.defaultTagDesc,
		DefaultSecurity: rg.defaultSecurity,
	})
}
