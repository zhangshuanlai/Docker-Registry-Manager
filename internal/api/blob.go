package api

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// handleBlobGet handles GET requests for blobs
func (r *Router) handleBlobGet(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	digest := vars["digest"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	if !r.isValidDigest(digest) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeDigestInvalid, "Invalid digest")
		return
	}

	// Get blob reader
	reader, size, err := r.storage.GetBlob(digest)
	if err != nil {
		logrus.Errorf("Failed to get blob %s: %v", digest, err)
		r.writeError(w, http.StatusNotFound, ErrorCodeBlobUnknown, "Blob not found")
		return
	}
	defer reader.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)

	// Stream blob data
	if _, err := io.Copy(w, reader); err != nil {
		logrus.Errorf("Failed to stream blob %s: %v", digest, err)
	}
}

// handleBlobHead handles HEAD requests for blobs
func (r *Router) handleBlobHead(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	digest := vars["digest"]

	if !r.isValidRepositoryName(name) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !r.isValidDigest(digest) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if blob exists
	size, err := r.storage.GetBlobSize(digest)
	if err != nil {
		logrus.Errorf("Failed to get blob size %s: %v", digest, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
}

// handleBlobDelete handles DELETE requests for blobs
func (r *Router) handleBlobDelete(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	digest := vars["digest"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	if !r.isValidDigest(digest) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeDigestInvalid, "Invalid digest")
		return
	}

	// Delete blob
	if err := r.storage.DeleteBlob(digest); err != nil {
		logrus.Errorf("Failed to delete blob %s: %v", digest, err)
		r.writeError(w, http.StatusNotFound, ErrorCodeBlobUnknown, "Blob not found")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// handleBlobUploadPost handles POST requests to initiate blob uploads
func (r *Router) handleBlobUploadPost(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Check for monolithic upload (digest parameter)
	digest := req.URL.Query().Get("digest")
	if digest != "" {
		r.handleMonolithicUpload(w, req, name, digest)
		return
	}

	// Start chunked upload
	uploadID, err := r.storage.StartBlobUpload()
	if err != nil {
		logrus.Errorf("Failed to start blob upload: %v", err)
		r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to start upload")
		return
	}

	// Set response headers
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uploadID))
	w.Header().Set("Range", "0-0")
	w.Header().Set("Docker-Upload-UUID", uploadID)
	w.WriteHeader(http.StatusAccepted)
}

// handleMonolithicUpload handles monolithic blob uploads
func (r *Router) handleMonolithicUpload(w http.ResponseWriter, req *http.Request, name, digest string) {
	if !r.isValidDigest(digest) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeDigestInvalid, "Invalid digest")
		return
	}

	// Read the entire blob
	data, err := io.ReadAll(req.Body)
	if err != nil {
		logrus.Errorf("Failed to read blob data: %v", err)
		r.writeError(w, http.StatusBadRequest, ErrorCodeBlobUploadInvalid, "Failed to read blob data")
		return
	}

	// Verify digest
	hash := sha256.Sum256(data)
	calculatedDigest := fmt.Sprintf("sha256:%x", hash)
	if calculatedDigest != digest {
		r.writeError(w, http.StatusBadRequest, ErrorCodeDigestInvalid, "Digest mismatch")
		return
	}

	// Store blob
	if err := r.storage.PutBlob(digest, data); err != nil {
		logrus.Errorf("Failed to store blob %s: %v", digest, err)
		r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to store blob")
		return
	}

	// Set response headers
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(http.StatusCreated)
}

// handleBlobUploadPatch handles PATCH requests for chunked uploads
func (r *Router) handleBlobUploadPatch(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	uuid := vars["uuid"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Read chunk data
	data, err := io.ReadAll(req.Body)
	if err != nil {
		logrus.Errorf("Failed to read chunk data: %v", err)
		r.writeError(w, http.StatusBadRequest, ErrorCodeBlobUploadInvalid, "Failed to read chunk data")
		return
	}

	// Append chunk to upload
	offset, err := r.storage.AppendBlobUpload(uuid, data)
	if err != nil {
		logrus.Errorf("Failed to append to blob upload %s: %v", uuid, err)
		r.writeError(w, http.StatusNotFound, ErrorCodeBlobUploadUnknown, "Upload not found")
		return
	}

	// Set response headers
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid))
	w.Header().Set("Range", fmt.Sprintf("0-%d", offset-1))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.WriteHeader(http.StatusAccepted)
}

// handleBlobUploadPut handles PUT requests to complete chunked uploads
func (r *Router) handleBlobUploadPut(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	uuid := vars["uuid"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Get digest from query parameter
	digest := req.URL.Query().Get("digest")
	if !r.isValidDigest(digest) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeDigestInvalid, "Invalid or missing digest")
		return
	}

	// Read final chunk (if any)
	data, err := io.ReadAll(req.Body)
	if err != nil {
		logrus.Errorf("Failed to read final chunk data: %v", err)
		r.writeError(w, http.StatusBadRequest, ErrorCodeBlobUploadInvalid, "Failed to read final chunk")
		return
	}

	// Complete upload
	if err := r.storage.CompleteBlobUpload(uuid, digest, data); err != nil {
		logrus.Errorf("Failed to complete blob upload %s: %v", uuid, err)
		if strings.Contains(err.Error(), "digest mismatch") {
			r.writeError(w, http.StatusBadRequest, ErrorCodeDigestInvalid, "Digest mismatch")
		} else {
			r.writeError(w, http.StatusNotFound, ErrorCodeBlobUploadUnknown, "Upload not found")
		}
		return
	}

	// Set response headers
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(http.StatusCreated)
}

// handleBlobUploadGet handles GET requests for upload status
func (r *Router) handleBlobUploadGet(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	uuid := vars["uuid"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Get upload status
	offset, err := r.storage.GetBlobUploadStatus(uuid)
	if err != nil {
		logrus.Errorf("Failed to get blob upload status %s: %v", uuid, err)
		r.writeError(w, http.StatusNotFound, ErrorCodeBlobUploadUnknown, "Upload not found")
		return
	}

	// Set response headers
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid))
	w.Header().Set("Range", fmt.Sprintf("0-%d", offset-1))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.WriteHeader(http.StatusNoContent)
}

// handleBlobUploadDelete handles DELETE requests to cancel uploads
func (r *Router) handleBlobUploadDelete(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	uuid := vars["uuid"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Cancel upload
	if err := r.storage.CancelBlobUpload(uuid); err != nil {
		logrus.Errorf("Failed to cancel blob upload %s: %v", uuid, err)
		r.writeError(w, http.StatusNotFound, ErrorCodeBlobUploadUnknown, "Upload not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

