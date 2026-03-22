package i18n

import (
	"net/http"
	"strings"
)

// Detector 从请求中提取首选语言 tag。
// 返回空字符串表示检测失败，Bundle.ForRequest 将降级到 fallback。
type Detector interface {
	Detect(r *http.Request) string
}

// QueryDetector 从 URL 查询参数中读取语言。
type QueryDetector struct {
	Key string // 查询参数名，默认 "lang"
}

func (d QueryDetector) Detect(r *http.Request) string {
	key := d.Key
	if key == "" {
		key = "lang"
	}
	return r.URL.Query().Get(key)
}

// CookieDetector 从 Cookie 中读取语言。
type CookieDetector struct {
	Name string // Cookie 名，默认 "lang"
}

func (d CookieDetector) Detect(r *http.Request) string {
	name := d.Name
	if name == "" {
		name = "lang"
	}
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// HeaderDetector 从自定义请求头中读取语言。
type HeaderDetector struct {
	Header string // 请求头名，默认 "X-Lang"
}

func (d HeaderDetector) Detect(r *http.Request) string {
	h := d.Header
	if h == "" {
		h = "X-Lang"
	}
	return r.Header.Get(h)
}

// AcceptDetector 解析标准 Accept-Language 请求头，取最高权重的语言。
type AcceptDetector struct{}

func (d AcceptDetector) Detect(r *http.Request) string {
	header := r.Header.Get("Accept-Language")
	if header == "" {
		return ""
	}
	// 格式："zh-CN,zh;q=0.9,en;q=0.8" — 取第一个（q 最高）
	first := strings.SplitN(header, ",", 2)[0]
	lang := strings.TrimSpace(strings.SplitN(first, ";", 2)[0])
	if lang == "" || len(lang) > 35 { // BCP 47 最长 tag 约 35 字符
		return ""
	}
	return lang
}

// ChainDetector 按顺序尝试多个 Detector，返回首个非空值。
type ChainDetector struct {
	detectors []Detector
}

// Chain 创建 ChainDetector。
func Chain(detectors ...Detector) *ChainDetector {
	return &ChainDetector{detectors: detectors}
}

func (d *ChainDetector) Detect(r *http.Request) string {
	for _, det := range d.detectors {
		if lang := det.Detect(r); lang != "" {
			return lang
		}
	}
	return ""
}

// DefaultDetector 返回默认检测链：Query → Cookie → X-Lang → Accept-Language。
func DefaultDetector() Detector {
	return Chain(
		QueryDetector{},
		CookieDetector{},
		HeaderDetector{},
		AcceptDetector{},
	)
}
