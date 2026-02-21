package openapi

// Config OpenAPI 文档生成配置
type Config struct {
	// Title API 标题
	Title string

	// Version API 版本号
	Version string

	// Description API 描述
	Description string

	// Path spec 端点路径，默认 "/openapi.json"
	Path string

	// SwaggerUI Swagger UI 端点路径，空字符串不启用
	SwaggerUI string

	// Servers 服务器列表
	Servers []Server

	// SecuritySchemes 安全方案定义
	SecuritySchemes map[string]SecurityScheme
}

// DefaultPath spec 端点默认路径
const DefaultPath = "/openapi.json"

// Normalize 填充默认值
func (c *Config) Normalize() {
	if c.Path == "" {
		c.Path = DefaultPath
	}
	if c.Version == "" {
		c.Version = "1.0.0"
	}
}
