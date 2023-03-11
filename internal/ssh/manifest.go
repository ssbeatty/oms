package ssh

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// Manifest The plugin manifest.
type Manifest struct {
	DisplayName string `yaml:"name"`
	Import      string `yaml:"import"`
}

func readManifest(p string) (*Manifest, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("failed to open the plugin manifest %s: %w", p, err)
	}

	defer func() { _ = file.Close() }()

	m := &Manifest{}
	err = yaml.NewDecoder(file).Decode(m)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the plugin manifest %s: %w", p, err)
	}

	return m, nil
}
