package manifest

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Parse converts YAML data to a Manifest.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}

	if err := validate(&m); err != nil {
		return nil, err
	}

	setDefaults(&m)
	return &m, nil
}

func validate(m *Manifest) error {
	if m.Name == "" {
		return fmt.Errorf("manifest name is required")
	}
	return nil
}

func setDefaults(m *Manifest) {
	for i := range m.Actions {
		if m.Actions[i].Timeout == 0 {
			m.Actions[i].Timeout = 30 * time.Second
		}
		if m.Actions[i].Auth == "" {
			m.Actions[i].Auth = "none"
		}
		if m.Actions[i].HTTP.Method == "" {
			m.Actions[i].HTTP.Method = "POST"
		}
	}
}
