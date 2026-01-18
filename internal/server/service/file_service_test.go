package service

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestFileService_GetFile(t *testing.T) {
	tempDir := t.TempDir()

	testContent := []byte("test content")
	testFilePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFilePath, testContent, 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	// Create subdirectory and file
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	subFilePath := filepath.Join(subDir, "nested.json")
	subFileContent := []byte(`{"key": "value"}`)
	err = os.WriteFile(subFilePath, subFileContent, 0644)
	if err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	service, err := NewFileService(tempDir, logger)
	if err != nil {
		t.Fatalf("failed to create file service: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		wantError   bool
		wantContent []byte
	}{
		{
			name:        "read existing file",
			path:        "test.txt",
			wantError:   false,
			wantContent: testContent,
		},
		{
			name:        "read nested file",
			path:        "subdir/nested.json",
			wantError:   false,
			wantContent: subFileContent,
		},
		{
			name:      "file not found",
			path:      "nonexistent.txt",
			wantError: true,
		},
		{
			name:      "path traversal attempt",
			path:      "../../../etc/passwd",
			wantError: true,
		},
		{
			name:      "path traversal with clean path",
			path:      "subdir/../../outside.txt",
			wantError: true,
		},
		{
			name:      "directory instead of file",
			path:      "subdir",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.GetFile(tt.path)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if string(resp.Content) != string(tt.wantContent) {
				t.Errorf("content mismatch: got %q, want %q", resp.Content, tt.wantContent)
			}

			if resp.Size != int64(len(tt.wantContent)) {
				t.Errorf("size mismatch: got %d, want %d", resp.Size, len(tt.wantContent))
			}
		})
	}
}

func TestFileService_FileExists(t *testing.T) {
	tempDir := t.TempDir()

	testFilePath := filepath.Join(tempDir, "exists.txt")
	err := os.WriteFile(testFilePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	service, err := NewFileService(tempDir, logger)
	if err != nil {
		t.Fatalf("failed to create file service: %v", err)
	}

	tests := []struct {
		name   string
		path   string
		exists bool
	}{
		{
			name:   "existing file",
			path:   "exists.txt",
			exists: true,
		},
		{
			name:   "nonexistent file",
			path:   "notexists.txt",
			exists: false,
		},
		{
			name:   "path traversal",
			path:   "../outside.txt",
			exists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := service.FileExists(tt.path)
			if exists != tt.exists {
				t.Errorf("FileExists() = %v, want %v", exists, tt.exists)
			}
		})
	}
}

func TestFileService_DetectContentType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tempDir := t.TempDir()
	service, _ := NewFileService(tempDir, logger)

	tests := []struct {
		name     string
		fileName string
		want     string
	}{
		{"json file", "file.json", "application/json"},
		{"yaml file", "file.yaml", "application/x-yaml"},
		{"html file", "file.html", "text/html; charset=utf-8"},
		{"javascript", "file.js", "text/javascript; charset=utf-8"}, 
		{"css file", "file.css", "text/css; charset=utf-8"},         
		{"svg image", "file.svg", "image/svg+xml"},
		{"markdown", "file.md", "text/markdown"},
		{"unknown", "file.unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.detectContentType(tt.fileName, nil)
			if got != tt.want {
				t.Errorf("detectContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileService_ListFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	files := []string{"file1.txt", "file2.json", "file3.md"}
	for _, file := range files {
		err := os.WriteFile(filepath.Join(tempDir, file), []byte("content"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Create subdirectory (should be ignored)
	err := os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	service, err := NewFileService(tempDir, logger)
	if err != nil {
		t.Fatalf("failed to create file service: %v", err)
	}

	result, err := service.ListFiles(".")
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	if len(result) != len(files) {
		t.Errorf("ListFiles() returned %d files, want %d", len(result), len(files))
	}

	fileMap := make(map[string]bool)
	for _, f := range result {
		fileMap[f] = true
	}

	for _, expected := range files {
		if !fileMap[expected] {
			t.Errorf("expected file %q not found in results", expected)
		}
	}
}
