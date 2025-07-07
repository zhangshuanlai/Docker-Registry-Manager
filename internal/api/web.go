package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"docker-registry-manager/web"
)

// WebData represents data passed to web templates
type WebData struct {
	Title                 string
	Repositories          []RepositoryData
	Repository            *RepositoryData
	Stats                 *StatsData
	IsLoggedIn            bool
	Username              string
	RepositoryDescription string
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

	totalSize, err := r.storage.GetTotalStorageSize()
	if err != nil {
		logrus.Errorf("Failed to get total storage size: %v", err)
	}

	// Convert bytes to MB
	totalSizeMB := float64(totalSize) / (1024 * 1024)
	formattedSize := fmt.Sprintf("%.2f MB", totalSizeMB)

	data := WebData{
		Title:        r.config.Web.Title,
		Repositories: repoData,
		Stats: &StatsData{
			RepositoryCount: len(repositories),
			TotalTags:       totalTags,
			TotalSize:       formattedSize,
		},
		IsLoggedIn: r.isLoggedIn(req),
		Username:   r.config.Auth.Username,
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
		IsLoggedIn:   r.isLoggedIn(req),
		Username:     r.config.Auth.Username,
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

	// 获取仓库说明
	desc, err := r.storage.GetRepositoryDescription(name)
	if err != nil {
		logrus.Errorf("Failed to get repository description for %s: %v", name, err)
		// Non-critical error, proceed without description
	}

	data := WebData{
		Title:                 r.config.Web.Title,
		Repository:            &repoData,
		IsLoggedIn:            r.isLoggedIn(req),
		Username:              r.config.Auth.Username,
		RepositoryDescription: desc,
	}

	r.renderTemplate(w, "repository.html", data)
}

// renderTemplate renders an HTML template
func (r *Router) renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	var tmpl *template.Template
	var err error

	if r.config.Web.Enabled {
		// If web is enabled, try to parse embedded assets
		tmpl, err = template.ParseFS(web.EmbeddedAssets, "templates/"+templateName)
		if err != nil {
			logrus.Errorf("Failed to parse embedded template %s: %v", templateName, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to ParseFiles if web is not enabled (should not happen in this context)
		templatePath := filepath.Join("web", "templates", templateName)
		tmpl, err = template.ParseFiles(templatePath)
		if err != nil {
			logrus.Errorf("Failed to parse file template %s: %v", templateName, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
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
		r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to list repositories")
		return
	}

	var totalTags int
	for _, repo := range repositories {
		tags, err := r.storage.ListTags(repo)
		if err != nil {
			logrus.Errorf("Failed to list tags for %s: %v", repo, err)
			continue
		}
		totalTags += len(tags)
	}

	totalSize, err := r.storage.GetTotalStorageSize()
	if err != nil {
		logrus.Errorf("Failed to get total storage size: %v", err)
	}

	// Convert bytes to MB
	totalSizeMB := float64(totalSize) / (1024 * 1024)
	formattedSize := fmt.Sprintf("%.2f MB", totalSizeMB)

	stats := StatsData{
		RepositoryCount: len(repositories),
		TotalTags:       totalTags,
		TotalSize:       formattedSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// 在 Router 结构体后添加辅助方法
func (r *Router) isLoggedIn(req *http.Request) bool {
	cookie, err := req.Cookie("login")
	return err == nil && cookie.Value == "1"
}

// 登录处理函数
func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	type loginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var lr loginRequest
	err := json.NewDecoder(req.Body).Decode(&lr)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if lr.Username == r.config.Auth.Username && lr.Password == r.config.Auth.Password {
		// 登录成功，设置 Cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "login",
			Value:    "1",
			Path:     "/",
			HttpOnly: true,
		})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
		return
	}
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// 登录页面渲染
func (r *Router) handleWebLogin(w http.ResponseWriter, req *http.Request) {
	data := WebData{
		Title: r.config.Web.Title,
	}
	r.renderTemplate(w, "login.html", data)
}

// handleLogout clears the login cookie
func (r *Router) handleLogout(w http.ResponseWriter, req *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "login",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Set expiry to past to delete
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}

// handleGetRepositoryDescription handles fetching a repository's description
func (r *Router) handleGetRepositoryDescription(w http.ResponseWriter, req *http.Request) {
	if !r.isLoggedIn(req) {
		r.writeError(w, http.StatusUnauthorized, ErrorCodeUnauthorized, "Unauthorized access")
		return
	}

	vars := mux.Vars(req)
	name := vars["name"]

	description, err := r.storage.GetRepositoryDescription(name)
	if err != nil {
		logrus.Errorf("Failed to get repository description for %s: %v", name, err)
		r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to get description")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"description": description})
}

// handlePutRepositoryDescription handles updating a repository's description
func (r *Router) handlePutRepositoryDescription(w http.ResponseWriter, req *http.Request) {
	if !r.isLoggedIn(req) {
		r.writeError(w, http.StatusUnauthorized, ErrorCodeUnauthorized, "Unauthorized access")
		return
	}

	vars := mux.Vars(req)
	name := vars["name"]

	body, err := io.ReadAll(req.Body)
	if err != nil {
		logrus.Errorf("Failed to read request body: %v", err)
		r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to read request body")
		return
	}

	description := string(body)

	if err := r.storage.PutRepositoryDescription(name, description); err != nil {
		logrus.Errorf("Failed to put repository description for %s: %v", name, err)
		r.writeError(w, http.StatusInternalServerError, ErrorCodeUnknown, "Failed to save description")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}
