package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const frontmatterSeparator = "---"

// ExitError represents an error with a specific exit code
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		if exitErr, ok := err.(*ExitError); ok {
			// Don't print error for "not found" cases (code 2)
			if exitErr.Code != 2 {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			os.Exit(exitErr.Code)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		printUsage()
		return fmt.Errorf("not enough arguments")
	}

	command := args[0]
	args = args[1:]

	dryRun := false
	deleteFrontmatter := false

	// Parse global flags like --dry-run or --delete
	processedArgs := []string{}
	for _, arg := range args {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--delete": // This is a custom flag for the delete functionality
			deleteFrontmatter = true
		default:
			processedArgs = append(processedArgs, arg)
		}
	}
	args = processedArgs

	switch command {
	case "get":
		return handleGet(args, dryRun)
	case "set":
		if deleteFrontmatter {
			return handleDelete(args, dryRun)
		}
		return handleSet(args, dryRun)
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func printUsage() {
	fmt.Println("Usage: frontmatter [get|set] [--dry-run] [--delete] [key=value...] <file>")
	fmt.Println("Examples:")
	fmt.Println("  frontmatter set message=\"Hello World\" file.md")
	fmt.Println("  frontmatter set object.field=5 file.md")
	fmt.Println("  frontmatter set a=1 b=value file.md")
	fmt.Println("  frontmatter get message file.md")
	fmt.Println("  frontmatter get file.md")
	fmt.Println("  frontmatter set --delete file.md")
}

func readFileContent(filePath string) (string, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, treat as empty frontmatter and no body
			return "", "", nil
		}
		return "", "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var frontmatterContent, bodyContent strings.Builder
	inFrontmatter := false
	separatorCount := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", "", fmt.Errorf("failed to read file: %w", err)
		}

		trimmed := strings.TrimSpace(line)
		// Treat only first two separators as frontmatter delimiters
		if trimmed == frontmatterSeparator && separatorCount < 2 {
			separatorCount++
			if separatorCount == 1 {
				inFrontmatter = true
			} else if separatorCount == 2 {
				inFrontmatter = false
			}
			if err == io.EOF {
				break
			}
			continue
		}

		if inFrontmatter && separatorCount == 1 {
			frontmatterContent.WriteString(line)
		} else {
			bodyContent.WriteString(line)
		}

		if err == io.EOF {
			break
		}
	}

	// If only one separator or no separators, it's not valid frontmatter block
	if separatorCount < 2 {
		// The entire content is body if no frontmatter was properly defined
		return "", frontmatterContent.String() + bodyContent.String(), nil
	}

	return frontmatterContent.String(), bodyContent.String(), nil
}

func parseFrontmatter(fmString string) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	if strings.TrimSpace(fmString) == "" {
		return data, nil // Empty frontmatter is valid
	}
	err := yaml.Unmarshal([]byte(fmString), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}
	return data, nil
}

func serializeFrontmatter(data map[string]interface{}) (string, error) {
	if len(data) == 0 {
		return "", nil // No data, no frontmatter string
	}
	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2) // Common YAML indent
	err := yamlEncoder.Encode(data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize YAML: %w", err)
	}
	raw := b.String()
	// Remove unnecessary quotes around simple keys
	re := regexp.MustCompile(`(?m)^(\s*)"([A-Za-z0-9_-]+)":`)
	cleaned := re.ReplaceAllString(raw, `$1$2:`)
	return cleaned, nil
}

func writeFileContent(filePath, fmString, bodyString string, dryRun bool) error {
	var finalContent strings.Builder
	hasFrontmatter := strings.TrimSpace(fmString) != ""

	if hasFrontmatter {
		finalContent.WriteString(frontmatterSeparator)
		finalContent.WriteString("\n")
		finalContent.WriteString(fmString)
		// Ensure frontmatter ends with a newline if it's not empty and doesn't have one
		if !strings.HasSuffix(fmString, "\n") && len(fmString) > 0 {
			finalContent.WriteString("\n")
		}
		finalContent.WriteString(frontmatterSeparator)
		finalContent.WriteString("\n")
	}

	finalContent.WriteString(bodyString)

	if dryRun {
		fmt.Print(finalContent.String())
		return nil
	}

	return os.WriteFile(filePath, []byte(finalContent.String()), 0644)
}

func handleGet(args []string, dryRun bool) error {
	if len(args) < 1 {
		return fmt.Errorf("no file specified for get")
	}

	filePath := args[len(args)-1]
	keys := args[:len(args)-1]

	fmString, _, err := readFileContent(filePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(fmString) == "" {
		// No frontmatter found or it's empty - return error code 2 (not found)
		return &ExitError{Code: 2, Message: "frontmatter not found"}
	}

	data, err := parseFrontmatter(fmString)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		// Get all frontmatter
		yamlBytes, err := yaml.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal data for get all: %w", err)
		}
		fmt.Print(string(yamlBytes))
		return nil
	}

	// Get specific key(s)
	// For simplicity, this implementation will handle one key. Multiple keys could return a map.
	key := keys[0]
	value, found := getValueByPath(data, key)
	if !found {
		// Key not found - return error code 2 (not found)
		return &ExitError{Code: 2, Message: "field not found"}
	}

	// If value is a map or slice, YAML marshal it. Otherwise, print directly.
	switch v := value.(type) {
	case map[string]interface{}, []interface{}, map[interface{}]interface{}:
		yamlBytes, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key '%s': %w", key, err)
		}
		fmt.Print(string(yamlBytes))
	default:
		fmt.Println(v)
	}

	return nil
}

func handleSet(args []string, dryRun bool) error {
	if len(args) < 2 {
		return fmt.Errorf("at least one key=value pair and a file must be specified for set")
	}

	filePath := args[len(args)-1]
	setArgs := args[:len(args)-1]

	fmString, bodyString, err := readFileContent(filePath)
	if err != nil {
		// If file doesn't exist, readFileContent returns empty strings, which is fine.
		// We only care about actual read errors here.
		if !os.IsNotExist(err) { // Check if it's a genuine read error, not just "file not found"
			return err
		}
		// If it does not exist, fmString and bodyString will be empty, which is handled.
	}

	data, err := parseFrontmatter(fmString)
	if err != nil {
		// If frontmatter is malformed, we might want to overwrite or error out.
		// For now, let's try to proceed with an empty map if parsing fails, effectively overwriting.
		// A stricter approach would be: return fmt.Errorf("failed to parse existing frontmatter: %w", err)
		fmt.Fprintf(os.Stderr, "Warning: could not parse existing frontmatter, new values will overwrite or be added to a new frontmatter block: %v\n", err)
		data = make(map[string]interface{})
	}

	for _, kvPair := range setArgs {
		parts := strings.SplitN(kvPair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key=value format: %s", kvPair)
		}
		keyPath := parts[0]
		valueStr := parts[1]

		var parsedValue interface{}
		// Try to parse value as YAML/JSON scalar types
		if valInt, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			parsedValue = valInt
		} else if valFloat, err := strconv.ParseFloat(valueStr, 64); err == nil {
			parsedValue = valFloat
		} else if valBool, err := strconv.ParseBool(valueStr); err == nil {
			parsedValue = valBool
		} else if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") ||
			strings.HasPrefix(valueStr, "{") && strings.HasSuffix(valueStr, "}") {
			// Attempt to parse as YAML if it looks like a list or map
			var yamlValue interface{}
			if err := yaml.Unmarshal([]byte(valueStr), &yamlValue); err == nil {
				parsedValue = yamlValue
			} else {
				// If YAML parsing fails, treat as string
				parsedValue = strings.Trim(valueStr, "\"") // Trim quotes if it was a quoted string
			}
		} else if strings.HasPrefix(valueStr, "{") && strings.HasSuffix(valueStr, "}") {
			// Attempt to parse JSON-like map first
			var jsonValue map[string]interface{}
			if err := json.Unmarshal([]byte(valueStr), &jsonValue); err == nil {
				parsedValue = jsonValue
			} else {
				// Fallback to YAML
				var yamlValue interface{}
				if err2 := yaml.Unmarshal([]byte(valueStr), &yamlValue); err2 == nil {
					parsedValue = yamlValue
				} else {
					parsedValue = strings.Trim(valueStr, "\"")
				}
			}
		} else {
			parsedValue = strings.Trim(valueStr, "\"") // Default to string, trim quotes
		}

		if err := setValueByPath(data, keyPath, parsedValue); err != nil {
			return fmt.Errorf("failed to set value for key '%s': %w", keyPath, err)
		}
	}

	newFmString, err := serializeFrontmatter(data)
	if err != nil {
		return err
	}

	return writeFileContent(filePath, newFmString, bodyString, dryRun)
}

func handleDelete(args []string, dryRun bool) error {
	if len(args) != 1 {
		return fmt.Errorf("file path must be specified for delete")
	}
	filePath := args[0]

	// Read the file to get the body content, ignore frontmatter
	_, bodyString, err := readFileContent(filePath)
	if err != nil {
		// If file doesn't exist, nothing to delete.
		if os.IsNotExist(err) {
			if dryRun {
				fmt.Print("") // Dry run on non-existent file shows empty output
			}
			return nil
		}
		return err
	}

	// For delete, the new frontmatter string is empty.
	return writeFileContent(filePath, "", bodyString, dryRun)
}

// setValueByPath sets a value in a nested map structure based on a dot-separated path.
func setValueByPath(data map[string]interface{}, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	currentMap := data

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, set the value
			currentMap[part] = value
		} else {
			// Navigate or create nested map
			if _, ok := currentMap[part]; !ok {
				currentMap[part] = make(map[string]interface{})
			}
			nestedMap, ok := currentMap[part].(map[string]interface{})
			if !ok {
				// Path conflict: part exists but is not a map.
				// Overwrite with a new map to continue, or return an error.
				// For simplicity, let's overwrite.
				// return fmt.Errorf("path conflict: '%s' in '%s' is not a map", part, path)
				newMap := make(map[string]interface{})
				currentMap[part] = newMap
				nestedMap = newMap
			}
			currentMap = nestedMap
		}
	}
	return nil
}

// getValueByPath retrieves a value from a nested map structure based on a dot-separated path.
func getValueByPath(data map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var currentValue interface{} = data

	for _, part := range parts {
		currentMap, ok := currentValue.(map[string]interface{})
		if !ok {
			// If at any point the path does not lead to a map, the key is not found as specified.
			return nil, false
		}
		value, found := currentMap[part]
		if !found {
			return nil, false
		}
		currentValue = value
	}
	return currentValue, true
}
