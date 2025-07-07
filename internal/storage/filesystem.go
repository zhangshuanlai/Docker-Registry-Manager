package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FilesystemStorage implements Storage interface using filesystem
type FilesystemStorage struct {
	basePath string
	uploads  map[string]*BlobUpload
	mutex    sync.RWMutex
}

// NewFilesystemStorage creates a new filesystem storage instance
func NewFilesystemStorage(basePath string) (*FilesystemStorage, error) {
	// Create base directories
	dirs := []string{
		filepath.Join(basePath, "blobs"),
		filepath.Join(basePath, "repositories"),
		filepath.Join(basePath, "uploads"),
		filepath.Join(basePath, "descriptions"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &FilesystemStorage{
		basePath: basePath,
		uploads:  make(map[string]*BlobUpload),
	}, nil
}

// ListRepositories returns a list of all repositories
func (fs *FilesystemStorage) ListRepositories() ([]string, error) {
	repoPath := filepath.Join(fs.basePath, "repositories")

	var repositories []string
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != repoPath {
			relPath, err := filepath.Rel(repoPath, path)
			if err != nil {
				return err
			}

			// Check if this directory contains manifests
			manifestsPath := filepath.Join(path, "manifests")
			if _, err := os.Stat(manifestsPath); err == nil {
				repositories = append(repositories, strings.ReplaceAll(relPath, string(filepath.Separator), "/"))
			}
		}
		return nil
	})

	return repositories, err
}

// ListTags returns a list of tags for a repository
func (fs *FilesystemStorage) ListTags(repository string) ([]string, error) {
	tagsPath := filepath.Join(fs.basePath, "repositories", repository, "tags")

	if _, err := os.Stat(tagsPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(tagsPath)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, file := range files {
		if !file.IsDir() {
			tags = append(tags, file.Name())
		}
	}

	return tags, nil
}

// GetTagDigest returns the digest for a tag
func (fs *FilesystemStorage) GetTagDigest(repository, tag string) (string, error) {
	tagPath := filepath.Join(fs.basePath, "repositories", repository, "tags", tag)

	data, err := os.ReadFile(tagPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// PutTag creates or updates a tag
func (fs *FilesystemStorage) PutTag(repository, tag, digest string) error {
	tagPath := filepath.Join(fs.basePath, "repositories", repository, "tags", tag)

	if err := os.MkdirAll(filepath.Dir(tagPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(tagPath, []byte(digest), 0644)
}

// DeleteTag removes a tag
func (fs *FilesystemStorage) DeleteTag(repository, tag string) error {
	tagPath := filepath.Join(fs.basePath, "repositories", repository, "tags", tag)
	return os.Remove(tagPath)
}

// GetManifest returns manifest data and media type
func (fs *FilesystemStorage) GetManifest(repository, digest string) ([]byte, string, error) {
	manifestPath := filepath.Join(fs.basePath, "repositories", repository, "manifests", digest)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", err
	}

	// Read metadata
	metaPath := manifestPath + ".meta"
	var metadata struct {
		MediaType string `json:"mediaType"`
	}

	if metaData, err := os.ReadFile(metaPath); err == nil {
		json.Unmarshal(metaData, &metadata)
	}

	if metadata.MediaType == "" {
		metadata.MediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}

	return data, metadata.MediaType, nil
}

// GetManifestInfo returns manifest size and media type
func (fs *FilesystemStorage) GetManifestInfo(repository, digest string) (int64, string, error) {
	manifestPath := filepath.Join(fs.basePath, "repositories", repository, "manifests", digest)

	info, err := os.Stat(manifestPath)
	if err != nil {
		return 0, "", err
	}

	// Read metadata
	metaPath := manifestPath + ".meta"
	var metadata struct {
		MediaType string `json:"mediaType"`
	}

	if metaData, err := os.ReadFile(metaPath); err == nil {
		json.Unmarshal(metaData, &metadata)
	}

	if metadata.MediaType == "" {
		metadata.MediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}

	return info.Size(), metadata.MediaType, nil
}

// PutManifest stores a manifest
func (fs *FilesystemStorage) PutManifest(repository, digest string, data []byte, mediaType string) error {
	manifestPath := filepath.Join(fs.basePath, "repositories", repository, "manifests", digest)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}

	// Write manifest data
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return err
	}

	// Write metadata
	metadata := struct {
		MediaType string `json:"mediaType"`
	}{
		MediaType: mediaType,
	}

	metaData, _ := json.Marshal(metadata)
	metaPath := manifestPath + ".meta"
	return os.WriteFile(metaPath, metaData, 0644)
}

// DeleteManifest removes a manifest
func (fs *FilesystemStorage) DeleteManifest(repository, digest string) error {
	manifestPath := filepath.Join(fs.basePath, "repositories", repository, "manifests", digest)

	// Remove manifest file
	if err := os.Remove(manifestPath); err != nil {
		return err
	}

	// Remove metadata file
	metaPath := manifestPath + ".meta"
	os.Remove(metaPath) // Ignore error for metadata

	return nil
}

// GetBlob returns a blob reader and size
func (fs *FilesystemStorage) GetBlob(digest string) (io.ReadCloser, int64, error) {
	blobPath := fs.getBlobPath(digest)

	info, err := os.Stat(blobPath)
	if err != nil {
		return nil, 0, err
	}

	file, err := os.Open(blobPath)
	if err != nil {
		return nil, 0, err
	}

	return file, info.Size(), nil
}

// GetBlobSize returns the size of a blob
func (fs *FilesystemStorage) GetBlobSize(digest string) (int64, error) {
	blobPath := fs.getBlobPath(digest)

	info, err := os.Stat(blobPath)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

// PutBlob stores a blob
func (fs *FilesystemStorage) PutBlob(digest string, data []byte) error {
	blobPath := fs.getBlobPath(digest)

	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(blobPath, data, 0644)
}

// DeleteBlob removes a blob
func (fs *FilesystemStorage) DeleteBlob(digest string) error {
	blobPath := fs.getBlobPath(digest)
	return os.Remove(blobPath)
}

// StartBlobUpload initiates a new blob upload
func (fs *FilesystemStorage) StartBlobUpload() (string, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	uploadID := fmt.Sprintf("%d", time.Now().UnixNano())
	uploadPath := filepath.Join(fs.basePath, "uploads", uploadID)

	// Create upload file
	file, err := os.Create(uploadPath)
	if err != nil {
		return "", err
	}
	file.Close()

	fs.uploads[uploadID] = &BlobUpload{
		ID:       uploadID,
		Size:     0,
		FilePath: uploadPath,
	}

	return uploadID, nil
}

// AppendBlobUpload appends data to an ongoing upload
func (fs *FilesystemStorage) AppendBlobUpload(uploadID string, data []byte) (int64, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	upload, exists := fs.uploads[uploadID]
	if !exists {
		return 0, fmt.Errorf("upload not found")
	}

	file, err := os.OpenFile(upload.FilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	n, err := file.Write(data)
	if err != nil {
		return 0, err
	}

	upload.Size += int64(n)
	return upload.Size, nil
}

// GetBlobUploadStatus returns the current size of an upload
func (fs *FilesystemStorage) GetBlobUploadStatus(uploadID string) (int64, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	upload, exists := fs.uploads[uploadID]
	if !exists {
		return 0, fmt.Errorf("upload not found")
	}

	return upload.Size, nil
}

// CompleteBlobUpload finalizes an upload and moves it to blob storage
func (fs *FilesystemStorage) CompleteBlobUpload(uploadID, digest string, finalChunk []byte) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	upload, exists := fs.uploads[uploadID]
	if !exists {
		return fmt.Errorf("upload not found")
	}

	// Append final chunk if provided
	if len(finalChunk) > 0 {
		file, err := os.OpenFile(upload.FilePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		file.Write(finalChunk)
		file.Close()
	}

	// Verify digest
	data, err := os.ReadFile(upload.FilePath)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)
	calculatedDigest := fmt.Sprintf("sha256:%x", hash)
	if calculatedDigest != digest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", digest, calculatedDigest)
	}

	// Move to blob storage
	blobPath := fs.getBlobPath(digest)
	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		return err
	}

	if err := os.Rename(upload.FilePath, blobPath); err != nil {
		return err
	}

	// Clean up upload
	delete(fs.uploads, uploadID)

	return nil
}

// CancelBlobUpload cancels an ongoing upload
func (fs *FilesystemStorage) CancelBlobUpload(uploadID string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	upload, exists := fs.uploads[uploadID]
	if !exists {
		return fmt.Errorf("upload not found")
	}

	// Remove upload file
	os.Remove(upload.FilePath)

	// Clean up upload
	delete(fs.uploads, uploadID)

	return nil
}

// getBlobPath returns the filesystem path for a blob
func (fs *FilesystemStorage) getBlobPath(digest string) string {
	// Remove sha256: prefix
	hash := strings.TrimPrefix(digest, "sha256:")

	// Create directory structure: blobs/ab/cd/abcd...
	return filepath.Join(fs.basePath, "blobs", hash[:2], hash[2:4], hash)
}

// GetRepositoryDescription returns the description for a repository
func (fs *FilesystemStorage) GetRepositoryDescription(repository string) (string, error) {
	descPath := filepath.Join(fs.basePath, "descriptions", repository+".md")

	data, err := os.ReadFile(descPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Return empty string if description file does not exist
		}
		return "", err
	}

	return string(data), nil
}

// PutRepositoryDescription saves the description for a repository
func (fs *FilesystemStorage) PutRepositoryDescription(repository string, description string) error {
	descPath := filepath.Join(fs.basePath, "descriptions", repository+".md")

	if err := os.MkdirAll(filepath.Dir(descPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(descPath, []byte(description), 0644)
}

// GetTotalStorageSize calculates the total size of the storage directory
func (fs *FilesystemStorage) GetTotalStorageSize() (int64, error) {
	var totalSize int64
	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log the error but continue walking for other files
			fmt.Fprintf(os.Stderr, "Error accessing path %s: %v\n", path, err)
			return nil // Do not stop walking for this error
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total storage size: %w", err)
	}

	return totalSize, nil
}
