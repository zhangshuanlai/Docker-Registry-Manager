package api

import (
	"docker-registry-manager/internal/config"
	"docker-registry-manager/internal/storage"
	"net/http"

	"docker-registry-manager/web"
	"io/fs"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Router handles HTTP routing for the registry
type Router struct {
	config  *config.Config
	storage storage.Storage
	router  *mux.Router
}

// NewRouter creates a new router instance
func NewRouter(cfg *config.Config, storage storage.Storage) *mux.Router {
	r := &Router{
		config:  cfg,
		storage: storage,
		router:  mux.NewRouter(),
	}

	r.setupRoutes()
	return r.router
}

// 添加认证中间件
func (r *Router) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !r.config.Auth.Enabled {
			next.ServeHTTP(w, req)
			return
		}

		username, password, ok := req.BasicAuth()
		if !ok || username != r.config.Auth.Username || password != r.config.Auth.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Docker Registry Manager"`)
			r.writeError(w, http.StatusUnauthorized, ErrorCodeUnauthorized, "Unauthorized access")
			return
		}

		next.ServeHTTP(w, req)
	})
}

// 修改setupRoutes方法，为上传路由添加认证中间件
func (r *Router) setupRoutes() {
	// Docker Registry API v2 routes
	v2 := r.router.PathPrefix("/v2").Subrouter()

	// Base endpoint - returns 200 OK to indicate v2 support
	v2.HandleFunc("/", r.handleV2Base).Methods("GET")

	// Manifest routes
	v2.HandleFunc("/{name:.+}/manifests/{reference}", r.handleManifestGet).Methods("GET")
	v2.HandleFunc("/{name:.+}/manifests/{reference}", r.handleManifestPut).Methods("PUT")
	v2.HandleFunc("/{name:.+}/manifests/{reference}", r.handleManifestHead).Methods("HEAD")
	v2.HandleFunc("/{name:.+}/manifests/{reference}", r.handleManifestDelete).Methods("DELETE")

	// Blob routes
	v2.HandleFunc("/{name:.+}/blobs/{digest}", r.handleBlobGet).Methods("GET")
	v2.HandleFunc("/{name:.+}/blobs/{digest}", r.handleBlobHead).Methods("HEAD")
	v2.HandleFunc("/{name:.+}/blobs/{digest}", r.handleBlobDelete).Methods("DELETE")

	// Blob upload routes - 添加认证保护
	uploadRouter := v2.PathPrefix("/{name:.+}/blobs/uploads/").Subrouter()
	uploadRouter.Use(r.authMiddleware) // 应用认证中间件
	uploadRouter.HandleFunc("/", r.handleBlobUploadPost).Methods("POST")
	uploadRouter.HandleFunc("/{uuid}", r.handleBlobUploadPatch).Methods("PATCH")
	uploadRouter.HandleFunc("/{uuid}", r.handleBlobUploadPut).Methods("PUT")
	uploadRouter.HandleFunc("/{uuid}", r.handleBlobUploadGet).Methods("GET")
	uploadRouter.HandleFunc("/{uuid}", r.handleBlobUploadDelete).Methods("DELETE")

	// Catalog and tags routes
	v2.HandleFunc("/_catalog", r.handleCatalog).Methods("GET")
	v2.HandleFunc("/{name:.+}/tags/list", r.handleTagsList).Methods("GET")

	// Web interface routes (if enabled)
	if r.config.Web.Enabled {
		r.router.HandleFunc("/", r.handleWebIndex).Methods("GET")
		r.router.HandleFunc("/repositories", r.handleWebRepositories).Methods("GET")
		r.router.HandleFunc("/repositories/{name:.+}", r.handleWebRepository).Methods("GET")

		// API endpoints for AJAX
		api := r.router.PathPrefix("/api").Subrouter()
		api.HandleFunc("/repositories", r.handleAPIRepositories).Methods("GET")
		api.HandleFunc("/stats", r.handleAPIStats).Methods("GET")
		api.HandleFunc("/login", r.handleLogin).Methods("POST")
		api.HandleFunc("/logout", r.handleLogout).Methods("POST")

		// Repository description API endpoints
		api.HandleFunc("/repositories/{name}/description", r.handleGetRepositoryDescription).Methods("GET")
		api.HandleFunc("/repositories/{name}/description", r.handlePutRepositoryDescription).Methods("PUT")

		// Static files
		// 静态文件服务 - 使用嵌入的文件系统
		staticFS, err := fs.Sub(web.EmbeddedAssets, "static")
		if err != nil {
			logrus.Fatalf("Failed to create static filesystem: %v", err)
		}
		r.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
			http.FileServer(http.FS(staticFS))))

		// 登录页面
		r.router.HandleFunc("/login", r.handleWebLogin).Methods("GET")
	}

	// Add logging middleware
	r.router.Use(r.loggingMiddleware)
}

func (r *Router) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logrus.WithFields(logrus.Fields{
			"method": req.Method,
			"path":   req.URL.Path,
			"remote": req.RemoteAddr,
		}).Info("HTTP request")
		next.ServeHTTP(w, req)
	})
}
