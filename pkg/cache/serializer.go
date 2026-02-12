package cache

import "encoding/json"

// JSONSerializer JSON 序列化器（默认）
type JSONSerializer struct{}

// Marshal 序列化
func (s *JSONSerializer) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal 反序列化
func (s *JSONSerializer) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
