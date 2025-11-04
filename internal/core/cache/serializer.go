package cache

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

// Serializer 序列化器接口
type Serializer interface {
	// Marshal 序列化
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal 反序列化
	Unmarshal(data []byte, v interface{}) error
}

// JSONSerializer JSON 序列化器
type JSONSerializer struct{}

// Marshal 序列化
func (s *JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal 反序列化
func (s *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// GobSerializer Gob 序列化器
type GobSerializer struct{}

// Marshal 序列化
func (s *GobSerializer) Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal 反序列化
func (s *GobSerializer) Unmarshal(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(v)
}

// newSerializer 创建序列化器
func newSerializer(t SerializerType) (Serializer, error) {
	switch t {
	case SerializerJSON:
		return &JSONSerializer{}, nil
	case SerializerGob:
		return &GobSerializer{}, nil
	default:
		return &JSONSerializer{}, nil
	}
}

// Marshal 序列化（全局函数）
func Marshal(v interface{}, serializerType SerializerType) ([]byte, error) {
	s, err := newSerializer(serializerType)
	if err != nil {
		return nil, err
	}
	return s.Marshal(v)
}

// Unmarshal 反序列化（全局函数）
func Unmarshal(data []byte, v interface{}) error {
	// 默认使用 JSON
	return json.Unmarshal(data, v)
}

