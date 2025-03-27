package test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	contextify "contextify/pkg"
)

func TestLoadGitignore(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	err := ioutil.WriteFile(gitignorePath, []byte("*.log\n# comment\n.dir/\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	patterns, err := contextify.LoadGitignore(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"*.log", ".dir/"}
	if !reflect.DeepEqual(patterns, expected) {
		t.Errorf("Expected patterns %v, got %v", expected, patterns)
	}

	// Test non-existent file
	patterns, err = contextify.LoadGitignore(filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Errorf("Expected no error for non-existent .gitignore, got %v", err)
	}
	if len(patterns) != 0 {
		t.Errorf("Expected empty patterns for non-existent file, got %v", patterns)
	}
}

func TestIsIgnored(t *testing.T) {
	patterns := []string{"*.log", ".dir/"}
	tests := []struct {
		path    string
		isDir   bool
		ignored bool
	}{
		{"file.log", false, true},
		{"file.txt", false, false},
		{".dir", true, true},
		{"subdir", true, false},
	}
	for _, tt := range tests {
		result := contextify.IsIgnored(tt.path, tt.isDir, patterns)
		if result != tt.ignored {
			t.Errorf("IsIgnored(%q, %v, patterns) = %v; want %v", tt.path, tt.isDir, result, tt.ignored)
		}
	}
}

func TestGenerateTree(t *testing.T) {
	dir := t.TempDir()
	testDir := filepath.Join(dir, "testdir")
	err := os.Mkdir(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(testDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	lines := contextify.GenerateTree(testDir, []string{}, "", testDir)
	expected := []string{
		"testdir",
		"├── subdir",
		"│   └── file2.txt",
		"└── file1.txt",
	}
	if !reflect.DeepEqual(lines, expected) {
		t.Errorf("Expected tree %v, got %v", expected, lines)
	}
}

func TestIsBinaryFile(t *testing.T) {
	dir := t.TempDir()
	textFile := filepath.Join(dir, "text.txt")
	err := ioutil.WriteFile(textFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	binaryFile := filepath.Join(dir, "binary.bin")
	err = ioutil.WriteFile(binaryFile, []byte{0x00, 0x01, 0x02}, 0644)
	if err != nil {
		t.Fatal(err)
	}

	if contextify.IsBinaryFile(textFile) {
		t.Error("Expected text file to be non-binary")
	}
	if !contextify.IsBinaryFile(binaryFile) {
		t.Error("Expected binary file to be binary")
	}
}

func TestProcessDirectory(t *testing.T) {
	dir := t.TempDir()
	err := ioutil.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content1"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(dir, "file2.bin"), []byte{0x00, 0x01}, 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := contextify.Config{
		Directory:  dir,
		TokenLimit: contextify.DefaultTokenLimit,
		Output:     "output.txt",
		Omit:       []string{"*.bin"},
		Preprompt:  "Preprompt\n",
	}
	var buf bytes.Buffer
	totalChars, err := contextify.ProcessDirectory(config, &buf)
	if err != nil {
		t.Fatal(err)
	}

	expected := "\ufeffPreprompt\nDirectory structure:\n" + filepath.Base(dir) + "\n└── file1.txt\n\nFile contents:\n\n=== File: file1.txt ===\ncontent1\n\n"
	if buf.String() != expected {
		t.Errorf("Expected output %q, got %q", expected, buf.String())
	}
	if totalChars != len(expected) {
		t.Errorf("Expected totalChars %d, got %d", len(expected), totalChars)
	}
}
