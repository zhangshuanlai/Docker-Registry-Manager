package storage

import (
	"io"
)

// Storage defines the interface for registry storage backend
type Storage interface {
	// Repository operations
	ListRepositories() ([]string, error)

	// Tag operations
	ListTags(repository string) ([]string, error)
	GetTagDigest(repository, tag string) (string, error)
	PutTag(repository, tag, digest string) error
	DeleteTag(repository, tag string) error

	// Manifest operations
	GetManifest(repository, digest string) ([]byte, string, error)
	GetManifestInfo(repository, digest string) (int64, string, error)
	PutManifest(repository, digest string, data []byte, mediaType string) error
	DeleteManifest(repository, digest string) error

	// Blob operations
	GetBlob(digest string) (io.ReadCloser, int64, error)
	GetBlobSize(digest string) (int64, error)
	PutBlob(digest string, data []byte) error
	DeleteBlob(digest string) error

	// Blob upload operations
	StartBlobUpload() (string, error)
	AppendBlobUpload(uploadID string, data []byte) (int64, error)
	GetBlobUploadStatus(uploadID string) (int64, error)
	CompleteBlobUpload(uploadID, digest string, finalChunk []byte) error
	CancelBlobUpload(uploadID string) error

	// Description operations
	GetRepositoryDescription(repository string) (string, error)
	PutRepositoryDescription(repository string, description string) error

	// Storage size operations
	GetTotalStorageSize() (int64, error)
}

// BlobUpload represents an ongoing blob upload
type BlobUpload struct {
	ID       string
	Size     int64
	FilePath string
}

// RepositoryInfo represents repository information
type RepositoryInfo struct {
	Name     string
	TagCount int
	Size     int64
}

// TagInfo represents tag information
type TagInfo struct {
	Name   string
	Digest string
	Size   int64
}
