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

// FrontmatterInfo zawiera informacje o pozycji frontmatter w pliku
type FrontmatterInfo struct {
	Content  string
	StartPos int64
	EndPos   int64
	HasFM    bool
}

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

	// Parse global flags like --dry-run
	processedArgs := []string{}
	for _, arg := range args {
		switch arg {
		case "--dry-run":
			dryRun = true
		default:
			processedArgs = append(processedArgs, arg)
		}
	}
	args = processedArgs

	switch command {
	case "get":
		return handleGet(args)
	case "set":
		return handleSet(args, dryRun)
	case "delete":
		return handleDelete(args, dryRun)
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func printUsage() {
	fmt.Println("Usage: frontmatter [get|set|delete] [--dry-run] [...] <file>")
	fmt.Println("Examples:")
	fmt.Println("  frontmatter set message=\"Hello World\" file.md")
	fmt.Println("  frontmatter set object.field=5 file.md")
	fmt.Println("  frontmatter set a=1 b=value file.md")
	fmt.Println("  frontmatter get message file.md")
	fmt.Println("  frontmatter get file.md")
	fmt.Println("  frontmatter delete file.md")
	fmt.Println("  frontmatter delete title file.md")
	fmt.Println("  frontmatter delete first second file.md")
	fmt.Println("  frontmatter delete object.field file.md")
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

func parseFrontmatter(fmString string) (map[string]any, error) {
	data := make(map[string]any)
	if strings.TrimSpace(fmString) == "" {
		return data, nil // Empty frontmatter is valid
	}
	err := yaml.Unmarshal([]byte(fmString), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}
	return data, nil
}

func serializeFrontmatter(data map[string]any) (string, error) {
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

func handleGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("no file specified for get")
	}

	filePath := args[len(args)-1]
	keys := args[:len(args)-1]

	// Używamy zoptymalizowanego odczytu
	info, err := readFrontmatterInfo(filePath)
	if err != nil {
		return err
	}

	if !info.HasFM || strings.TrimSpace(info.Content) == "" {
		// No frontmatter found or it's empty - return error code 2 (not found)
		return &ExitError{Code: 2, Message: "frontmatter not found"}
	}

	data, err := parseFrontmatter(info.Content)
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
	case map[string]any, []any, map[any]any:
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

	// Używamy zoptymalizowanego odczytu
	info, err := readFrontmatterInfo(filePath)
	if err != nil {
		return err
	}

	data, err := parseFrontmatter(info.Content)
	if err != nil {
		// If frontmatter is malformed, we might want to overwrite or error out.
		// For now, let's try to proceed with an empty map if parsing fails, effectively overwriting.
		// A stricter approach would be: return fmt.Errorf("failed to parse existing frontmatter: %w", err)
		fmt.Fprintf(os.Stderr, "Warning: could not parse existing frontmatter, new values will overwrite or be added to a new frontmatter block: %v\n", err)
		data = make(map[string]any)
	}

	for _, kvPair := range setArgs {
		parts := strings.SplitN(kvPair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key=value format: %s", kvPair)
		}
		keyPath := parts[0]
		valueStr := parts[1]

		var parsedValue any
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
			var yamlValue any
			if err := yaml.Unmarshal([]byte(valueStr), &yamlValue); err == nil {
				parsedValue = yamlValue
			} else {
				// If YAML parsing fails, treat as string
				parsedValue = strings.Trim(valueStr, "\"") // Trim quotes if it was a quoted string
			}
		} else if strings.HasPrefix(valueStr, "{") && strings.HasSuffix(valueStr, "}") {
			// Attempt to parse JSON-like map first
			var jsonValue map[string]any
			if err := json.Unmarshal([]byte(valueStr), &jsonValue); err == nil {
				parsedValue = jsonValue
			} else {
				// Fallback to YAML
				var yamlValue any
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

	return writeOptimizedFrontmatter(filePath, newFmString, info, dryRun)
}

func handleDelete(args []string, dryRun bool) error {
	if len(args) < 1 {
		return fmt.Errorf("file path must be specified for delete")
	}

	filePath := args[len(args)-1]
	fieldsToDelete := args[:len(args)-1]

	// Dla delete używamy bezpieczniejszej metody - całego odczytu pliku
	fmString, bodyString, err := readFileContent(filePath)
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

	if strings.TrimSpace(fmString) == "" {
		// No frontmatter to delete
		if dryRun {
			fmt.Print(bodyString)
		} else {
			return writeFileContent(filePath, "", bodyString, false)
		}
		return nil
	}

	// If no fields specified, delete entire frontmatter
	if len(fieldsToDelete) == 0 {
		return writeFileContent(filePath, "", bodyString, dryRun)
	}

	// Parse existing frontmatter
	data, err := parseFrontmatter(fmString)
	if err != nil {
		return fmt.Errorf("failed to parse existing frontmatter: %w", err)
	}

	// Delete specified fields
	for _, fieldPath := range fieldsToDelete {
		deleteValueByPath(data, fieldPath)
	}

	// Serialize updated frontmatter
	newFmString, err := serializeFrontmatter(data)
	if err != nil {
		return err
	}

	return writeFileContent(filePath, newFmString, bodyString, dryRun)
}

// readFrontmatterInfo reads only the frontmatter section and returns position info
func readFrontmatterInfo(filePath string) (*FrontmatterInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &FrontmatterInfo{Content: "", StartPos: 0, EndPos: 0, HasFM: false}, nil
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var frontmatterContent strings.Builder
	var bytesRead int64
	separatorCount := 0

	for {
		line, err := reader.ReadString('\n')
		bytesRead += int64(len(line))

		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == frontmatterSeparator && separatorCount < 2 {
			separatorCount++
			if separatorCount == 2 {
				// Znaleźliśmy koniec frontmatter
				return &FrontmatterInfo{
					Content:  frontmatterContent.String(),
					StartPos: 0,
					EndPos:   bytesRead,
					HasFM:    true,
				}, nil
			}
			if err == io.EOF {
				break
			}
			continue
		}

		if separatorCount == 1 {
			frontmatterContent.WriteString(line)
		} else if separatorCount == 0 {
			// Nie ma frontmatter na początku
			if err == io.EOF || bytesRead > 1024 { // Sprawdź tylko pierwsze 1KB
				return &FrontmatterInfo{Content: "", StartPos: 0, EndPos: 0, HasFM: false}, nil
			}
		}

		if err == io.EOF {
			break
		}
	}

	// Niepełny frontmatter lub brak
	return &FrontmatterInfo{Content: "", StartPos: 0, EndPos: 0, HasFM: false}, nil
}

// readBodyFromPosition reads file content from a specific position to the end
func readBodyFromPosition(filePath string, startPos int64) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Przejdź do pozycji po frontmatter
	if _, err := file.Seek(startPos, 0); err != nil {
		return "", fmt.Errorf("failed to seek to position %d: %w", startPos, err)
	}

	// Przeczytaj resztę pliku
	bodyBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read body content: %w", err)
	}

	return string(bodyBytes), nil
}

// writeOptimizedFrontmatter writes frontmatter using optimized strategy
func writeOptimizedFrontmatter(filePath, newFmString string, info *FrontmatterInfo, dryRun bool) error {
	if dryRun {
		return writeFileContentForDryRun(filePath, newFmString, info)
	}

	// Dla bezpieczeństwa, zawsze używamy przepisania całego pliku
	// In-place editing jest ryzykowne i może uszkodzić dane
	return writeFileContentSafe(filePath, newFmString, info)
}

// writeFileContentForDryRun handles dry-run output efficiently
func writeFileContentForDryRun(filePath, newFmString string, info *FrontmatterInfo) error {
	var finalContent strings.Builder
	hasFrontmatter := strings.TrimSpace(newFmString) != ""

	if hasFrontmatter {
		finalContent.WriteString(frontmatterSeparator)
		finalContent.WriteString("\n")
		finalContent.WriteString(newFmString)
		if !strings.HasSuffix(newFmString, "\n") && len(newFmString) > 0 {
			finalContent.WriteString("\n")
		}
		finalContent.WriteString(frontmatterSeparator)
		finalContent.WriteString("\n")
	}

	// Dodaj body content jeśli istnieje
	if info.HasFM && info.EndPos > 0 {
		bodyContent, err := readBodyFromPosition(filePath, info.EndPos)
		if err != nil {
			return err
		}
		finalContent.WriteString(bodyContent)
	} else if !info.HasFM {
		// Cały plik to body
		content, err := os.ReadFile(filePath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil {
			finalContent.WriteString(string(content))
		}
	}

	fmt.Print(finalContent.String())
	return nil
}

// writeFileContentSafe safely rewrites the entire file (fallback method)
func writeFileContentSafe(filePath, newFmString string, info *FrontmatterInfo) error {
	var finalContent strings.Builder
	hasFrontmatter := strings.TrimSpace(newFmString) != ""

	if hasFrontmatter {
		finalContent.WriteString(frontmatterSeparator)
		finalContent.WriteString("\n")
		finalContent.WriteString(newFmString)
		if !strings.HasSuffix(newFmString, "\n") && len(newFmString) > 0 {
			finalContent.WriteString("\n")
		}
		finalContent.WriteString(frontmatterSeparator)
		finalContent.WriteString("\n")
	}

	// Dodaj body content jeśli istnieje
	if info.HasFM && info.EndPos > 0 {
		bodyContent, err := readBodyFromPosition(filePath, info.EndPos)
		if err != nil {
			return err
		}
		finalContent.WriteString(bodyContent)
	} else if !info.HasFM {
		// Cały plik to body - przeczytaj go w całości
		content, err := os.ReadFile(filePath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil {
			finalContent.WriteString(string(content))
		}
	}

	// Bezpieczny zapis: użyj pliku tymczasowego
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, []byte(finalContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomowe przeniesienie
	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile) // Oczyść w przypadku błędu
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// setValueByPath sets a value in a nested map structure based on a dot-separated path.
func setValueByPath(data map[string]any, path string, value any) error {
	parts := strings.Split(path, ".")
	currentMap := data

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, set the value
			currentMap[part] = value
		} else {
			// Navigate or create nested map
			if _, ok := currentMap[part]; !ok {
				currentMap[part] = make(map[string]any)
			}
			nestedMap, ok := currentMap[part].(map[string]any)
			if !ok {
				// Path conflict: part exists but is not a map.
				// Overwrite with a new map to continue, or return an error.
				// For simplicity, let's overwrite.
				// return fmt.Errorf("path conflict: '%s' in '%s' is not a map", part, path)
				newMap := make(map[string]any)
				currentMap[part] = newMap
				nestedMap = newMap
			}
			currentMap = nestedMap
		}
	}
	return nil
}

// getValueByPath retrieves a value from a nested map structure based on a dot-separated path.
func getValueByPath(data map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var currentValue any = data

	for _, part := range parts {
		currentMap, ok := currentValue.(map[string]any)
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

// deleteValueByPath removes a value from a nested map structure based on a dot-separated path.
func deleteValueByPath(data map[string]any, path string) bool {
	parts := strings.Split(path, ".")

	// If there's only one part, delete directly from the root map
	if len(parts) == 1 {
		_, existed := data[parts[0]]
		delete(data, parts[0])
		return existed
	}

	// Navigate to the parent of the field to delete
	var currentValue any = data
	for _, part := range parts[:len(parts)-1] {
		currentMap, ok := currentValue.(map[string]any)
		if !ok {
			// Path doesn't exist, nothing to delete
			return false
		}
		value, found := currentMap[part]
		if !found {
			// Path doesn't exist, nothing to delete
			return false
		}
		currentValue = value
	}

	// Delete the final key
	if finalMap, ok := currentValue.(map[string]any); ok {
		finalKey := parts[len(parts)-1]
		_, existed := finalMap[finalKey]
		delete(finalMap, finalKey)
		return existed
	}

	return false
}
