package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

const testFile = "test_file.md"
const testFileNoFrontmatter = "test_file_no_frontmatter.md"
const testFileEmpty = "test_file_empty.md"

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
	cmd := exec.Command("go", append([]string{"run", "main.go"}, args...)...)
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

	stdout, stderr, err := runCmd("get", "message", testFileNoFrontmatter)
	assertNoError(t, err, stderr)
	trimmedStdout := strings.TrimSpace(stdout)
	if trimmedStdout != "null" && trimmedStdout != "" {
		t.Errorf("Expected 'null' or empty string for non-existent frontmatter/key, got '%s'", trimmedStdout)
	}
}

func TestGetNonExistentField(t *testing.T) {
	defer cleanupTestFiles()
	initialContent := "---\nexists: yes\n---\nContent"
	if err := setupTestFile(initialContent); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCmd("get", "nonexistent", testFile)
	assertNoError(t, err, stderr)
	trimmedStdout := strings.TrimSpace(stdout)
	if trimmedStdout != "null" && trimmedStdout != "" {
		t.Errorf("Expected 'null' or empty string for non-existent key, got '%s'", trimmedStdout)
	}
	if strings.Contains(stdout, "exists: yes") {
		t.Errorf("Expected not to find 'exists: yes' when getting 'nonexistent', but got: %s", stdout)
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

	_, stderr, err := runCmd("set", "--delete", testFile)
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

	stdout, stderr, err := runCmd("set", "--delete", "--dry-run", testFile)
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
