package middleware

import "testing"

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "单个语言",
			header:   "zh-CN",
			expected: "zh-CN",
		},
		{
			name:     "多个语言，无 q 值",
			header:   "zh-CN,en",
			expected: "zh-CN",
		},
		{
			name:     "多个语言，带 q 值",
			header:   "zh-CN,zh;q=0.9,en;q=0.8",
			expected: "zh-CN",
		},
		{
			name:     "q 值最高的不是第一个",
			header:   "en;q=0.8,zh-CN;q=0.9",
			expected: "zh-CN",
		},
		{
			name:     "带空格",
			header:   "zh-CN, zh;q=0.9, en;q=0.8",
			expected: "zh-CN",
		},
		{
			name:     "复杂情况",
			header:   "fr;q=0.7, en;q=0.8, zh-CN;q=0.9, ja;q=0.6",
			expected: "zh-CN",
		},
		{
			name:     "空字符串",
			header:   "",
			expected: "",
		},
		{
			name:     "只有 q 值",
			header:   "en;q=1",
			expected: "en",
		},
		{
			name:     "小数点 q 值",
			header:   "en;q=0.123,zh;q=0.999",
			expected: "zh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAcceptLanguage(tt.header)
			if result != tt.expected {
				t.Errorf("parseAcceptLanguage(%q) = %q, want %q", tt.header, result, tt.expected)
			}
		})
	}
}

func TestParseQuality(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  float64
		shouldErr bool
	}{
		{
			name:      "整数 1",
			input:     "1",
			expected:  1.0,
			shouldErr: false,
		},
		{
			name:      "小数 0.9",
			input:     "0.9",
			expected:  0.9,
			shouldErr: false,
		},
		{
			name:      "小数 0.123",
			input:     "0.123",
			expected:  0.123,
			shouldErr: false,
		},
		{
			name:      "大于 1 的值",
			input:     "1.5",
			expected:  1.0,
			shouldErr: false,
		},
		{
			name:      "空字符串",
			input:     "",
			expected:  0,
			shouldErr: true,
		},
		{
			name:      "非数字",
			input:     "abc",
			expected:  0,
			shouldErr: true,
		},
		{
			name:      "多个小数点",
			input:     "0.1.2",
			expected:  0,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseQuality(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("parseQuality(%q) should return error", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseQuality(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("parseQuality(%q) = %f, want %f", tt.input, result, tt.expected)
				}
			}
		})
	}
}
