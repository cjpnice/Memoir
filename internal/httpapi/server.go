package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"memoir/internal/app"
	"memoir/internal/config"
	"memoir/internal/media"
)

const (
	defaultMaxUploadBytes = int64(256 << 20)
	defaultMaxUploadFiles = 200
)

// ServerOptions configures HTTP boundary behavior.
type ServerOptions struct {
	AllowedOrigins []string
	MaxUploadBytes int64
	MaxUploadFiles int
	WebAssets      fs.FS
}

// Server exposes Memoir's HTTP API.
type Server struct {
	service        *app.Service
	media          *media.Manager
	router         *gin.Engine
	allowedOrigins []string
	maxUploadBytes int64
	maxUploadFiles int
	webAssets      fs.FS
}

// NewServer wires routes and middleware.
func NewServer(service *app.Service, media *media.Manager, options ServerOptions) *Server {
	gin.SetMode(gin.ReleaseMode)
	server := &Server{
		service:        service,
		media:          media,
		router:         gin.New(),
		allowedOrigins: normalizeOrigins(options.AllowedOrigins),
		maxUploadBytes: normalizeMaxUploadBytes(options.MaxUploadBytes),
		maxUploadFiles: normalizeMaxUploadFiles(options.MaxUploadFiles),
		webAssets:      options.WebAssets,
	}
	_ = server.router.SetTrustedProxies(nil)
	server.router.Use(gin.Logger(), server.recoverMiddleware(), server.corsMiddleware())
	server.routes()
	return server
}

// Handler returns the full HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() {
	s.router.GET("/healthz", s.handle(s.health))
	s.router.GET("/api/v1/projects", s.handle(s.listProjects))
	s.router.POST("/api/v1/projects", s.handle(s.createProject))
	s.router.GET("/api/v1/projects/:projectID", s.handle(s.getProject, "projectID"))
	s.router.PATCH("/api/v1/projects/:projectID", s.handle(s.updateProject, "projectID"))
	s.router.DELETE("/api/v1/projects/:projectID", s.handle(s.deleteProject, "projectID"))
	s.router.POST("/api/v1/projects/:projectID/images", s.handle(s.uploadImages, "projectID"))
	s.router.POST("/api/v1/projects/:projectID/analyze", s.handle(s.startAnalysis, "projectID"))
	s.router.POST("/api/v1/projects/:projectID/albums/generate", s.handle(s.generateAlbum, "projectID"))
	s.router.PATCH("/api/v1/projects/:projectID/albums", s.handle(s.updateAlbum, "projectID"))
	s.router.POST("/api/v1/projects/:projectID/albums/undo", s.handle(s.undoAlbum, "projectID"))
	s.router.POST("/api/v1/projects/:projectID/albums/redo", s.handle(s.redoAlbum, "projectID"))
	s.router.POST("/api/v1/projects/:projectID/exports", s.handle(s.exportAlbumArtifact, "projectID"))
	s.router.GET("/api/v1/projects/:projectID/exports/github-progress", s.handle(s.getGitHubPublishProgress, "projectID"))
	s.router.POST("/api/v1/projects/:projectID/albums/export", s.handle(s.exportAlbum, "projectID"))
	s.router.PATCH("/api/v1/images/:imageID/decision", s.handle(s.updateImageDecision, "imageID"))
	s.router.DELETE("/api/v1/images/:imageID", s.handle(s.deleteImage, "imageID"))
	s.router.POST("/api/v1/images/:imageID/process", s.handle(s.processImage, "imageID"))
	s.router.POST("/api/v1/images/:imageID/generate-edit", s.handle(s.generateImageEdit, "imageID"))
	s.router.POST("/api/v1/images/:imageID/undo-edit", s.handle(s.undoImageEdit, "imageID"))
	s.router.POST("/api/v1/images/:imageID/crop", s.handle(s.applyCrop, "imageID"))
	s.router.GET("/api/v1/settings/ai", s.handle(s.getAISettings))
	s.router.PUT("/api/v1/settings/ai", s.handle(s.updateAISettings))
	s.router.GET("/api/v1/settings/github", s.handle(s.getGitHubSettings))
	s.router.PUT("/api/v1/settings/github", s.handle(s.updateGitHubSettings))
	s.router.POST("/api/v1/settings/github/publish-listing", s.handle(s.publishAlbumListing))
	s.router.StaticFS("/media", http.Dir(s.media.Root()))
	if s.webAssets != nil {
		s.router.NoRoute(s.serveWebAsset)
	}
}

func (s *Server) handle(handler http.HandlerFunc, pathParams ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, name := range pathParams {
			c.Request.SetPathValue(name, c.Param(name))
		}
		handler(c.Writer, c.Request)
	}
}

func (s *Server) serveWebAsset(c *gin.Context) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		writeError(c.Writer, http.StatusNotFound, errors.New("not found"))
		return
	}
	requestPath := c.Request.URL.Path
	if strings.HasPrefix(requestPath, "/api/") || strings.HasPrefix(requestPath, "/media/") {
		writeError(c.Writer, http.StatusNotFound, errors.New("not found"))
		return
	}

	assetPath := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(requestPath, "/")), "/")
	if assetPath == "." || assetPath == "" {
		assetPath = "index.html"
	}
	if info, err := fs.Stat(s.webAssets, assetPath); err == nil && !info.IsDir() {
		s.serveWebAssetFile(c, assetPath, info)
		return
	}
	if info, err := fs.Stat(s.webAssets, path.Join(assetPath, "index.html")); err == nil && !info.IsDir() {
		s.serveWebAssetFile(c, path.Join(assetPath, "index.html"), info)
		return
	}
	if _, err := fs.Stat(s.webAssets, "index.html"); err != nil {
		writeError(c.Writer, http.StatusInternalServerError, errors.New("embedded web UI is missing index.html; run the release packaging script to build web assets"))
		return
	}
	if info, err := fs.Stat(s.webAssets, "index.html"); err == nil {
		s.serveWebAssetFile(c, "index.html", info)
		return
	}
	writeError(c.Writer, http.StatusInternalServerError, errors.New("embedded web UI is missing index.html; run the release packaging script to build web assets"))
}

func (s *Server) serveWebAssetFile(c *gin.Context, name string, info fs.FileInfo) {
	data, err := fs.ReadFile(s.webAssets, name)
	if err != nil {
		writeError(c.Writer, http.StatusNotFound, err)
		return
	}
	http.ServeContent(c.Writer, c.Request, path.Base(name), info.ModTime(), bytes.NewReader(data))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.service.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var input app.CreateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	project, err := s.service.CreateProject(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.service.GetProject(r.PathValue("projectID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	var input app.UpdateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	project, err := s.service.UpdateProject(r.PathValue("projectID"), input)
	if err != nil {
		if errors.Is(err, app.ErrInvalidProjectPlace) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	if err := s.service.DeleteProject(r.PathValue("projectID")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) uploadImages(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUploadBytes)
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("上传内容超过 %s 限制，请减少单次导入数量", describeUploadLimit(s.maxUploadBytes)))
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		files = r.MultipartForm.File["images"]
	}
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("no files uploaded"))
		return
	}
	if len(files) > s.maxUploadFiles {
		writeError(w, http.StatusBadRequest, fmt.Errorf("一次最多导入 %d 张照片，请分批上传", s.maxUploadFiles))
		return
	}

	images, err := s.service.UploadImages(r.PathValue("projectID"), files)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, images)
}

func (s *Server) startAnalysis(w http.ResponseWriter, r *http.Request) {
	if err := s.service.StartAnalysis(r.Context(), r.PathValue("projectID")); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	project, err := s.service.GetProject(r.PathValue("projectID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, project)
}

func (s *Server) generateAlbum(w http.ResponseWriter, r *http.Request) {
	project, err := s.service.StartAlbumGeneration(r.PathValue("projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, project)
}

func (s *Server) updateAlbum(w http.ResponseWriter, r *http.Request) {
	var input app.UpdateAlbumInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	album, err := s.service.UpdateAlbum(r.PathValue("projectID"), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, album)
}

func (s *Server) undoAlbum(w http.ResponseWriter, r *http.Request) {
	album, err := s.service.UndoAlbum(r.PathValue("projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, album)
}

func (s *Server) redoAlbum(w http.ResponseWriter, r *http.Request) {
	album, err := s.service.RedoAlbum(r.PathValue("projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, album)
}

func (s *Server) exportAlbum(w http.ResponseWriter, r *http.Request) {
	export, err := s.service.ExportAlbumHTML(r.PathValue("projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, export)
}

func (s *Server) exportAlbumArtifact(w http.ResponseWriter, r *http.Request) {
	var body app.ExportAlbumInput
	_ = json.NewDecoder(r.Body).Decode(&body)
	export, err := s.service.ExportAlbum(r.PathValue("projectID"), body.Type)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, export)
}

func (s *Server) updateImageDecision(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	body.Decision = strings.TrimSpace(body.Decision)
	if body.Decision == "" {
		writeError(w, http.StatusBadRequest, errors.New("decision is required"))
		return
	}
	image, err := s.service.SetImageDecision(r.PathValue("imageID"), body.Decision)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, image)
}

func (s *Server) deleteImage(w http.ResponseWriter, r *http.Request) {
	if err := s.service.DeleteImage(r.PathValue("imageID")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) processImage(w http.ResponseWriter, r *http.Request) {
	var input app.ProcessImageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	image, err := s.service.ProcessImage(r.PathValue("imageID"), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, image)
}

func (s *Server) generateImageEdit(w http.ResponseWriter, r *http.Request) {
	var input app.GenerativeImageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	image, err := s.service.GenerateImageEdit(r.Context(), r.PathValue("imageID"), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, image)
}

func (s *Server) undoImageEdit(w http.ResponseWriter, r *http.Request) {
	image, err := s.service.UndoImageEdit(r.PathValue("imageID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, image)
}

func (s *Server) applyCrop(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CropID string `json:"cropId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	image, err := s.service.ApplyCrop(r.PathValue("imageID"), body.CropID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, image)
}

func (s *Server) getAISettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.service.GetAISettings())
}

func (s *Server) updateAISettings(w http.ResponseWriter, r *http.Request) {
	var settings app.AISettingsInput
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.service.UpdateAISettings(settings.ToConfig()); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.service.GetAISettings())
}

func (s *Server) getGitHubSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.service.GetGitHubSettings())
}

func (s *Server) getGitHubPublishProgress(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if strings.TrimSpace(projectID) == "" {
		writeError(w, http.StatusBadRequest, errors.New("projectID is required"))
		return
	}
	writeJSON(w, http.StatusOK, s.service.GetPublishProgress(projectID))
}

func (s *Server) publishAlbumListing(w http.ResponseWriter, r *http.Request) {
	if err := s.service.PublishAlbumListing(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) updateGitHubSettings(w http.ResponseWriter, r *http.Request) {
	var settings config.GitHubSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.service.UpdateGitHubSettings(settings); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.service.GetGitHubSettings())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}

func normalizeOrigins(origins []string) []string {
	out := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		out = append(out, strings.TrimRight(origin, "/"))
	}
	return out
}

func originAllowed(origin string, allowedOrigins []string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func normalizeMaxUploadBytes(limit int64) int64 {
	if limit <= 0 {
		return defaultMaxUploadBytes
	}
	return limit
}

func normalizeMaxUploadFiles(limit int) int {
	if limit <= 0 {
		return defaultMaxUploadFiles
	}
	return limit
}

func describeUploadLimit(size int64) string {
	if size <= 0 {
		return "unlimited"
	}
	switch {
	case size%(1<<20) == 0:
		return fmt.Sprintf("%d MB", size>>(20))
	case size%(1<<10) == 0:
		return fmt.Sprintf("%d KB", size>>(10))
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			switch {
			case len(s.allowedOrigins) == 0 || originAllowed(origin, s.allowedOrigins):
				if len(s.allowedOrigins) == 0 {
					c.Header("Access-Control-Allow-Origin", "*")
				} else {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Vary", "Origin")
				}
			default:
				writeError(c.Writer, http.StatusForbidden, errors.New("origin not allowed"))
				c.Abort()
				return
			}
		} else if len(s.allowedOrigins) == 0 {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) recoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic: %v", recovered)
				if !c.Writer.Written() {
					writeError(c.Writer, http.StatusInternalServerError, errors.New("internal server error"))
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}
