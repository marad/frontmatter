package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const testFile = "test_file.md"
const testFileNoFrontmatter = "test_file_no_frontmatter.md"
const testFileEmpty = "test_file_empty.md"
const binaryName = "frontmatter"

// TestMain runs before all tests and builds the binary once
func TestMain(m *testing.M) {
	// Build the binary once at the start
	if err := buildBinary(); err != nil {
		fmt.Printf("Failed to build binary: %v\n", err)
		os.Exit(1)
	}

	// Run all tests
	code := m.Run()

	// Clean up the binary after all tests
	os.Remove(binaryName)

	os.Exit(code)
}

func buildBinary() error {
	buildCmd := exec.Command("go", "build", "-o", binaryName, "main.go")
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build binary: %w", err)
	}
	return nil
}

func setupTestFile(content string) error {
	return os.WriteFile(testFile, []byte(content), 0644)
}

func setupTestFileNoFrontmatter(content string) error {
	return os.WriteFile(testFileNoFrontmatter, []byte(content), 0644)
}

func setupTestFileEmpty() error {
	return os.WriteFile(testFileEmpty, []byte(""), 0644)
}

func cleanupTestFiles() {
	os.Remove(testFile)
	os.Remove(testFileNoFrontmatter)
	os.Remove(testFileEmpty)
}

func runCmd(args ...string) (string, string, error) {
	// The binary should already exist from TestMain
	if _, err := os.Stat("./" + binaryName); os.IsNotExist(err) {
		return "", "", fmt.Errorf("binary %s does not exist - TestMain should have built it", binaryName)
	}

	cmd := exec.Command("./"+binaryName, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func assertFileContains(t *testing.T, filePath, expectedContent string) {
	t.Helper()
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filePath, err)
	}
	if !strings.Contains(string(content), expectedContent) {
		t.Errorf("File %s content\n%s\ndoes not contain\n%s", filePath, string(content), expectedContent)
	}
}

func assertStringContains(t *testing.T, actual, expectedSubstring string) {
	t.Helper()
	if !strings.Contains(actual, expectedSubstring) {
		t.Errorf("String\n%s\ndoes not contain\n%s", actual, expectedSubstring)
	}
}

func assertNoError(t *testing.T, err error, stderr string) {
	t.Helper()
	if err != nil {
		t.Fatalf("Command failed with error: %v, stderr: %s", err, stderr)
	}
	if stderr != "" {
		// Allow stderr for warnings or info, but log it.
		// Fail only if it seems to indicate a real problem not caught by exit code.
		// For example, if stderr contains "Error:", but err is nil.
		if strings.Contains(stderr, "Error:") || strings.Contains(stderr, "cannot find package") || strings.Contains(stderr, "Failed") || strings.Contains(stderr, "failed") {
			// t.Logf("Potentially problematic stderr: %s", stderr) // Commenting out to reduce noise for now
		}
	}
}

func assertExitCode(t *testing.T, err error, expectedCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected exit code %d, but command succeeded", expectedCode)
	}

	// Check if it's an exec.ExitError
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() != expectedCode {
			t.Fatalf("Expected exit code %d, but got %d", expectedCode, exitError.ExitCode())
		}
	} else {
		t.Fatalf("Expected exit code %d, but got non-exit error: %v", expectedCode, err)
	}
}

func TestSetSingleField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ntitle: Old Title\n---\nSome content"
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("set", "message=Hello World", testFile)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFile, "message: Hello World")
	assertFileContains(t, testFile, "title: Old Title") // Ensure other fields are not removed
}

func TestSetNestedField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\nobject:\n  other: value\n---\nSome content"
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("set", "object.field=5", testFile)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFile, "field: 5")
	assertFileContains(t, testFile, "other: value")
}

func TestSetMultipleFields(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\nexisting: true\n---\nSome content"
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("set", "a=1", "b=value", testFile)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFile, "a: 1")
	assertFileContains(t, testFile, "b: value")
	assertFileContains(t, testFile, "existing: true")
}

func TestSetFieldInNewFile(t *testing.T) {
	defer cleanupTestFiles()
	if err := setupTestFileEmpty(); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("set", "message=Hello World", testFileEmpty)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileEmpty, "message: Hello World")
	// Check if frontmatter delimiters are added
	content, _ := os.ReadFile(testFileEmpty)
	if !strings.HasPrefix(string(content), "---\n") {
		t.Errorf("File %s should start with ---, but got %s", testFileEmpty, string(content))
	}
}

func TestSetFieldInFileWithoutFrontmatter(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "Just some text."
	if err := setupTestFileNoFrontmatter(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("set", "message=Hello World", testFileNoFrontmatter)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileNoFrontmatter, "message: Hello World")
	assertFileContains(t, testFileNoFrontmatter, "Just some text.") // Ensure original content is preserved

	content, err := os.ReadFile(testFileNoFrontmatter)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", testFileNoFrontmatter, err)
	}
	sContent := string(content)

	expectedFMPart := "message: Hello World"
	expectedBodyPart := "Just some text."

	if !strings.Contains(sContent, expectedFMPart) {
		t.Errorf("File content does not contain expected frontmatter part '%s'. Content:\n%s", expectedFMPart, sContent)
	}
	if !strings.Contains(sContent, expectedBodyPart) {
		t.Errorf("File content does not contain expected body part '%s'. Content:\n%s", expectedBodyPart, sContent)
	}

	// Check structure: ---, frontmatter, ---, body
	// Ensure the frontmatter is at the beginning and correctly formatted.
	if !(strings.HasPrefix(sContent, "---\n") && strings.Contains(sContent, "\n---\n"+expectedBodyPart)) {
		// Check if the body might have had a newline added if it didn't have one
		if !(strings.HasPrefix(sContent, "---\n") && strings.Contains(sContent, "\n---\n"+expectedBodyPart+"\n")) {
			t.Errorf("File %s content\n%s\ndoes not correctly prepend frontmatter and preserve content in the expected structure", testFileNoFrontmatter, sContent)
		}
	}
}

func TestGetSingleField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\nmessage: Hello Test\nauthor: Tester\n---\nContent here."
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCmd("get", "message", testFile)
	assertNoError(t, err, stderr)
	trimmedStdout := strings.TrimSpace(stdout)
	if trimmedStdout != "Hello Test" {
		t.Errorf("Expected stdout to be 'Hello Test', got '%s'", trimmedStdout)
	}
	if strings.Contains(stdout, "author:") {
		t.Errorf("Expected only 'message' field value, but got more: %s", stdout)
	}
}

func TestGetAllFrontmatter(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\nmessage: Hello All\ncount: 123\n---\nBody"
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCmd("get", testFile)
	assertNoError(t, err, stderr)
	assertStringContains(t, stdout, "message: Hello All")
	assertStringContains(t, stdout, "count: 123")
	if strings.Contains(stdout, "Body") {
		t.Errorf("Expected only frontmatter, but got body content: %s", stdout)
	}
}

func TestGetFieldFromFileWithoutFrontmatter(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "No frontmatter here."
	if err := setupTestFileNoFrontmatter(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runCmd("get", "message", testFileNoFrontmatter)
	assertExitCode(t, err, 2)
	// Should not output anything to stdout when returning error code 2 (not found)
	trimmedStdout := strings.TrimSpace(stdout)
	if trimmedStdout != "" {
		t.Errorf("Expected no output for 404 case, got '%s'", trimmedStdout)
	}
}

func TestGetNonExistentField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\nexists: yes\n---\nContent"
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runCmd("get", "nonexistent", testFile)
	assertExitCode(t, err, 2)
	// Should not output anything to stdout when returning error code 2 (not found)
	trimmedStdout := strings.TrimSpace(stdout)
	if trimmedStdout != "" {
		t.Errorf("Expected no output for 404 case, got '%s'", trimmedStdout)
	}
}

func TestSetDryRun(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ntitle: Original\n---\nBody"
	originalFileContent := initialContent
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCmd("set", "--dry-run", "message=Dry Run Test", testFile)
	assertNoError(t, err, stderr)
	assertStringContains(t, stdout, "message: Dry Run Test")
	assertStringContains(t, stdout, "title: Original")
	assertStringContains(t, stdout, "---")
	assertStringContains(t, stdout, "Body")

	currentContent, _ := os.ReadFile(testFile)
	if string(currentContent) != originalFileContent {
		t.Errorf("File %s was modified during --dry-run. Content:\n%s", testFile, string(currentContent))
	}
}

func TestGetDryRun(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\nfield: value\n---\nContent"
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCmd("get", "--dry-run", "field", testFile)
	assertNoError(t, err, stderr)
	trimmedStdout := strings.TrimSpace(stdout)
	if trimmedStdout != "value" {
		t.Errorf("Expected stdout to be 'value', got '%s'", trimmedStdout)
	}

	stdoutAll, stderrAll, errAll := runCmd("get", "--dry-run", testFile)
	assertNoError(t, errAll, stderrAll)
	assertStringContains(t, stdoutAll, "field: value")
}

func TestDeleteFrontmatter(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ndelete: me\n---\nKeep this body."
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("delete", testFile)
	assertNoError(t, err, stderr)

	content, _ := os.ReadFile(testFile)
	sContent := string(content)
	if strings.Contains(sContent, "---") || strings.Contains(sContent, "delete: me") {
		t.Errorf("Frontmatter not deleted. File content:\n%s", sContent)
	}
	assertStringContains(t, sContent, "Keep this body.")
}

func TestDeleteFrontmatterDryRun(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ndelete: this\n---\nBody remains."
	originalFileContent := initialContent
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCmd("delete", "--dry-run", testFile)
	assertNoError(t, err, stderr)
	if strings.Contains(stdout, "---") || strings.Contains(stdout, "delete: this") {
		t.Errorf("Dry run output for delete still contains frontmatter: %s", stdout)
	}
	assertStringContains(t, stdout, "Body remains.")

	currentContent, _ := os.ReadFile(testFile)
	if string(currentContent) != originalFileContent {
		t.Errorf("File %s was modified during delete --dry-run. Content:\n%s", testFile, string(currentContent))
	}
}

func TestSetFieldWithEqualsInValue(t *testing.T) {
	defer cleanupTestFiles()
	if err := setupTestFileEmpty(); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runCmd("set", "url=http://example.com?query=123", testFileEmpty)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileEmpty, "url: http://example.com?query=123")
}

func TestSetFieldWithSpacesInValueAndQuotes(t *testing.T) {
	defer cleanupTestFiles()
	if err := setupTestFileEmpty(); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runCmd("set", "text=value with spaces", testFileEmpty)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileEmpty, "text: value with spaces")
}

func TestSetBooleanField(t *testing.T) {
	defer cleanupTestFiles()
	if err := setupTestFileEmpty(); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runCmd("set", "isTrue=true", testFileEmpty)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileEmpty, "isTrue: true")

	_, stderr, err = runCmd("set", "isFalse=false", testFileEmpty)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileEmpty, "isFalse: false")
}

func TestSetNumberField(t *testing.T) {
	defer cleanupTestFiles()
	if err := setupTestFileEmpty(); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runCmd("set", "count=123", testFileEmpty)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileEmpty, "count: 123")

	_, stderr, err = runCmd("set", "value=12.34", testFileEmpty)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileEmpty, "value: 12.34")
}

func TestSetArrayField(t *testing.T) {
	defer cleanupTestFiles()
	if err := setupTestFileEmpty(); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("set", `tags=["tag1", "tag2", "tag3"]`, testFileEmpty)
	assertNoError(t, err, stderr)

	content, _ := os.ReadFile(testFileEmpty)
	sContent := string(content)

	assertStringContains(t, sContent, "tags:")
	assertStringContains(t, sContent, "tag1")
	assertStringContains(t, sContent, "tag2")
	assertStringContains(t, sContent, "tag3")

	if !(strings.Contains(sContent, "- tag1") && strings.Contains(sContent, "- tag2") && strings.Contains(sContent, "- tag3")) {
		if !strings.Contains(sContent, "tags: [tag1, tag2, tag3]") && !strings.Contains(sContent, "tags: [ tag1, tag2, tag3 ]") {
			t.Errorf("Array field not set as expected list. Content:\n%s", sContent)
		}
	}
}

func TestSetInSubdirectory(t *testing.T) {
	defer cleanupTestFiles()
	subDir := "sub"
	testFileInSubDir := subDir + "/" + "file_in_sub.md"

	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	if err := os.WriteFile(testFileInSubDir, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create file in subdirectory: %v", err)
	}

	defer func() {
		os.RemoveAll(subDir)
	}()

	_, stderr, err := runCmd("set", "message=HelloSub", testFileInSubDir)
	assertNoError(t, err, stderr)
	assertFileContains(t, testFileInSubDir, "message: HelloSub")
}

func TestOnlyFrontmatterFileCases(t *testing.T) {
	// Plik z samym frontmatter, bez body
	file := "only_fm.md"
	content := "---\na: 1\nb: 2\n---"
	err := os.WriteFile(file, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	// get all
	stdout, stderr, err := runCmd("get", file)
	assertNoError(t, err, stderr)
	if !strings.Contains(stdout, "a: 1") || !strings.Contains(stdout, "b: 2") {
		t.Errorf("Expected frontmatter fields a and b, got: %s", stdout)
	}

	// set new field
	_, stderr, err = runCmd("set", "new=3", file)
	assertNoError(t, err, stderr)
	data, _ := os.ReadFile(file)
	sData := string(data)
	if !strings.Contains(sData, "new: 3") || !strings.Contains(sData, "a: 1") {
		t.Errorf("Expected new and a in frontmatter, got: %s", sData)
	}

	// delete frontmatter
	_, stderr, err = runCmd("delete", file)
	assertNoError(t, err, stderr)
	data, _ = os.ReadFile(file)
	sData = string(data)
	if strings.TrimSpace(sData) != "" {
		t.Errorf("Expected file to be empty after delete, got: %s", sData)
	}

	// dry-run delete
	stdout, stderr, err = runCmd("delete", "--dry-run", file)
	assertNoError(t, err, stderr)
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("Expected dry-run delete to produce empty output, got: %s", stdout)
	}
}

func TestBodyWithSeparators(t *testing.T) {
	file := "sep_body.md"
	content := "---\nkey: val\n---\nLine1\n---\nLine2\n---\nLine3"
	err := os.WriteFile(file, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	// ensure get body correctly ignores extra separators
	stdout, stderr, err := runCmd("get", file)
	assertNoError(t, err, stderr)
	assertStringContains(t, stdout, "key: val")

	// delete frontmatter should leave rest including separators
	_, stderr, err = runCmd("delete", file)
	assertNoError(t, err, stderr)
	data, _ := os.ReadFile(file)
	sData := string(data)
	if !strings.Contains(sData, "Line1") || !strings.Contains(sData, "---") {
		t.Errorf("Expected body with separators preserved, got: %s", sData)
	}
}

func TestOverrideScalarToMap(t *testing.T) {
	file := "override.md"
	content := "---\na: scalar\n---\nBody"
	err := os.WriteFile(file, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	_, stderr, err := runCmd("set", "a.b=child", file)
	assertNoError(t, err, stderr)
	data, _ := os.ReadFile(file)
	sData := string(data)
	// a should now be a map with b: child
	if !strings.Contains(sData, "a:") || !strings.Contains(sData, "b: child") {
		t.Errorf("Expected a as map with b, got: %s", sData)
	}
	// old scalar should be gone
	if strings.Contains(sData, "scalar") {
		t.Errorf("Old scalar value should be removed, got: %s", sData)
	}
}

func TestJSONMapValueParsing(t *testing.T) {
	file := "json.md"
	err := os.WriteFile(file, []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	// setting value as JSON-like map
	_, stderr, err := runCmd("set", `config={"x":1,"y":"two"}`, file)
	assertNoError(t, err, stderr)
	data, _ := os.ReadFile(file)
	sData := string(data)
	if !strings.Contains(sData, "config:") || !strings.Contains(sData, "x: 1") || !strings.Contains(sData, "y: two") {
		t.Errorf("Expected config map with x and y, got: %s", sData)
	}
}

func TestGetAllFromFileWithoutFrontmatter(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "No frontmatter here."
	if err := setupTestFileNoFrontmatter(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runCmd("get", testFileNoFrontmatter)
	assertExitCode(t, err, 2)
	// Should not output anything to stdout when returning error code 2 (not found)
	trimmedStdout := strings.TrimSpace(stdout)
	if trimmedStdout != "" {
		t.Errorf("Expected no output for 404 case, got '%s'", trimmedStdout)
	}
}

func TestDeleteSingleField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ntitle: Hello\nauthor: John\ndate: 2023-01-01\n---\nBody content."
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("delete", "author", testFile)
	assertNoError(t, err, stderr)

	stdout, stderr, err := runCmd("get", testFile)
	assertNoError(t, err, stderr)

	// Should still have title and date, but not author
	assertStringContains(t, stdout, "title: Hello")
	assertStringContains(t, stdout, "date: 2023-01-01")
	if strings.Contains(stdout, "author") {
		t.Errorf("Field 'author' should have been deleted, but was found in: %s", stdout)
	}
}

func TestDeleteNestedField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ntitle: Test\nconfig:\n  debug: true\n  timeout: 30\n  nested:\n    value: deep\n---\nBody content."
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("delete", "config.debug", testFile)
	assertNoError(t, err, stderr)

	stdout, stderr, err := runCmd("get", testFile)
	assertNoError(t, err, stderr)

	// Should still have title, config.timeout, and config.nested.value
	assertStringContains(t, stdout, "title: Test")
	assertStringContains(t, stdout, "timeout: 30")
	assertStringContains(t, stdout, "value: deep")
	if strings.Contains(stdout, "debug") {
		t.Errorf("Field 'config.debug' should have been deleted, but was found in: %s", stdout)
	}
}

func TestDeleteMultipleFields(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ntitle: Test\nauthor: John\ndate: 2023-01-01\ntags:\n  - go\n  - cli\n---\nBody content."
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("delete", "author", "date", testFile)
	assertNoError(t, err, stderr)

	stdout, stderr, err := runCmd("get", testFile)
	assertNoError(t, err, stderr)

	// Should still have title and tags, but not author or date
	assertStringContains(t, stdout, "title: Test")
	assertStringContains(t, stdout, "tags:")
	assertStringContains(t, stdout, "- go")
	if strings.Contains(stdout, "author") || strings.Contains(stdout, "date") {
		t.Errorf("Fields 'author' and 'date' should have been deleted, but were found in: %s", stdout)
	}
}

func TestDeleteNonExistentField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ntitle: Test\n---\nBody content."
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("delete", "nonexistent", testFile)
	assertNoError(t, err, stderr)

	stdout, stderr, err := runCmd("get", testFile)
	assertNoError(t, err, stderr)

	// Should still have title unchanged
	assertStringContains(t, stdout, "title: Test")
}

func TestDeleteFieldDryRun(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\ntitle: Test\nauthor: John\n---\nBody content."
	originalContent := initialContent
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCmd("delete", "author", "--dry-run", testFile)
	assertNoError(t, err, stderr)

	// Should show result without author field
	assertStringContains(t, stdout, "title: Test")
	assertStringContains(t, stdout, "Body content.")
	if strings.Contains(stdout, "author") {
		t.Errorf("Dry run should not show deleted field 'author', but found in: %s", stdout)
	}

	// File should remain unchanged
	currentContent, _ := os.ReadFile(testFile)
	if string(currentContent) != originalContent {
		t.Errorf("File was modified during dry run. Expected:\n%s\nGot:\n%s", originalContent, string(currentContent))
	}
}

func TestDeleteDeepNestedField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\nconfig:\n  database:\n    host: localhost\n    port: 5432\n    credentials:\n      user: admin\n      pass: secret\n---\nBody content."
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd("delete", "config.database.credentials.pass", testFile)
	assertNoError(t, err, stderr)

	stdout, stderr, err := runCmd("get", testFile)
	assertNoError(t, err, stderr)

	// Should still have other fields but not the password
	assertStringContains(t, stdout, "host: localhost")
	assertStringContains(t, stdout, "port: 5432")
	assertStringContains(t, stdout, "user: admin")
	if strings.Contains(stdout, "pass: secret") {
		t.Errorf("Field 'config.database.credentials.pass' should have been deleted, but was found in: %s", stdout)
	}
}
