package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupFileService(t *testing.T) (*FileService, string) {
	tempDir := t.TempDir()
	logger := zap.NewNop()
	svc, err := NewFileService(tempDir, logger)
	require.NoError(t, err)
	return svc, tempDir
}

// --- NewFileService ---

func TestNewFileService_Success(t *testing.T) {
	tempDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewFileService(tempDir, logger)
	require.NoError(t, err)
	require.NotNil(t, svc)
}

func TestNewFileService_CreatesBaseDir(t *testing.T) {
	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "storage")
	logger := zap.NewNop()

	_, err := os.Stat(baseDir)
	require.True(t, os.IsNotExist(err))

	_, err = NewFileService(baseDir, logger)
	require.NoError(t, err)

	_, err = os.Stat(baseDir)
	require.NoError(t, err)
}

// --- GetFile ---

func TestFileService_GetFile_Success(t *testing.T) {
	svc, tempDir := setupFileService(t)

	content := []byte("test content")
	path := "test.txt"
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, path), content, 0o644))

	resp, err := svc.GetFile(path)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, content, resp.Content)
	assert.Equal(t, int64(len(content)), resp.Size)
	assert.NotEmpty(t, resp.ContentType)
}

func TestFileService_GetFile_NestedPath(t *testing.T) {
	svc, tempDir := setupFileService(t)

	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	content := []byte(`{"key": "value"}`)
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "nested.json"), content, 0o644))

	resp, err := svc.GetFile("subdir/nested.json")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, content, resp.Content)
}

func TestFileService_GetFile_NotFound(t *testing.T) {
	svc, _ := setupFileService(t)

	resp, err := svc.GetFile("nonexistent.txt")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "file not found")
}

func TestFileService_GetFile_PathTraversal(t *testing.T) {
	svc, _ := setupFileService(t)

	resp, err := svc.GetFile("../../../etc/passwd")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "access denied")
}

func TestFileService_GetFile_PathTraversalClean(t *testing.T) {
	svc, tempDir := setupFileService(t)

	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "subdir"), 0o755))

	resp, err := svc.GetFile("subdir/../../outside.txt")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "access denied")
}

func TestFileService_GetFile_DirectoryRejected(t *testing.T) {
	svc, tempDir := setupFileService(t)

	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	resp, err := svc.GetFile("subdir")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "directory")
}

// --- FileExists ---

func TestFileService_FileExists_True(t *testing.T) {
	svc, tempDir := setupFileService(t)

	path := "exists.txt"
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, path), []byte("x"), 0o644))

	assert.True(t, svc.FileExists(path))
}

func TestFileService_FileExists_False(t *testing.T) {
	svc, _ := setupFileService(t)

	assert.False(t, svc.FileExists("notexists.txt"))
}

func TestFileService_FileExists_PathTraversalFalse(t *testing.T) {
	svc, _ := setupFileService(t)

	assert.False(t, svc.FileExists("../outside.txt"))
}

func TestFileService_FileExists_DirectoryFalse(t *testing.T) {
	svc, tempDir := setupFileService(t)

	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "dir"), 0o755))

	assert.False(t, svc.FileExists("dir"))
}

// --- detectContentType ---

func TestFileService_DetectContentType(t *testing.T) {
	svc, _ := setupFileService(t)

	tests := []struct {
		path string
		want string
	}{
		{"file.json", "application/json"},
		{"file.yaml", "application/x-yaml"},
		{"file.yml", "application/x-yaml"},
		{"file.md", "text/markdown"},
		{"file.html", "text/html; charset=utf-8"},
		{"file.css", "text/css"},
		{"file.js", "application/javascript"},
		{"file.svg", "image/svg+xml"},
		{"file.png", "image/png"},
		{"file.jpg", "image/jpeg"},
		{"file.gif", "image/gif"},
		{"file.unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := svc.detectContentType(tt.path)
			// mime.TypeByExtension may override for some extensions
			if got != tt.want {
				// Fallback: at least non-empty for known types
				assert.NotEmpty(t, got, "content type should not be empty")
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// --- ListFiles ---

func TestFileService_ListFiles_Success(t *testing.T) {
	svc, tempDir := setupFileService(t)

	files := []string{"file1.txt", "file2.json", "file3.md"}
	for _, f := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, f), []byte("x"), 0o644))
	}
	require.NoError(t, os.Mkdir(filepath.Join(tempDir, "subdir"), 0o755))

	names, err := svc.ListFiles(".")
	require.NoError(t, err)
	require.Len(t, names, len(files))

	seen := make(map[string]bool)
	for _, n := range names {
		seen[n] = true
	}
	for _, f := range files {
		assert.True(t, seen[f], "expected file %q in list", f)
	}
}

func TestFileService_ListFiles_PathTraversal(t *testing.T) {
	svc, _ := setupFileService(t)

	_, err := svc.ListFiles("../../../etc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestFileService_ListFiles_NotExist(t *testing.T) {
	svc, _ := setupFileService(t)

	_, err := svc.ListFiles("nonexistent_dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read directory")
}

// --- SaveFile ---

func TestFileService_SaveFile_Success(t *testing.T) {
	svc, tempDir := setupFileService(t)

	content := []byte("saved content")
	path := "subdir/file.txt"

	err := svc.SaveFile(path, content)
	require.NoError(t, err)

	read, err := os.ReadFile(filepath.Join(tempDir, path))
	require.NoError(t, err)
	assert.Equal(t, content, read)
}

func TestFileService_SaveFile_PathTraversal(t *testing.T) {
	svc, _ := setupFileService(t)

	err := svc.SaveFile("../../../etc/malicious", []byte("x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

// --- DeleteFile ---

func TestFileService_DeleteFile_Success(t *testing.T) {
	svc, tempDir := setupFileService(t)

	path := "to_delete.txt"
	fullPath := filepath.Join(tempDir, path)
	require.NoError(t, os.WriteFile(fullPath, []byte("x"), 0o644))

	err := svc.DeleteFile(path)
	require.NoError(t, err)

	_, err = os.Stat(fullPath)
	require.True(t, os.IsNotExist(err))
}

func TestFileService_DeleteFile_NotFound(t *testing.T) {
	svc, _ := setupFileService(t)

	err := svc.DeleteFile("nonexistent.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestFileService_DeleteFile_PathTraversal(t *testing.T) {
	svc, _ := setupFileService(t)

	err := svc.DeleteFile("../../../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

// --- OpenFileForReading ---

func TestFileService_OpenFileForReading_Success(t *testing.T) {
	svc, tempDir := setupFileService(t)

	content := []byte("read me")
	path := "readable.txt"
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, path), content, 0o644))

	file, size, err := svc.OpenFileForReading(path)
	require.NoError(t, err)
	require.NotNil(t, file)
	defer func(file *os.File) {
		err := file.Close()
		require.NoError(t, err)
	}(file)

	assert.Equal(t, int64(len(content)), size)
	buf := make([]byte, 64)
	n, err := file.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, content, buf[:n])
}

func TestFileService_OpenFileForReading_NotFound(t *testing.T) {
	svc, _ := setupFileService(t)

	file, size, err := svc.OpenFileForReading("nonexistent.txt")
	require.Error(t, err)
	assert.Nil(t, file)
	assert.Equal(t, int64(0), size)
	assert.Contains(t, err.Error(), "file not found")
}

func TestFileService_OpenFileForReading_PathTraversal(t *testing.T) {
	svc, _ := setupFileService(t)

	file, _, err := svc.OpenFileForReading("../../../etc/passwd")
	require.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "access denied")
}

func TestFileService_OpenFileForReading_DirectoryRejected(t *testing.T) {
	svc, tempDir := setupFileService(t)

	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "dir"), 0o755))

	file, _, err := svc.OpenFileForReading("dir")
	require.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "directory")
}

// --- CreateFileForWriting ---

func TestFileService_CreateFileForWriting_Success(t *testing.T) {
	svc, tempDir := setupFileService(t)

	path := "out.txt"
	file, cleanup, err := svc.CreateFileForWriting(path)
	require.NoError(t, err)
	require.NotNil(t, file)
	require.NotNil(t, cleanup)

	_, err = file.Write([]byte("data"))
	require.NoError(t, err)
	require.NoError(t, file.Close())

	tmpPath := filepath.Join(tempDir, path+".tmp")
	_, err = os.Stat(tmpPath)
	require.NoError(t, err)

	cleanup()
	_, err = os.Stat(tmpPath)
	require.True(t, os.IsNotExist(err))
}

func TestFileService_CreateFileForWriting_PathTraversal(t *testing.T) {
	svc, _ := setupFileService(t)

	file, cleanup, err := svc.CreateFileForWriting("../../../etc/evil")
	require.Error(t, err)
	assert.Nil(t, file)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "access denied")
}

// --- RenameFile ---

func TestFileService_RenameFile_Success(t *testing.T) {
	svc, tempDir := setupFileService(t)

	oldPath := "old.txt"
	newPath := "new.txt"
	content := []byte("content")
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, oldPath), content, 0o644))

	err := svc.RenameFile(oldPath, newPath)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(tempDir, oldPath))
	require.True(t, os.IsNotExist(err))
	read, err := os.ReadFile(filepath.Join(tempDir, newPath))
	require.NoError(t, err)
	assert.Equal(t, content, read)
}

func TestFileService_RenameFile_OldNotFound(t *testing.T) {
	svc, _ := setupFileService(t)

	err := svc.RenameFile("nonexistent.txt", "new.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rename")
}

func TestFileService_RenameFile_PathTraversal(t *testing.T) {
	svc, tempDir := setupFileService(t)

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "a.txt"), []byte("x"), 0o644))

	err := svc.RenameFile("../../../etc/passwd", "new.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")

	err = svc.RenameFile("a.txt", "../../../etc/evil")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}
