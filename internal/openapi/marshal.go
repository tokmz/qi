package openapi

import (
	"encoding/json"

	gyaml "github.com/goccy/go-yaml"
)

func (m *Manager) MarshalJSON() ([]byte, error) {
	doc, err := m.Build()
	if err != nil {
		return nil, err
	}
	return MarshalJSON(doc)
}

func (m *Manager) MarshalYAML() ([]byte, error) {
	doc, err := m.Build()
	if err != nil {
		return nil, err
	}
	return MarshalYAML(doc)
}

func MarshalJSON(doc *Document) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}

func MarshalYAML(doc *Document) ([]byte, error) {
	return gyaml.Marshal(doc)
}
