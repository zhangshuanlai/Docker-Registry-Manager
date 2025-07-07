package api

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// WebData represents data passed to web templates
type WebData struct {
	Title        string
	Repositories []RepositoryData
	Repository   *RepositoryData
	Stats        *StatsData
}

// RepositoryData represents repository information for web display
type RepositoryData struct {
	Name     string
	TagCount int
	Tags     []TagData
}

// TagData represents tag information for web display
type TagData struct {
	Name   string
	Digest string
}

// StatsData represents overall statistics
type StatsData struct {
	RepositoryCount int
	TotalTags       int
	TotalSize       string
}

// handleWebIndex handles the main web interface
func (r *Router) handleWebIndex(w http.ResponseWriter, req *http.Request) {
	repositories, err := r.storage.ListRepositories()
	if err != nil {
		logrus.Errorf("Failed to list repositories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var repoData []RepositoryData
	totalTags := 0

	for _, repo := range repositories {
		tags, err := r.storage.ListTags(repo)
		if err != nil {
			logrus.Errorf("Failed to list tags for %s: %v", repo, err)
			continue
		}

		repoData = append(repoData, RepositoryData{
			Name:     repo,
			TagCount: len(tags),
		})
		totalTags += len(tags)
	}

	data := WebData{
		Title:        r.config.Web.Title,
		Repositories: repoData,
		Stats: &StatsData{
			RepositoryCount: len(repositories),
			TotalTags:       totalTags,
			TotalSize:       "N/A", // Could be calculated if needed
		},
	}

	r.renderTemplate(w, "index.html", data)
}

// handleWebRepositories handles the repositories list page
func (r *Router) handleWebRepositories(w http.ResponseWriter, req *http.Request) {
	repositories, err := r.storage.ListRepositories()
	if err != nil {
		logrus.Errorf("Failed to list repositories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var repoData []RepositoryData
	for _, repo := range repositories {
		tags, err := r.storage.ListTags(repo)
		if err != nil {
			logrus.Errorf("Failed to list tags for %s: %v", repo, err)
			continue
		}

		repoData = append(repoData, RepositoryData{
			Name:     repo,
			TagCount: len(tags),
		})
	}

	data := WebData{
		Title:        r.config.Web.Title,
		Repositories: repoData,
	}

	r.renderTemplate(w, "repositories.html", data)
}

// handleWebRepository handles individual repository page
func (r *Router) handleWebRepository(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["name"]

	tags, err := r.storage.ListTags(name)
	if err != nil {
		logrus.Errorf("Failed to list tags for %s: %v", name, err)
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var tagData []TagData
	for _, tag := range tags {
		digest, err := r.storage.GetTagDigest(name, tag)
		if err != nil {
			logrus.Errorf("Failed to get digest for %s:%s: %v", name, tag, err)
			continue
		}

		tagData = append(tagData, TagData{
			Name:   tag,
			Digest: digest,
		})
	}

	repoData := RepositoryData{
		Name:     name,
		TagCount: len(tags),
		Tags:     tagData,
	}

	data := WebData{
		Title:      r.config.Web.Title,
		Repository: &repoData,
	}

	r.renderTemplate(w, "repository.html", data)
}

// renderTemplate renders an HTML template
func (r *Router) renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	templatePath := filepath.Join("web", "templates", templateName)
	
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		logrus.Errorf("Failed to parse template %s: %v", templateName, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		logrus.Errorf("Failed to execute template %s: %v", templateName, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// API endpoints for AJAX requests

// handleAPIRepositories returns repositories as JSON
func (r *Router) handleAPIRepositories(w http.ResponseWriter, req *http.Request) {
	repositories, err := r.storage.ListRepositories()
	if err != nil {
		logrus.Errorf("Failed to list repositories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var repoData []RepositoryData
	for _, repo := range repositories {
		tags, err := r.storage.ListTags(repo)
		if err != nil {
			logrus.Errorf("Failed to list tags for %s: %v", repo, err)
			continue
		}

		repoData = append(repoData, RepositoryData{
			Name:     repo,
			TagCount: len(tags),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repoData)
}

// handleAPIStats returns statistics as JSON
func (r *Router) handleAPIStats(w http.ResponseWriter, req *http.Request) {
	repositories, err := r.storage.ListRepositories()
	if err != nil {
		logrus.Errorf("Failed to list repositories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	totalTags := 0
	for _, repo := range repositories {
		tags, err := r.storage.ListTags(repo)
		if err != nil {
			continue
		}
		totalTags += len(tags)
	}

	stats := StatsData{
		RepositoryCount: len(repositories),
		TotalTags:       totalTags,
		TotalSize:       "N/A",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

