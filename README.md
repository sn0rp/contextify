# Contextify

Contextify is a POSIX-compliant command-line tool written in Go that merges the contents of a directory (typically a code repository) into a single text file. This file is intended for use with large language models (LLMs), providing them with context about your codebase. The tool processes a specified directory, respects `.gitignore` and user-defined ignore patterns, generates a directory tree, includes text file contents (skipping binaries), and prepends a customizable preprompt and request. It also enforces a token limit and optionally copies the output to the clipboard.

## Installation

You can install Contextify either by downloading pre-built binaries or building from source. Follow these steps:

### Option 1: Pre-built Binaries

1. **Visit the Releases Page**
   - Go to [GitHub Releases](https://github.com/sn0rp/contextify/releases).
   - Verify: Ensure the page loads and lists available releases.

2. **Download the Binary**
   - Select the latest release and download the binary for your OS (e.g., `contextify_linux_amd64.tar.gz` for Linux).
   - Verify: Check that the file is downloaded to your system.

3. **Extract the Archive**
   - Run: `tar -xzf contextify_linux_amd64.tar.gz` (Linux/macOS) or unzip for Windows.
   - Verify: Confirm the `contextify` binary is present in the extracted directory.

4. **Make Executable (Linux/macOS)**
   - Run: `chmod +x contextify`
   - Verify: Run `ls -l contextify` and ensure it shows executable permissions (e.g., `-rwxr-xr-x`).

5. **Move to PATH (Optional)**
   - Run: `sudo mv contextify /usr/local/bin/`
   - Verify: Run `contextify --help` from any terminal and see the help message.

### Option 2: Building from Source

1. **Install Go**
   - Download and install Go 1.23.2+ from [golang.org](https://golang.org/dl/).
   - Verify: Run `go version` and confirm the output (e.g., `go1.23.2`).

2. **Clone the Repository**
   - Run: `git clone https://github.com/sn0rp/contextify.git && cd contextify`
   - Verify: Check that the directory contains `main.go`, `go.mod`, etc.

3. **Build the Binary**
   - Run: `go build -o contextify main.go`
   - Verify: Confirm `contextify` binary exists in the current directory.

4. **Move to PATH (Optional)**
   - Run: `sudo mv contextify /usr/local/bin/`
   - Verify: Run `contextify --help` from any terminal.

## Usage

Contextify operates via command-line flags or a YAML config file. Below are all available flags:

- `--config`, `-c` <path>: Path to a YAML config file (exclusive with other flags except `-g`).
- `--directory`, `-d` <path>: Directory to process (defaults to `.` if unspecified).
- `--tokens`, `-t` <int>: Token limit (defaults to 128,000).
- `--output`, `-o` <path>: Output file path (required unless using `--config`).
- `--skip`, `-s` <pattern>: Files/directories to omit (can be used multiple times).
- `--preprompt`, `-p` <message>: Message to prepend to the output.
- `--generate-config`, `-g` <path>: Generate a default config file at the specified path.
- `--request`, `-r` <request>: Request to include in the preprompt.

### Step-by-Step Instructions

#### Basic Usage
   - Run: `contextify -o output.txt`
   - This processes the current directory and writes to `output.txt`.

#### Using a Config File
1. **Generate Default Config**
   - Run: `contextify -g config.yaml`
   - Verify: Check that `config.yaml` is created in the current directory.

2. **Edit Config**
   - Open `config.yaml` in an editor and modify as needed (see [Configuration](#configuration)).
   - Verify: Save the file and confirm changes (e.g., `cat config.yaml`).

3. **Run with Config**
   - Run: `contextify -c config.yaml`
   - Verify: Check the output file specified in `config.yaml`.

## Configuration

You can use a YAML file to configure Contextify. Here’s an example:

```yaml
directory: "."
token_limit: 128000
output: "/tmp/contextify/my_project_codebase.txt"
omit:
  - ".git/"
  - "node_modules/"
preprompt: "Analyze this codebase:\n"
request: "Find all TODO comments."
```

### Fields
- `directory`: Directory to process.
- `token_limit`: Maximum tokens allowed.
- `output`: Output file path.
- `omit`: List of files/directories to skip.
- `preprompt`: Message prepended to the output (replaces `<request>` with `request` if present).
- `request`: Specific request to include in the preprompt.

### Steps to Use
1. **Generate Config**
   - Run: `contextify -g config.yaml`
   - Verify: File exists with default values.

2. **Customize Config**
   - Edit `config.yaml` with your settings.
   - Verify: Changes are saved.

3. **Execute**
   - Run: `contextify -c config.yaml`
   - Verify: Output matches config settings.

## Contributing
Yeah

## License

Contextify is released under the [GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html). See [LICENSE.txt](LICENSE.txt) for details.