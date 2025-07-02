package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Manifest represents a Docker manifest
type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

// handleManifestGet handles GET requests for manifests
func (r *Router) handleManifestGet(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	reference := vars["reference"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Check if reference is a tag or digest
	var digest string
	if r.isValidDigest(reference) {
		digest = reference
	} else if r.isValidTag(reference) {
		// Look up digest by tag
		tagDigest, err := r.storage.GetTagDigest(name, reference)
		if err != nil {
			logrus.Errorf("Failed to get digest for tag %s/%s: %v", name, reference, err)
			r.writeError(w, http.StatusNotFound, ErrorCodeManifestUnknown, "Manifest not found")
			return
		}
		digest = tagDigest
	} else {
		r.writeError(w, http.StatusBadRequest, ErrorCodeTagInvalid, "Invalid tag or digest")
		return
	}

	// Get manifest data
	manifestData, mediaType, err := r.storage.GetManifest(name, digest)
	if err != nil {
		logrus.Errorf("Failed to get manifest %s/%s: %v", name, digest, err)
		r.writeError(w, http.StatusNotFound, ErrorCodeManifestUnknown, "Manifest not found")
		return
	}

	// Set headers
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", strconv.Itoa(len(manifestData)))
	w.WriteHeader(http.StatusOK)
	w.Write(manifestData)
}

// handleManifestPut handles PUT requests for manifests
func (r *Router) handleManifestPut(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	reference := vars["reference"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Read manifest data
	manifestData, err := io.ReadAll(req.Body)
	if err != nil {
		logrus.Errorf("Failed to read manifest data: %v", err)
		r.writeError(w, http.StatusBadRequest, ErrorCodeManifestInvalid, "Failed to read manifest")
		return
	}

	// Validate manifest JSON
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		logrus.Errorf("Failed to parse manifest JSON: %v", err)
		r.writeError(w, http.StatusBadRequest, ErrorCodeManifestInvalid, "Invalid manifest JSON")
		return
	}

	// Calculate digest
	hash := sha256.Sum256(manifestData)
	digest := fmt.Sprintf("sha256:%x", hash)

	// Get content type
	mediaType := req.Header.Get("Content-Type")
	if mediaType == "" {
		mediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}

	// Store manifest
	if err := r.storage.PutManifest(name, digest, manifestData, mediaType); err != nil {
		logrus.Errorf("Failed to store manifest %s/%s: %v", name, digest, err)
		r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to store manifest")
		return
	}

	// If reference is a tag, create tag mapping
	if !r.isValidDigest(reference) && r.isValidTag(reference) {
		if err := r.storage.PutTag(name, reference, digest); err != nil {
			logrus.Errorf("Failed to create tag %s/%s -> %s: %v", name, reference, digest, err)
			r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to create tag")
			return
		}
	}

	// Set response headers
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/manifests/%s", name, digest))
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(http.StatusCreated)
}

// handleManifestHead handles HEAD requests for manifests
func (r *Router) handleManifestHead(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	reference := vars["reference"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Check if reference is a tag or digest
	var digest string
	if r.isValidDigest(reference) {
		digest = reference
	} else if r.isValidTag(reference) {
		// Look up digest by tag
		tagDigest, err := r.storage.GetTagDigest(name, reference)
		if err != nil {
			logrus.Errorf("Failed to get digest for tag %s/%s: %v", name, reference, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		digest = tagDigest
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if manifest exists
	size, mediaType, err := r.storage.GetManifestInfo(name, digest)
	if err != nil {
		logrus.Errorf("Failed to get manifest info %s/%s: %v", name, digest, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
}

// handleManifestDelete handles DELETE requests for manifests
func (r *Router) handleManifestDelete(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]
	reference := vars["reference"]

	if !r.isValidRepositoryName(name) {
		r.writeError(w, http.StatusBadRequest, ErrorCodeNameInvalid, "Invalid repository name")
		return
	}

	// Check if reference is a tag or digest
	if r.isValidDigest(reference) {
		// Delete manifest by digest
		if err := r.storage.DeleteManifest(name, reference); err != nil {
			logrus.Errorf("Failed to delete manifest %s/%s: %v", name, reference, err)
			r.writeError(w, http.StatusNotFound, ErrorCodeManifestUnknown, "Manifest not found")
			return
		}
	} else if r.isValidTag(reference) {
		// Delete tag
		if err := r.storage.DeleteTag(name, reference); err != nil {
			logrus.Errorf("Failed to delete tag %s/%s: %v", name, reference, err)
			r.writeError(w, http.StatusNotFound, ErrorCodeManifestUnknown, "Tag not found")
			return
		}
	} else {
		r.writeError(w, http.StatusBadRequest, ErrorCodeTagInvalid, "Invalid tag or digest")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

