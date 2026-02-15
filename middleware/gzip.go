package middleware

import (
	"compress/gzip"
	"io"
	"strings"
	"sync"

	"qi"

	"github.com/gin-gonic/gin"
)

// GzipConfig Gzip 压缩中间件配置
type GzipConfig struct {
	// Level 压缩级别（默认 gzip.DefaultCompression）
	// 可选：gzip.NoCompression, gzip.BestSpeed, gzip.BestCompression, gzip.DefaultCompression
	Level int

	// MinLength 最小压缩长度（字节），小于此值不压缩（默认 256）
	MinLength int

	// ExcludePaths 排除的路径（不压缩）
	ExcludePaths []string

	// ExcludeExtensions 排除的文件扩展名（如 .png, .gif）
	ExcludeExtensions []string
}

// defaultGzipConfig 返回默认配置
func defaultGzipConfig() *GzipConfig {
	return &GzipConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 256,
	}
}

// gzipWriter 包装 gin.ResponseWriter，实现 gzip 压缩写入
type gzipWriter struct {
	gin.ResponseWriter
	writer      *gzip.Writer
	minLength   int
	buf         []byte
	wroteHeader bool
	useGzip     bool
}

// Write 写入数据
func (g *gzipWriter) Write(data []byte) (int, error) {
	if !g.wroteHeader {
		g.buf = append(g.buf, data...)
		// 缓冲区未达到最小长度，继续缓冲
		if len(g.buf) < g.minLength {
			return len(data), nil
		}
		// 达到最小长度，启用 gzip
		g.useGzip = true
		g.wroteHeader = true
		g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		g.ResponseWriter.Header().Set("Vary", "Accept-Encoding")
		g.ResponseWriter.Header().Del("Content-Length")
		// 刷出缓冲区
		_, err := g.writer.Write(g.buf)
		if err != nil {
			return 0, err
		}
		g.buf = nil
		return len(data), nil
	}
	if g.useGzip {
		return g.writer.Write(data)
	}
	return g.ResponseWriter.Write(data)
}

// WriteString 写入字符串
func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.Write([]byte(s))
}

// flush 刷出剩余缓冲区数据
func (g *gzipWriter) flush() {
	if !g.wroteHeader && len(g.buf) > 0 {
		// 数据未达到最小长度，直接写入不压缩
		g.wroteHeader = true
		_, _ = g.ResponseWriter.Write(g.buf)
		g.buf = nil
	}
}

// gzipPool gzip.Writer 对象池
var gzipPools = sync.Map{}

func getGzipPool(level int) *sync.Pool {
	if pool, ok := gzipPools.Load(level); ok {
		return pool.(*sync.Pool)
	}
	pool := &sync.Pool{
		New: func() any {
			w, _ := gzip.NewWriterLevel(io.Discard, level)
			return w
		},
	}
	actual, _ := gzipPools.LoadOrStore(level, pool)
	return actual.(*sync.Pool)
}

// Gzip 创建 Gzip 压缩中间件
// 对支持 gzip 的客户端自动压缩响应
func Gzip(cfgs ...*GzipConfig) qi.HandlerFunc {
	cfg := defaultGzipConfig()
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	}

	// 构建跳过路径 map
	skipMap := make(map[string]bool)
	for _, path := range cfg.ExcludePaths {
		skipMap[path] = true
	}

	// 构建排除扩展名 map
	extMap := make(map[string]bool)
	for _, ext := range cfg.ExcludeExtensions {
		extMap[ext] = true
	}

	pool := getGzipPool(cfg.Level)

	return func(c *qi.Context) {
		// 检查客户端是否支持 gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// 检查是否跳过
		reqPath := c.Request().URL.Path
		if skipMap[reqPath] {
			c.Next()
			return
		}

		// 检查扩展名
		if len(extMap) > 0 {
			for ext := range extMap {
				if strings.HasSuffix(reqPath, ext) {
					c.Next()
					return
				}
			}
		}

		// 从池中获取 gzip.Writer
		gz := pool.Get().(*gzip.Writer)

		// 获取底层 gin.ResponseWriter 并包装
		ginWriter := c.Writer()
		gz.Reset(ginWriter)

		gw := &gzipWriter{
			ResponseWriter: ginWriter,
			writer:         gz,
			minLength:      cfg.MinLength,
		}

		// 替换 Writer
		c.SetWriter(gw)

		c.Next()

		// 刷出剩余缓冲区
		gw.flush()

		// 关闭 gzip writer 并归还池
		if gw.useGzip {
			_ = gz.Close()
		}
		gz.Reset(io.Discard)
		pool.Put(gz)
	}
}
