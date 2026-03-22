package cache

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

// Serializer 序列化接口
type Serializer interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// JSONSerializer JSON 序列化（默认）
type JSONSerializer struct{}

func (JSONSerializer) Marshal(v any) ([]byte, error)        { return json.Marshal(v) }
func (JSONSerializer) Unmarshal(data []byte, v any) error   { return json.Unmarshal(data, v) }

// GOBSerializer GOB 序列化（适合 Go 内部类型，性能更好）
type GOBSerializer struct{}

func (GOBSerializer) Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (GOBSerializer) Unmarshal(data []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}
