package qi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"", ""},
		{"en-US", "en-US"},
		{"zh-CN,en-US;q=0.9", "zh-CN"},
		{"en-US;q=0.9,zh-CN;q=0.8", "en-US"},
		{"fr-FR;q=0.7", "fr-FR"},
		{" ja-JP , en-US", "ja-JP"},
	}
	for _, tt := range tests {
		got := parseAcceptLanguage(tt.header)
		if got != tt.want {
			t.Errorf("parseAcceptLanguage(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestI18nMiddlewareLanguageDetection(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		xLang    string
		accept   string
		wantLang string
	}{
		{"query param", "?lang=en-US", "", "", "en-US"},
		{"x-language header", "", "zh-CN", "", "zh-CN"},
		{"accept-language header", "", "", "ja-JP,en-US;q=0.9", "ja-JP"},
		{"query overrides header", "?lang=fr-FR", "zh-CN", "en-US", "fr-FR"},
		{"x-language overrides accept", "", "ko-KR", "en-US", "ko-KR"},
		{"no language", "", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var detectedLang string

			g := gin.New()
			// 使用一个 mock translator（nil 即可，中间件只做语言检测）
			g.Use(wrap(i18nMiddleware(nil)))
			g.GET("/test", func(c *gin.Context) {
				ctx := newContext(c)
				detectedLang = GetContextLanguage(ctx)
				c.Status(http.StatusOK)
			})

			url := "/test" + tt.query
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.xLang != "" {
				req.Header.Set("X-Language", tt.xLang)
			}
			if tt.accept != "" {
				req.Header.Set("Accept-Language", tt.accept)
			}

			w := httptest.NewRecorder()
			g.ServeHTTP(w, req)

			if detectedLang != tt.wantLang {
				t.Errorf("detected language = %q, want %q", detectedLang, tt.wantLang)
			}
		})
	}
}

func TestContextT_NoTranslator(t *testing.T) {
	g := gin.New()
	g.GET("/test", func(c *gin.Context) {
		ctx := newContext(c)
		result := ctx.T("hello.world")
		if result != "hello.world" {
			t.Errorf("T() without translator = %q, want %q", result, "hello.world")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
}

func TestContextTn_NoTranslator(t *testing.T) {
	g := gin.New()
	g.GET("/test", func(c *gin.Context) {
		ctx := newContext(c)
		result := ctx.Tn("item.one", "item.many", 5)
		if result != "item.one" {
			t.Errorf("Tn() without translator = %q, want %q", result, "item.one")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
}
