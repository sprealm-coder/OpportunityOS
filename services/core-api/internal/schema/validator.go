package schema

import (
	"encoding/json"
	"fmt"
)

type Definition map[string]any

func Parse(raw []byte) (Definition, error) {
	var definition Definition
	if err := json.Unmarshal(raw, &definition); err != nil {
		return nil, fmt.Errorf("invalid JSON schema: %w", err)
	}
	if err := Validate(definition); err != nil {
		return nil, err
	}
	return definition, nil
}

func Validate(definition Definition) error {
	if definition["type"] != "object" {
		return fmt.Errorf("root schema type must be object")
	}
	properties, ok := definition["properties"].(map[string]any)
	if !ok {
		return fmt.Errorf("schema properties must be an object")
	}
	for name, raw := range properties {
		property, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("property %s must be an object", name)
		}
		typeName, ok := property["type"].(string)
		if !ok || typeName == "" {
			return fmt.Errorf("property %s must declare a type", name)
		}
	}
	return nil
}
