package test

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	contextify "contextify/pkg"
)

func TestLoadConfigFromFlags(t *testing.T) {
	// Test loading from flags
	config, err := contextify.LoadConfigFromFlags("", "dir", "output.txt", "preprompt", "request", 1000, []string{"omit"})
	if err != nil {
		t.Fatal(err)
	}
	expected := contextify.Config{
		Directory:  "dir",
		TokenLimit: 1000,
		Output:     "output.txt",
		Omit:       []string{"omit"},
		Preprompt:  "preprompt\n\nRequest:\n\nrequest",
		Request:    "request",
	}
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("Expected config %v, got %v", expected, config)
	}

	// Test loading from YAML
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	configData := []byte(`directory: configdir
token_limit: 2000
output: out.txt
omit:
  - omit1
preprompt: preprompt
request: req`)
	err = ioutil.WriteFile(configFile, configData, 0644)
	if err != nil {
		t.Fatal(err)
	}
	config, err = contextify.LoadConfigFromFlags(configFile, "", "", "", "", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected = contextify.Config{
		Directory:  "configdir",
		TokenLimit: 2000,
		Output:     "out.txt",
		Omit:       []string{"omit1"},
		Preprompt:  "preprompt\n\nRequest:\n\nreq", // Corrected to match behavior
		Request:    "req",
	}
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("Expected config %v, got %v", expected, config)
	}

	// Test error: no output
	_, err = contextify.LoadConfigFromFlags("", "dir", "", "", "", 0, nil)
	if err == nil || err.Error() != "output path is required; use -o or --output to specify" {
		t.Errorf("Expected output path error, got %v", err)
	}
}
