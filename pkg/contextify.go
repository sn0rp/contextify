package contextify

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cheggaaa/pb/v3"
)

// Constants used across the package
const (
	DefaultTokenLimit = 128000
	CharPerToken      = 4
)

const DefaultPreprompt = `I have dumped the entire context of my code base, and I have a specific request. Please come up with a proposal to address my request - including the code and general approach.

Ensure that you leave no details out, and specifically follow my requirements. I know what I am doing, and you can assume that there is a reason for my arbitrary requirements.

When generating the full instructions with all of the details, keep in mind that I require very specific, step-by-step instructions. Come up with discrete steps such that I can build incrementally and verify success at each step, keeping your response concise.

Request:

<request>

The entire codebase is pasted below as context:

`

// LoadConfigFromFlags constructs a Config from flag values or a YAML file
func LoadConfigFromFlags(configFlag, directoryFlag, outputFlag, prepromptFlag, requestFlag string, tokenLimitFlag int, skipFlags []string) (Config, error) {
	var config Config
	if configFlag != "" {
		configData, err := ioutil.ReadFile(configFlag)
		if err != nil {
			return config, fmt.Errorf("error reading config file: %v", err)
		}
		err = yaml.Unmarshal(configData, &config)
		if err != nil {
			return config, fmt.Errorf("error parsing config file: %v", err)
		}
		if config.Output == "" {
			return config, fmt.Errorf("output path is required in the config file")
		}
	} else {
		if outputFlag == "" {
			return config, fmt.Errorf("output path is required; use -o or --output to specify")
		}
		config = Config{
			Directory:  directoryFlag,
			TokenLimit: tokenLimitFlag,
			Output:     outputFlag,
			Omit:       skipFlags,
			Preprompt:  prepromptFlag,
			Request:    requestFlag,
		}
		if config.Directory == "" {
			config.Directory = "."
		}
		if config.TokenLimit == 0 {
			config.TokenLimit = DefaultTokenLimit
		}
		if config.Preprompt == "" {
			config.Preprompt = DefaultPreprompt
		}
	}
	if config.Request != "" {
		if strings.Contains(config.Preprompt, "<request>") {
			config.Preprompt = strings.Replace(config.Preprompt, "<request>", config.Request, 1)
		} else {
			config.Preprompt += "\n\nRequest:\n\n" + config.Request
		}
	}
	return config, nil
}

// Config holds the configuration settings
type Config struct {
	Directory  string   `yaml:"directory"`
	TokenLimit int      `yaml:"token_limit"`
	Output     string   `yaml:"output"`
	Omit       []string `yaml:"omit"`
	Preprompt  string   `yaml:"preprompt"`
	Request    string   `yaml:"request"`
}

// countingWriter wraps an io.Writer and counts bytes written
type countingWriter struct {
	writer io.Writer
	count  int
}

func (cw *countingWriter) Write(p []byte) (n int, err error) {
	n, err = cw.writer.Write(p)
	cw.count += n
	return n, err
}

// LoadGitignore loads ignore patterns from .gitignore
func LoadGitignore(gitignorePath string) ([]string, error) {
	var patterns []string
	file, err := os.OpenFile(gitignorePath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return patterns, nil
		}
		return nil, fmt.Errorf("error opening .gitignore: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .gitignore: %v", err)
	}
	return patterns, nil
}

// IsIgnored checks if a file or directory should be ignored
func IsIgnored(path string, isDir bool, ignorePatterns []string) bool {
	// Normalize the input path
	normalizedPath := filepath.Clean(path)
	if isDir {
		normalizedPath += string(filepath.Separator)
	}

	for _, pattern := range ignorePatterns {
		// Replace '/' with the platform's separator in the pattern
		normalizedPattern := strings.ReplaceAll(pattern, "/", string(filepath.Separator))
		if matched, _ := filepath.Match(normalizedPattern, normalizedPath); matched {
			return true
		}
		if matched, _ := filepath.Match(normalizedPattern+"*", normalizedPath); matched {
			return true
		}
	}
	return false
}

// GenerateTree generates a directory tree structure
func GenerateTree(currentDir string, ignorePatterns []string, prefix string, rootDir string) []string {
	if filepath.Base(currentDir) == ".git" {
		return nil
	}
	var lines []string
	if prefix == "" {
		lines = append(lines, filepath.Base(currentDir))
	}
	files, err := ioutil.ReadDir(currentDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory %s: %v\n", currentDir, err)
		return lines
	}
	var contents []string
	for _, file := range files {
		fullPath := filepath.Join(currentDir, file.Name())
		relPath, _ := filepath.Rel(rootDir, fullPath)
		if file.IsDir() {
			if !IsIgnored(relPath, true, ignorePatterns) {
				contents = append(contents, file.Name())
			}
		} else {
			if !IsIgnored(relPath, false, ignorePatterns) {
				contents = append(contents, file.Name())
			}
		}
	}
	dirs := []string{}
	filesList := []string{}
	for _, content := range contents {
		if stat, err := os.Stat(filepath.Join(currentDir, content)); err == nil {
			if stat.IsDir() {
				dirs = append(dirs, content)
			} else {
				filesList = append(filesList, content)
			}
		}
	}
	sort.Strings(dirs)
	sort.Strings(filesList)
	for i, d := range dirs {
		isLast := (i == len(dirs)-1 && len(filesList) == 0)
		pointer := "├── "
		if isLast {
			pointer = "└── "
		}
		extension := "│   "
		if isLast {
			extension = "    "
		}
		lines = append(lines, prefix+pointer+d)
		subLines := GenerateTree(filepath.Join(currentDir, d), ignorePatterns, prefix+extension, rootDir)
		lines = append(lines, subLines...)
	}
	for i, f := range filesList {
		pointer := "├── "
		if i == len(filesList)-1 {
			pointer = "└── "
		}
		lines = append(lines, prefix+pointer+f)
	}
	return lines
}

// IsBinaryFile checks if a file is binary
func IsBinaryFile(filepath string) bool {
	file, err := os.OpenFile(filepath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "File does not exist: %s\n", filepath)
		} else if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "Permission denied: %s\n", filepath)
		} else {
			fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", filepath, err)
		}
		return false
	}
	defer file.Close()
	chunk := make([]byte, 1024)
	n, err := file.Read(chunk)
	if err != nil {
		return false
	}
	chunk = chunk[:n]
	if bytes.Contains(chunk, []byte{0}) {
		return true
	}
	for _, b := range chunk {
		if b < 32 && b != 7 && b != 8 && b != 9 && b != 10 && b != 12 && b != 13 && b != 27 {
			return true
		}
	}
	return false
}

// ProcessDirectory processes the directory and writes output to writer, returning total characters written
func ProcessDirectory(config Config, writer io.Writer) (int, error) {
	cw := &countingWriter{writer: writer}

	// Load ignore patterns
	gitignorePath := filepath.Join(config.Directory, ".gitignore")
	ignorePatterns, err := LoadGitignore(gitignorePath)
	if err != nil {
		return 0, fmt.Errorf("error loading .gitignore: %v", err)
	}
	ignorePatterns = append(ignorePatterns, config.Omit...)

	// Collect all files
	var allFiles []string
	err = filepath.Walk(config.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(config.Directory, path)
		if info.IsDir() {
			if IsIgnored(relPath, true, ignorePatterns) {
				return filepath.SkipDir
			}
		} else {
			if !IsIgnored(relPath, false, ignorePatterns) {
				allFiles = append(allFiles, relPath)
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error walking directory: %v", err)
	}

	// Write UTF-8 BOM
	_, err = cw.Write([]byte{0xEF, 0xBB, 0xBF})
	if err != nil {
		return 0, fmt.Errorf("error writing BOM: %v", err)
	}

	// Write preprompt
	_, err = cw.Write([]byte(config.Preprompt))
	if err != nil {
		return 0, fmt.Errorf("error writing preprompt: %v", err)
	}

	// Write directory structure
	_, err = cw.Write([]byte("Directory structure:\n"))
	if err != nil {
		return 0, fmt.Errorf("error writing directory header: %v", err)
	}
	treeLines := GenerateTree(config.Directory, ignorePatterns, "", config.Directory)
	treeStr := strings.Join(treeLines, "\n") + "\n\n"
	_, err = cw.Write([]byte(treeStr))
	if err != nil {
		return 0, fmt.Errorf("error writing directory tree: %v", err)
	}

	// Write file contents header
	_, err = cw.Write([]byte("File contents:\n\n"))
	if err != nil {
		return 0, fmt.Errorf("error writing contents header: %v", err)
	}

	// Process files with progress bar
	bar := pb.New(len(allFiles))
	bar.SetWriter(os.Stderr)
	bar.Set("desc", "Combining files")
	bar.Start()
	defer bar.Finish()

	for _, relPath := range allFiles {
		fullPath := filepath.Join(config.Directory, relPath)
		if IsBinaryFile(fullPath) {
			fmt.Fprintf(os.Stderr, "Skipping binary file: %s\n", relPath)
			bar.Increment()
			continue
		}
		content, err := ioutil.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "File not found: %s\n", relPath)
			} else if os.IsPermission(err) {
				fmt.Fprintf(os.Stderr, "Permission denied: %s\n", relPath)
			} else {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", relPath, err)
			}
			bar.Increment()
			continue
		}
		header := fmt.Sprintf("=== File: %s ===\n", relPath)
		_, err = cw.Write([]byte(header))
		if err != nil {
			return 0, fmt.Errorf("error writing file header for %s: %v", relPath, err)
		}
		_, err = cw.Write(content)
		if err != nil {
			return 0, fmt.Errorf("error writing file content for %s: %v", relPath, err)
		}
		_, err = cw.Write([]byte("\n\n"))
		if err != nil {
			return 0, fmt.Errorf("error writing file footer for %s: %v", relPath, err)
		}
		bar.Increment()
	}

	return cw.count, nil
}
