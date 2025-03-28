package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	contextify "contextify/pkg"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func main() {
	// Define command-line flags
	var configFlag, directoryFlag, outputFlag, prepromptFlag, generateConfigFlag string
	var tokenLimitFlag int
	var skipFlags []string
	var requestFlag string

	flag.StringVarP(&configFlag, "config", "c", "", "Path to config YAML file.")
	flag.StringVarP(&directoryFlag, "directory", "d", "", "Directory to process.")
	flag.IntVarP(&tokenLimitFlag, "tokens", "t", 0, "Context/token limit.")
	flag.StringVarP(&outputFlag, "output", "o", "", "Output file path (relative or absolute).")
	flag.StringSliceVarP(&skipFlags, "skip", "s", []string{}, "Files or directories to omit.")
	flag.StringVarP(&prepromptFlag, "preprompt", "p", "", "Preprompt message to prepend to the output.")
	flag.StringVarP(&generateConfigFlag, "generate-config", "g", "", "Generate a default config file at the specified path.")
	flag.StringVarP(&requestFlag, "request", "r", "", "Request to include in the preprompt.")
	flag.Parse()

	// Show help if no arguments provided
	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Handle config generation
	if generateConfigFlag != "" {
		directory := directoryFlag
		if directory == "" {
			directory = "."
		}

		// Resolve the absolute path of the directory
		absDir, err := filepath.Abs(directory)
		if err != nil {
			fmt.Printf("Error getting absolute path for directory: %v\n", err)
			os.Exit(1)
		}

		// Get the base name of the absolute path
		basename := filepath.Base(absDir)

		// If directory is ".", explicitly get the current directory's name
		if directory == "." {
			currentDir, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error getting current working directory: %v\n", err)
				os.Exit(1)
			}
			basename = filepath.Base(currentDir)
		}

		// Handle edge case where basename might still be empty or a separator
		if basename == "" || basename == "." || basename == string(filepath.Separator) {
			basename = "current_dir" // Fallback name
		}

		// Construct the default output path
		defaultOutput := filepath.Join(os.TempDir(), "contextify", fmt.Sprintf("%s_codebase.txt", basename))

		// Build the config map
		configToWrite := map[string]interface{}{
			"directory":   directory,
			"token_limit": contextify.DefaultTokenLimit,
			"output":      defaultOutput,
			"omit":        []string{".git" + string(filepath.Separator)},
			"preprompt":   contextify.DefaultPreprompt,
		}
		if requestFlag != "" {
			configToWrite["request"] = requestFlag
		}

		// Marshal and write the config file
		configData, err := yaml.Marshal(configToWrite)
		if err != nil {
			fmt.Printf("Error generating config: %v\n", err)
			os.Exit(1)
		}
		err = os.MkdirAll(filepath.Dir(generateConfigFlag), 0755)
		if err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			os.Exit(1)
		}
		err = ioutil.WriteFile(generateConfigFlag, configData, 0644)
		if err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Default config file generated at %s\n", generateConfigFlag)
		os.Exit(0)
	}

	// Prevent mixing config file with other options
	if configFlag != "" && (directoryFlag != "" || tokenLimitFlag != 0 || outputFlag != "" || len(skipFlags) != 0 || prepromptFlag != "" || requestFlag != "") {
		fmt.Println("Cannot use --config with other options.")
		os.Exit(1)
	}

	// Load configuration
	config, err := contextify.LoadConfigFromFlags(configFlag, directoryFlag, outputFlag, prepromptFlag, requestFlag, tokenLimitFlag, skipFlags)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Add additional ignore patterns
	ignorePatterns := []string{}
	scriptPath, err := os.Executable()
	if err == nil {
		relScriptPath, err := filepath.Rel(config.Directory, scriptPath)
		if err == nil && !strings.HasPrefix(relScriptPath, "..") && !filepath.IsAbs(relScriptPath) {
			ignorePatterns = append(ignorePatterns, relScriptPath)
		}
	}
	if configFlag != "" {
		relConfigPath, err := filepath.Rel(config.Directory, configFlag)
		if err == nil && !strings.HasPrefix(relConfigPath, "..") && !filepath.IsAbs(relConfigPath) {
			ignorePatterns = append(ignorePatterns, relConfigPath)
		}
	}
	relOutputPath, err := filepath.Rel(config.Directory, config.Output)
	if err == nil && !strings.HasPrefix(relOutputPath, "..") && !filepath.IsAbs(relOutputPath) {
		ignorePatterns = append(ignorePatterns, relOutputPath)
	}
	config.Omit = append(config.Omit, ignorePatterns...)

	// Ensure output directory exists
	err = os.MkdirAll(filepath.Dir(config.Output), 0755)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Process directory and write output
	outfile, err := os.Create(config.Output)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outfile.Close()

	totalChars, err := contextify.ProcessDirectory(config, outfile)
	if err != nil {
		fmt.Printf("Error processing directory: %v\n", err)
		os.Exit(1)
	}

	// Verify size against context limit
	totalTokens := totalChars / contextify.CharPerToken
	differenceTokens := config.TokenLimit - totalTokens

	fmt.Printf("\nOutput written to %s\n", config.Output)
	fmt.Printf("Estimated size: %d characters (~%d tokens)\n", totalChars, totalTokens)
	fmt.Printf("Context limit: %d characters (~%d tokens)\n", config.TokenLimit*contextify.CharPerToken, config.TokenLimit)
	fmt.Printf("Difference: %d tokens (%s)\n", differenceTokens, map[bool]string{true: "Fits within limit", false: "Exceeds limit"}[differenceTokens >= 0])

	if differenceTokens < 0 {
		fmt.Printf("Warning: The combined file exceeds the context limit of %d tokens. You may need to split it or reduce the number of files.\n", config.TokenLimit)
	} else {
		content, err := ioutil.ReadFile(config.Output)
		if err != nil {
			fmt.Printf("Failed to read output file for clipboard: %v\n", err)
		} else {
			err = contextify.CopyToClipboard(string(content))
			if err != nil {
				fmt.Printf("Clipboard not supported: %v. Output is still available in %s.\n", err, config.Output)
			} else {
				fmt.Println("Output copied to clipboard.")
			}
		}
	}
}
