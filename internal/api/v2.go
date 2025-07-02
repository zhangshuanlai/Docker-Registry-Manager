package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Docker Registry API v2 error codes
const (
	ErrorCodeBlobUnknown         = "BLOB_UNKNOWN"
	ErrorCodeBlobUploadInvalid   = "BLOB_UPLOAD_INVALID"
	ErrorCodeBlobUploadUnknown   = "BLOB_UPLOAD_UNKNOWN"
	ErrorCodeDigestInvalid       = "DIGEST_INVALID"
	ErrorCodeManifestBlobUnknown = "MANIFEST_BLOB_UNKNOWN"
	ErrorCodeManifestInvalid     = "MANIFEST_INVALID"
	ErrorCodeManifestUnknown     = "MANIFEST_UNKNOWN"
	ErrorCodeNameInvalid         = "NAME_INVALID"
	ErrorCodeNameUnknown         = "NAME_UNKNOWN"
	ErrorCodeSizeInvalid         = "SIZE_INVALID"
	ErrorCodeTagInvalid          = "TAG_INVALID"
	ErrorCodeUnauthorized        = "UNAUTHORIZED"
	ErrorCodeDenied              = "DENIED"
	ErrorCodeUnsupported         = "UNSUPPORTED"
	ErrorCodeUnknown             = "UNKNOWN"
)

// RegistryError represents a registry API error
type RegistryError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail,omitempty"`
}

// ErrorResponse represents the error response format
type ErrorResponse struct {
	Errors []RegistryError `json:"errors"`
}

// CatalogResponse represents the catalog response
type CatalogResponse struct {
	Repositories []string `json:"repositories"`
}

// TagsResponse represents the tags list response
type TagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// handleV2Base handles the base v2 endpoint
func (r *Router) handleV2Base(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusOK)
}

// handleCatalog handles the catalog endpoint
func (r *Router) handleCatalog(w http.ResponseWriter, req *http.Request) {
	repositories, err := r.storage.ListRepositories()
	if err != nil {
		logrus.Errorf("Failed to list repositories: %v", err)
		r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to list repositories")
		return
	}

	response := CatalogResponse{
		Repositories: repositories,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleTagsList handles the tags list endpoint
func (r *Router) handleTagsList(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	tags, err := r.storage.ListTags(name)
	if err != nil {
		logrus.Errorf("Failed to list tags for repository %s: %v", name, err)
		r.writeError(w, http.StatusNotFound, ErrorCodeNameUnknown, "Repository not found")
		return
	}

	response := TagsResponse{
		Name: name,
		Tags: tags,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeError writes an error response
func (r *Router) writeError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := ErrorResponse{
		Errors: []RegistryError{
			{
				Code:    errorCode,
				Message: message,
			},
		},
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// isValidRepositoryName validates repository name
func (r *Router) isValidRepositoryName(name string) bool {
	if name == "" {
		return false
	}
	// Add more validation as needed
	return true
}

// isValidTag validates tag name
func (r *Router) isValidTag(tag string) bool {
	if tag == "" {
		return false
	}
	// Add more validation as needed
	return true
}

// isValidDigest validates digest format
func (r *Router) isValidDigest(digest string) bool {
	if len(digest) < 7 || digest[:7] != "sha256:" {
		return false
	}
	return len(digest) == 71 // sha256: + 64 hex characters
}

