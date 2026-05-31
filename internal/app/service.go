package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"image/draw"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	xdraw "golang.org/x/image/draw"
	"memoir/internal/ai"
	"memoir/internal/config"
	"memoir/internal/domain"
	"memoir/internal/id"
	"memoir/internal/media"
	"memoir/internal/store"
)

// Service orchestrates project, media, and AI workflows.
type Service struct {
	store              store.ProjectStore
	media              *media.Manager
	analyzer           ai.Analyzer
	analyzerFactory    func(config.Config) (ai.Analyzer, error)
	settingsPath       string
	githubSettingsPath string
	cfg                config.Config

	publishProgress   map[string]GitHubPublishProgress
	publishProgressMu sync.Mutex
}

// GitHubPublishProgress tracks live progress of a GitHub Pages publish.
type GitHubPublishProgress struct {
	ProjectID string `json:"projectId"`
	Active    bool   `json:"active"`
	Phase     string `json:"phase"`   // "uploading_images" | "uploading_html" | "updating_listing" | "done" | "error"
	Current   int    `json:"current"` // images uploaded so far
	Total     int    `json:"total"`   // total images to upload
	Message   string `json:"message"` // human-readable status
	Error     string `json:"error,omitempty"`
}

// ErrInvalidProjectPlace identifies invalid structured place metadata.
var ErrInvalidProjectPlace = errors.New("invalid project place")

// CreateProjectInput captures the fields needed for a new project.
type CreateProjectInput struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Location    string               `json:"location"`
	Place       *domain.ProjectPlace `json:"place,omitempty"`
	Tone        string               `json:"tone"`
	ThemeID     string               `json:"themeId"`
}

// UpdateProjectInput updates editable project properties.
type UpdateProjectInput struct {
	Title       *string              `json:"title,omitempty"`
	Description *string              `json:"description,omitempty"`
	Location    *string              `json:"location,omitempty"`
	Place       *domain.ProjectPlace `json:"place,omitempty"`
	ClearPlace  *bool                `json:"clearPlace,omitempty"`
	Tone        *string              `json:"tone,omitempty"`
	ThemeID     *string              `json:"themeId,omitempty"`
}

// UpdateAlbumInput replaces editable album metadata and page structure.
type UpdateAlbumInput struct {
	Title   *string          `json:"title,omitempty"`
	Intro   *string          `json:"intro,omitempty"`
	ThemeID *string          `json:"themeId,omitempty"`
	Pages   []AlbumPageInput `json:"pages,omitempty"`
	Reason  string           `json:"reason,omitempty"`
}

// AlbumPageInput carries a full editable page snapshot from the editor.
type AlbumPageInput struct {
	ID       string   `json:"id"`
	Order    int      `json:"order"`
	PageType string   `json:"pageType"`
	LayoutID string   `json:"layoutId"`
	ImageIDs []string `json:"imageIds"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Caption  string   `json:"caption"`
}

// ExportAlbumInput chooses the export artifact type.
type ExportAlbumInput struct {
	Type string `json:"type"`
}

// ProcessImageInput describes one non-destructive image edit.
type ProcessImageInput struct {
	Operation   string `json:"operation"`
	Angle       int    `json:"angle,omitempty"`
	Brightness  int    `json:"brightness,omitempty"`
	Contrast    int    `json:"contrast,omitempty"`
	Saturation  int    `json:"saturation,omitempty"`
	Warmth      int    `json:"warmth,omitempty"`
	ExpandRatio int    `json:"expandRatio,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
}

// GenerativeImageInput asks the configured image model to produce a retouched derivative.
type GenerativeImageInput struct {
	Prompt string `json:"prompt"`
}

// AISettingsInput carries user-facing AI provider settings.
type AISettingsInput struct {
	BaseURL      string `json:"baseUrl"`
	APIKey       string `json:"apiKey"`
	Model        string `json:"model"`
	ImageBaseURL string `json:"imageBaseUrl"`
	ImageAPIKey  string `json:"imageApiKey"`
	ImageModel   string `json:"imageModel"`
	Enabled      bool   `json:"enabled"`
}

// ToConfig converts input into config settings.
func (i AISettingsInput) ToConfig() config.AISettings {
	apiKey := strings.TrimSpace(i.APIKey)
	if !i.Enabled {
		apiKey = ""
	}
	return config.AISettings{
		BaseURL:      strings.TrimSpace(i.BaseURL),
		APIKey:       apiKey,
		Model:        strings.TrimSpace(i.Model),
		ImageBaseURL: strings.TrimSpace(i.ImageBaseURL),
		ImageAPIKey:  strings.TrimSpace(i.ImageAPIKey),
		ImageModel:   strings.TrimSpace(i.ImageModel),
		Enabled:      i.Enabled && apiKey != "",
	}
}

// NewService builds the application service.
func NewService(
	store store.ProjectStore,
	media *media.Manager,
	analyzer ai.Analyzer,
	cfg config.Config,
	analyzerFactory func(config.Config) (ai.Analyzer, error),
	settingsPath string,
	githubSettingsPath string,
) *Service {
	if strings.TrimSpace(cfg.InternalBaseURL) == "" {
		cfg.InternalBaseURL = "http://127.0.0.1:8090"
	}
	return &Service{
		store:              store,
		media:              media,
		analyzer:           analyzer,
		analyzerFactory:    analyzerFactory,
		settingsPath:       settingsPath,
		githubSettingsPath: githubSettingsPath,
		cfg:                cfg,
		publishProgress:    make(map[string]GitHubPublishProgress),
	}
}

func startProjectTask(project *domain.Project, taskType domain.TaskType, message string) {
	if project == nil {
		return
	}
	if project.ActiveTask != nil {
		project.TaskHistory = append(project.TaskHistory, cloneTask(project.ActiveTask))
		project.TaskHistory = trimTaskHistory(project.TaskHistory, 20)
	}
	now := time.Now()
	project.ActiveTask = &domain.ProjectTask{
		ID:        id.New("task"),
		Type:      taskType,
		Status:    domain.TaskStatusRunning,
		Progress:  0,
		Message:   message,
		StartedAt: now,
		UpdatedAt: now,
	}
}

func updateProjectTask(project *domain.Project, progress int, message string) {
	if project == nil || project.ActiveTask == nil {
		return
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	project.ActiveTask.Status = domain.TaskStatusRunning
	project.ActiveTask.Progress = progress
	if message != "" {
		project.ActiveTask.Message = message
	}
	project.ActiveTask.UpdatedAt = time.Now()
}

func completeProjectTask(project *domain.Project, message string) {
	if project == nil || project.ActiveTask == nil {
		return
	}
	project.ActiveTask.Status = domain.TaskStatusCompleted
	project.ActiveTask.Progress = 100
	if message != "" {
		project.ActiveTask.Message = message
	}
	project.ActiveTask.UpdatedAt = time.Now()
	project.ActiveTask.CompletedAt = project.ActiveTask.UpdatedAt
}

func failProjectTask(project *domain.Project, err error) {
	if project == nil || project.ActiveTask == nil {
		return
	}
	project.ActiveTask.Status = domain.TaskStatusFailed
	if project.ActiveTask.Progress <= 0 {
		project.ActiveTask.Progress = 1
	}
	project.ActiveTask.Error = err.Error()
	project.ActiveTask.UpdatedAt = time.Now()
	project.ActiveTask.CompletedAt = project.ActiveTask.UpdatedAt
}

func cloneTask(task *domain.ProjectTask) *domain.ProjectTask {
	if task == nil {
		return nil
	}
	raw, _ := json.Marshal(task)
	var out domain.ProjectTask
	_ = json.Unmarshal(raw, &out)
	return &out
}

func trimTaskHistory(tasks []*domain.ProjectTask, limit int) []*domain.ProjectTask {
	if limit <= 0 || len(tasks) <= limit {
		return tasks
	}
	start := len(tasks) - limit
	out := make([]*domain.ProjectTask, 0, limit)
	for _, task := range tasks[start:] {
		out = append(out, cloneTask(task))
	}
	return out
}

// CreateProject inserts a new project with default state.
func (s *Service) CreateProject(input CreateProjectInput) (*domain.Project, error) {
	place, err := normalizeProjectPlace(input.Place)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	project := &domain.Project{
		ID:               id.New("prj"),
		Title:            strings.TrimSpace(input.Title),
		Description:      strings.TrimSpace(input.Description),
		Location:         strings.TrimSpace(input.Location),
		Place:            place,
		Tone:             strings.TrimSpace(input.Tone),
		ThemeID:          defaultIfEmpty(strings.TrimSpace(input.ThemeID), "film_travel"),
		Status:           domain.ProjectStatusDraft,
		AnalysisStatus:   "idle",
		AnalysisProgress: 0,
		CurrentStep:      "等待导入",
		CreatedAt:        now,
		UpdatedAt:        now,
		Images:           []*domain.ImageAsset{},
		TaskHistory:      []*domain.ProjectTask{},
	}
	if project.Title == "" {
		project.Title = "未命名相册"
	}
	if err := s.store.CreateProject(project); err != nil {
		return nil, err
	}
	return s.annotateProject(project), nil
}

// ListProjects returns all projects.
func (s *Service) ListProjects() ([]*domain.Project, error) {
	projects, err := s.store.ListProjects()
	if err != nil {
		return nil, err
	}
	for _, project := range projects {
		s.annotateProject(project)
	}
	return projects, nil
}

// GetProject returns a single project by id.
func (s *Service) GetProject(id string) (*domain.Project, error) {
	project, err := s.store.GetProject(id)
	if err != nil {
		return nil, err
	}
	return s.annotateProject(project), nil
}

// GetAISettings returns the current runtime AI settings.
func (s *Service) GetAISettings() config.AISettings {
	return s.cfg.AISettings()
}

func (s *Service) currentAnalyzerVersions() ai.AnalyzerVersions {
	if s.analyzer == nil {
		return ai.AnalyzerVersions{}
	}
	return s.analyzer.Versions()
}

func (s *Service) annotateProject(project *domain.Project) *domain.Project {
	if project == nil {
		return nil
	}
	versions := s.currentAnalyzerVersions()
	project.CurrentAnalysisModelVersion = versions.AnalysisModelVersion
	project.CurrentAnalysisPromptVersion = versions.AnalysisPromptVersion
	pending, stale := analysisCounts(project.Images, versions)
	project.PendingAnalysisCount = pending
	project.StaleAnalysisCount = stale
	return project
}

// UpdateAISettings persists settings and rebuilds the analyzer when needed.
func (s *Service) UpdateAISettings(settings config.AISettings) error {
	nextCfg := s.cfg.ApplyAISettings(settings)
	if s.analyzerFactory != nil {
		analyzer, err := s.analyzerFactory(nextCfg)
		if err != nil {
			return err
		}
		s.analyzer = analyzer
	}
	s.cfg = nextCfg
	if s.settingsPath != "" {
		if err := config.SaveAISettings(s.settingsPath, nextCfg.AISettings()); err != nil {
			return err
		}
	}
	return nil
}

// GetGitHubSettings returns the current runtime GitHub Pages settings.
func (s *Service) GetGitHubSettings() config.GitHubSettings {
	return s.cfg.GitHubSettings()
}

// UpdateGitHubSettings persists GitHub Pages settings.
func (s *Service) UpdateGitHubSettings(settings config.GitHubSettings) error {
	nextCfg := s.cfg.ApplyGitHubSettings(settings)
	s.cfg = nextCfg
	if s.githubSettingsPath != "" {
		if err := config.SaveGitHubSettings(s.githubSettingsPath, nextCfg.GitHubSettings()); err != nil {
			return err
		}
	}

	// If settings are complete, proactively upload the album listing page
	gh := nextCfg.GitHubSettings()
	if gh.Owner != "" && gh.Repo != "" && gh.Token != "" {
		go s.PublishAlbumListing()
	}

	return nil
}

// PublishAlbumListing regenerates and uploads the root album listing page.
// This can be called standalone to refresh the landing page.
func (s *Service) PublishAlbumListing() error {
	gh := s.cfg.GitHubSettings()
	if gh.Owner == "" || gh.Repo == "" || gh.Token == "" {
		return errors.New("GitHub 设置不完整")
	}
	client := newGitHubClient(gh.Owner, gh.Repo, gh.Branch, gh.Token)
	return client.publishAlbumListing()
}

// DeleteProject removes a project.
func (s *Service) DeleteProject(id string) error {
	return s.store.DeleteProject(id)
}

// DeleteImage removes a photo and its media artifacts.
func (s *Service) DeleteImage(imageID string) error {
	project, _, err := s.store.FindImage(imageID)
	if err != nil {
		return err
	}

	deleted, err := s.store.DeleteImage(imageID)
	if err != nil {
		return err
	}
	_ = s.media.DeleteRelativeURL(deleted.OriginalURL)
	_ = s.media.DeleteRelativeURL(deleted.ThumbnailURL)
	_ = s.media.DeleteRelativeURL(deleted.DerivedURL)

	if project.Album != nil {
		_ = s.store.UpdateProject(project.ID, func(project *domain.Project) error {
			if project.Album == nil {
				return nil
			}
			project.Album.Pages = removeEmptyAlbumPages(project.Album.Pages)
			if len(project.Album.Pages) == 0 {
				project.Album = nil
				project.Status = domain.ProjectStatusReviewing
				project.CurrentStep = "等待筛选"
			}
			return nil
		})
	}
	return nil
}

// UpdateProject mutates the editable metadata.
func (s *Service) UpdateProject(id string, input UpdateProjectInput) (*domain.Project, error) {
	err := s.store.UpdateProject(id, func(project *domain.Project) error {
		if input.Title != nil {
			if title := strings.TrimSpace(*input.Title); title != "" {
				project.Title = title
			}
		}
		if input.Description != nil {
			project.Description = strings.TrimSpace(*input.Description)
		}
		if input.Location != nil {
			project.Location = strings.TrimSpace(*input.Location)
		}
		if input.ClearPlace != nil && *input.ClearPlace {
			project.Place = nil
		}
		if input.Place != nil {
			place, err := normalizeProjectPlace(input.Place)
			if err != nil {
				return err
			}
			project.Place = place
		}
		if input.Tone != nil {
			if tone := strings.TrimSpace(*input.Tone); tone != "" {
				project.Tone = tone
			}
		}
		if input.ThemeID != nil {
			if themeID := strings.TrimSpace(*input.ThemeID); themeID != "" {
				project.ThemeID = themeID
				if project.Album != nil {
					project.Album.ThemeID = project.ThemeID
					project.Album.UpdatedAt = time.Now()
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetProject(id)
}

func normalizeProjectPlace(place *domain.ProjectPlace) (*domain.ProjectPlace, error) {
	if place == nil {
		return nil, nil
	}
	out := *place
	out.City = strings.TrimSpace(out.City)
	out.Region = strings.TrimSpace(out.Region)
	out.Country = strings.TrimSpace(out.Country)
	out.Source = strings.TrimSpace(out.Source)
	if out.Country == "" {
		out.Country = "中国"
	}
	if out.City == "" {
		return nil, fmt.Errorf("%w: city is required", ErrInvalidProjectPlace)
	}
	if out.Latitude < -90 || out.Latitude > 90 {
		return nil, fmt.Errorf("%w: latitude must be between -90 and 90", ErrInvalidProjectPlace)
	}
	if out.Longitude < -180 || out.Longitude > 180 {
		return nil, fmt.Errorf("%w: longitude must be between -180 and 180", ErrInvalidProjectPlace)
	}
	switch out.Source {
	case "city_catalog", "manual":
	default:
		return nil, fmt.Errorf("%w: source must be city_catalog or manual", ErrInvalidProjectPlace)
	}
	if out.Confidence < 0 {
		out.Confidence = 0
	}
	if out.Confidence > 1 {
		out.Confidence = 1
	}
	return &out, nil
}

// UploadImages stores uploaded photos and registers them in the project.
func (s *Service) UploadImages(projectID string, files []*multipart.FileHeader) ([]*domain.ImageAsset, error) {
	if _, err := s.store.GetProject(projectID); err != nil {
		return nil, err
	}
	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		project.Status = domain.ProjectStatusUploading
		project.CurrentStep = "导入图片"
		project.AnalysisStatus = "idle"
		project.LastError = ""
		return nil
	}); err != nil {
		return nil, err
	}

	items := make([]*domain.ImageAsset, 0, len(files))
	for _, header := range files {
		imageID := id.New("img")
		src, err := header.Open()
		if err != nil {
			s.markUploadFailed(projectID, err)
			return nil, err
		}
		saved, err := s.media.SaveOriginal(projectID, imageID, header.Filename, src)
		_ = src.Close()
		if err != nil {
			s.markUploadFailed(projectID, err)
			return nil, err
		}

		image := &domain.ImageAsset{
			ID:           imageID,
			ProjectID:    projectID,
			FileName:     header.Filename,
			MimeType:     saved.MimeType,
			FileSize:     saved.FileSize,
			Width:        saved.Width,
			Height:       saved.Height,
			OriginalURL:  saved.OriginalURL,
			ThumbnailURL: saved.ThumbnailURL,
			Status:       domain.ImageStatusUploaded,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Analysis: &domain.ImageAnalysis{
				Metrics: saved.Metrics,
			},
		}
		if err := s.store.AddImage(projectID, image); err != nil {
			s.markUploadFailed(projectID, err)
			return nil, err
		}
		items = append(items, image)
	}

	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		project.Status = domain.ProjectStatusReviewing
		project.CurrentStep = "等待筛选"
		return nil
	}); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Service) markUploadFailed(projectID string, uploadErr error) {
	_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
		if len(project.Images) == 0 {
			project.Status = domain.ProjectStatusDraft
			project.CurrentStep = "等待导入"
		} else {
			project.Status = domain.ProjectStatusReviewing
			project.CurrentStep = "导入失败，可继续重试"
		}
		project.LastError = uploadErr.Error()
		return nil
	})
}

// StartAnalysis launches background analysis for images that have not completed AI analysis.
func (s *Service) StartAnalysis(ctx context.Context, projectID string) error {
	if s.analyzer == nil {
		return errors.New("AI 分析功能需要配置 API Key，请在设置中配置")
	}
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}
	versions := s.currentAnalyzerVersions()
	pending := pendingAnalysisImages(project.Images, versions)
	if len(pending) == 0 {
		if len(project.Images) == 0 {
			return errors.New("no images uploaded")
		}
		if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
			project.Status = domain.ProjectStatusReviewing
			project.AnalysisStatus = "done"
			project.CurrentStep = "没有新增照片需要分析"
			project.AnalysisProgress = 100
			project.AnalysisModelVersion = versions.AnalysisModelVersion
			project.AnalysisPromptVersion = versions.AnalysisPromptVersion
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		project.Status = domain.ProjectStatusAnalyzing
		project.AnalysisStatus = "running"
		project.AnalysisProgress = 0
		stepMessage := fmt.Sprintf("准备分析 %d 张新增照片", len(pending))
		if stale := countStaleAnalysisImages(project.Images, versions); stale > 0 {
			stepMessage = fmt.Sprintf("准备分析 %d 张需重新筛选的照片", len(pending))
		}
		project.CurrentStep = stepMessage
		project.LastError = ""
		startProjectTask(project, domain.TaskTypeAnalysis, stepMessage)
		for _, image := range project.Images {
			if needsAnalysis(image, versions) {
				image.Status = domain.ImageStatusAnalyzing
			}
		}
		return nil
	}); err != nil {
		return err
	}

	go s.analyzeProject(context.Background(), projectID)
	return nil
}

// SetImageDecision stores a manual keep/exclude decision.
func (s *Service) SetImageDecision(imageID, decision string) (*domain.ImageAsset, error) {
	image, err := s.store.UpdateImage(imageID, func(image *domain.ImageAsset) error {
		image.UserDecision = decision
		switch decision {
		case "keep":
			image.Status = domain.ImageStatusApproved
		case "exclude":
			image.Status = domain.ImageStatusExcluded
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return image, nil
}

// ApplyCrop creates a derived image from the selected crop suggestion.
func (s *Service) ApplyCrop(imageID string, cropID string) (*domain.ImageAsset, error) {
	project, image, err := s.store.FindImage(imageID)
	if err != nil {
		return nil, err
	}
	if image.Analysis == nil || len(image.Analysis.CropSuggestions) == 0 {
		return nil, errors.New("no crop suggestions available")
	}

	var selected *domain.CropSuggestion
	for _, crop := range image.Analysis.CropSuggestions {
		if crop.ID == cropID {
			selected = &crop
			break
		}
	}
	if selected == nil {
		selected = &image.Analysis.CropSuggestions[0]
	}

	source, _, err := s.media.LoadImageFromURL(image.OriginalURL)
	if err != nil {
		return nil, err
	}
	derivedURL, err := s.media.SaveDerived(project.ID, image.ID, source, selected.Box)
	if err != nil {
		return nil, err
	}

	updated, err := s.store.UpdateImage(imageID, func(img *domain.ImageAsset) error {
		pushImageHistory(img, "crop")
		img.DerivedURL = derivedURL
		img.Status = domain.ImageStatusImproveThenKeep
		img.UserDecision = "crop_applied"
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// ProcessImage creates a non-destructive derivative for local image edits.
func (s *Service) ProcessImage(imageID string, input ProcessImageInput) (*domain.ImageAsset, error) {
	project, asset, err := s.store.FindImage(imageID)
	if err != nil {
		return nil, err
	}
	sourceURL := asset.DerivedURL
	if sourceURL == "" {
		sourceURL = asset.OriginalURL
	}
	source, _, err := s.media.LoadImageFromURL(sourceURL)
	if err != nil {
		return nil, err
	}

	operation := strings.TrimSpace(strings.ToLower(input.Operation))
	var processed image.Image
	suffix := operation
	switch operation {
	case "rotate":
		angle := input.Angle
		if angle == 0 {
			angle = 90
		}
		processed = media.Rotate(source, angle)
		suffix = fmt.Sprintf("rotate_%d", ((angle%360)+360)%360)
	case "color":
		processed = media.Adjust(source, media.ImageAdjustments{
			Brightness: input.Brightness,
			Contrast:   input.Contrast,
			Saturation: input.Saturation,
			Warmth:     input.Warmth,
		})
		suffix = "color"
	case "expand":
		processed = expandCanvas(source, input.ExpandRatio)
		suffix = "expand"
	case "cleanup":
		if asset.Analysis == nil || len(asset.Analysis.CropSuggestions) == 0 {
			return nil, errors.New("当前照片暂无可用于去干扰物的裁剪建议，请先完成 AI 分析或使用裁剪建议")
		}
		box := asset.Analysis.CropSuggestions[0].Box
		processed = cropImage(source, box)
		suffix = "cleanup_crop"
	default:
		return nil, fmt.Errorf("unsupported image operation: %s", input.Operation)
	}

	derivedURL, width, height, err := s.media.SaveVariant(project.ID, asset.ID, uniqueSuffix(suffix), processed)
	if err != nil {
		return nil, err
	}
	updated, err := s.store.UpdateImage(imageID, func(img *domain.ImageAsset) error {
		pushImageHistory(img, operation)
		img.DerivedURL = derivedURL
		img.Width = width
		img.Height = height
		if operation == "cleanup" {
			img.UserDecision = "cleanup_applied"
		} else {
			img.UserDecision = operation + "_applied"
		}
		if img.Status == domain.ImageStatusUploaded || img.Status == domain.ImageStatusReview {
			img.Status = domain.ImageStatusImproveThenKeep
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// GenerateImageEdit creates a model-backed retouched derivative and keeps the original for comparison.
func (s *Service) GenerateImageEdit(ctx context.Context, imageID string, input GenerativeImageInput) (*domain.ImageAsset, error) {
	project, asset, err := s.store.FindImage(imageID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.cfg.OpenAIImageAPIKey) == "" {
		return nil, errors.New("生成式图片优化需要先在 AI 设置里配置图像优化 API Key")
	}
	sourceURL := asset.DerivedURL
	if sourceURL == "" {
		sourceURL = asset.OriginalURL
	}
	sourcePath, err := s.media.StoragePathFromURL(sourceURL)
	if err != nil {
		return nil, err
	}
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		prompt = buildGenerativeEditPrompt(asset)
	}
	raw, err := s.requestImageEdit(ctx, sourcePath, prompt)
	if err != nil {
		return nil, err
	}
	derivedURL, width, height, err := s.media.SaveJPEGBytes(project.ID, asset.ID, uniqueSuffix("ai_edit"), raw)
	if err != nil {
		return nil, err
	}
	updated, err := s.store.UpdateImage(imageID, func(img *domain.ImageAsset) error {
		pushImageHistory(img, "generative_edit")
		img.DerivedURL = derivedURL
		img.Width = width
		img.Height = height
		img.UserDecision = "generative_edit_applied"
		if img.Status == domain.ImageStatusUploaded || img.Status == domain.ImageStatusReview || img.Status == domain.ImageStatusRejectSuggested {
			img.Status = domain.ImageStatusImproveThenKeep
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// UndoImageEdit restores the previous image derivative state.
func (s *Service) UndoImageEdit(imageID string) (*domain.ImageAsset, error) {
	updated, err := s.store.UpdateImage(imageID, func(img *domain.ImageAsset) error {
		if img == nil || len(img.EditHistory) == 0 {
			return errors.New("没有可撤销的图片优化")
		}
		last := img.EditHistory[len(img.EditHistory)-1]
		img.EditHistory = img.EditHistory[:len(img.EditHistory)-1]
		img.DerivedURL = last.DerivedURL
		img.Width = last.Width
		img.Height = last.Height
		img.Status = last.Status
		img.UserDecision = last.UserDecision
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) requestImageEdit(ctx context.Context, sourcePath string, prompt string) ([]byte, error) {
	if isDashScopeImageProvider(s.cfg.OpenAIImageBaseURL, s.cfg.OpenAIImageModel) {
		return s.requestDashScopeImageEdit(ctx, sourcePath, prompt)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", defaultIfEmpty(strings.TrimSpace(s.cfg.OpenAIImageModel), "gpt-image-1.5")); err != nil {
		return nil, err
	}
	if err := writer.WriteField("prompt", prompt); err != nil {
		return nil, err
	}
	if err := writer.WriteField("size", "1024x1024"); err != nil {
		return nil, err
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	part, err := writer.CreateFormFile("image", filepath.Base(sourcePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(s.cfg.OpenAIImageBaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/images/edits", &body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.cfg.OpenAIImageAPIKey))
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("生成式图片优化失败: %s", describeProviderError(payload, response.StatusCode))
	}
	var decoded struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Data) == 0 || strings.TrimSpace(decoded.Data[0].B64JSON) == "" {
		if len(decoded.Data) > 0 && strings.TrimSpace(decoded.Data[0].URL) != "" {
			return downloadImageEditResult(ctx, decoded.Data[0].URL)
		}
		return nil, errors.New("生成式图片优化没有返回可保存的图片数据")
	}
	raw, err := base64.StdEncoding.DecodeString(decoded.Data[0].B64JSON)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) requestDashScopeImageEdit(ctx context.Context, sourcePath string, prompt string) ([]byte, error) {
	imageRaw, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	imageDataURL := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imageRaw)
	body := map[string]any{
		"model": defaultIfEmpty(strings.TrimSpace(s.cfg.OpenAIImageModel), "qwen-image-edit"),
		"input": map[string]any{
			"messages": []map[string]any{
				{
					"role": "user",
					"content": []map[string]any{
						{"image": imageDataURL},
						{"text": prompt},
					},
				},
			},
		},
		"parameters": map[string]any{
			"n": 1,
		},
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(s.cfg.OpenAIImageBaseURL), "/")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/api/v1"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, dashScopeImageEditEndpoint(baseURL), bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.cfg.OpenAIImageAPIKey))
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("生成式图片优化失败: %s", describeProviderError(payload, response.StatusCode))
	}
	var decoded struct {
		Output struct {
			Choices []struct {
				Message struct {
					Content []struct {
						Image string `json:"image"`
						Text  string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	for _, choice := range decoded.Output.Choices {
		for _, content := range choice.Message.Content {
			if imageURL := strings.TrimSpace(content.Image); imageURL != "" {
				return downloadImageEditResult(ctx, imageURL)
			}
		}
	}
	return nil, errors.New("生成式图片优化没有返回可保存的图片数据")
}

func isDashScopeImageProvider(baseURL string, model string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	if strings.Contains(parsed.Host, "dashscope.aliyuncs.com") && strings.Contains(parsed.Path, "/api/v1") {
		return true
	}
	return strings.Contains(parsed.Path, "/api/v1") && strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "qwen-image")
}

func dashScopeImageEditEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/services/aigc/multimodal-generation/generation") {
		return baseURL
	}
	return baseURL + "/services/aigc/multimodal-generation/generation"
}

func downloadImageEditResult(ctx context.Context, imageURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(imageURL), nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("下载生成式图片优化结果失败: %s", describeProviderError(payload, response.StatusCode))
	}
	return payload, nil
}

func buildGenerativeEditPrompt(image *domain.ImageAsset) string {
	base := "在不改变人物身份、地点事实和照片真实感的前提下，优化这张照片用于长期相册保存。"
	if image == nil || image.Analysis == nil || len(image.Analysis.EditSuggestions) == 0 {
		return base + " 请自然改善构图、光线、色彩和轻微干扰物，保持纪实照片质感。"
	}
	parts := make([]string, 0, len(image.Analysis.EditSuggestions)+1)
	parts = append(parts, base)
	for _, suggestion := range image.Analysis.EditSuggestions {
		if suggestion.ProviderBacked || suggestion.Execution == "provider_generative" || suggestion.Type == "cleanup" || suggestion.Type == "expand" {
			parts = append(parts, suggestion.Reason)
		}
	}
	if len(parts) == 1 {
		parts = append(parts, "请自然改善构图、光线、色彩和轻微干扰物，保持纪实照片质感。")
	}
	return strings.Join(parts, " ")
}

func pushImageHistory(image *domain.ImageAsset, reason string) {
	if image == nil {
		return
	}
	image.EditHistory = append(image.EditHistory, &domain.ImageSnapshot{
		DerivedURL:   image.DerivedURL,
		Width:        image.Width,
		Height:       image.Height,
		Status:       image.Status,
		UserDecision: image.UserDecision,
		Reason:       strings.TrimSpace(reason),
		CreatedAt:    time.Now(),
	})
	if len(image.EditHistory) > 20 {
		image.EditHistory = image.EditHistory[len(image.EditHistory)-20:]
	}
}

func describeProviderError(payload []byte, status int) string {
	var decoded struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &decoded); err == nil && decoded.Error.Message != "" {
		return decoded.Error.Message
	}
	raw := strings.TrimSpace(string(payload))
	if raw == "" {
		return fmt.Sprintf("HTTP %d", status)
	}
	return raw
}

// StartAlbumGeneration begins async album generation and returns immediately.
func (s *Service) StartAlbumGeneration(projectID string) (*domain.Project, error) {
	if s.analyzer == nil {
		return nil, errors.New("AI 相册生成功能需要配置 API Key，请在设置中配置")
	}
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	selected := selectAlbumImages(project.Images)
	if len(selected) == 0 {
		selected = project.Images
	}
	if len(selected) == 0 {
		return nil, errors.New("no images available for album")
	}

	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		project.Status = domain.ProjectStatusGeneratingAlbum
		project.CurrentStep = fmt.Sprintf("生成相册草稿（%d 张照片）", len(selected))
		project.AnalysisStatus = "done"
		project.AnalysisProgress = 100
		project.LastError = ""
		startProjectTask(project, domain.TaskTypeAlbumGeneration, fmt.Sprintf("准备生成相册草稿（%d 张照片）", len(selected)))
		return nil
	}); err != nil {
		return nil, err
	}

	go s.generateAlbumInBackground(context.Background(), projectID)

	updated, err := s.store.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) generateAlbumInBackground(ctx context.Context, projectID string) {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return
	}

	selected := selectAlbumImages(project.Images)
	if len(selected) == 0 {
		selected = project.Images
	}

	narrative, err := s.analyzer.WriteAlbumNarrative(ctx, ai.AlbumNarrativeInput{
		Project:       *project,
		Images:        selected,
		ThemeID:       project.ThemeID,
		SelectedCount: len(selected),
	})
	if err != nil {
		_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
			project.Status = domain.ProjectStatusFailed
			project.CurrentStep = "相册生成失败"
			project.LastError = err.Error()
			failProjectTask(project, err)
			return nil
		})
		return
	}

	album := &domain.Album{
		ID:            id.New("alb"),
		ProjectID:     projectID,
		ThemeID:       project.ThemeID,
		Title:         defaultIfEmpty(narrative.Title, project.Title),
		Intro:         defaultIfEmpty(narrative.Intro, "把这一段时间整理成一本可以反复翻看的相册。"),
		DesignNotes:   narrative.DesignNotes,
		Status:        domain.AlbumStatusGenerated,
		Version:       1,
		ModelVersion:  narrative.ModelVersion,
		PromptVersion: narrative.PromptVersion,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	album.Pages = buildAlbumPages(selected, narrative)
	album.SocialPosts = buildSocialPosts(selected, narrative)
	album.EditHistory = nil
	album.RedoStack = nil

	_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
		project.Status = domain.ProjectStatusEditing
		project.CurrentStep = "相册已生成"
		project.AnalysisStatus = "done"
		project.AnalysisProgress = 100
		project.Album = album
		completeProjectTask(project, fmt.Sprintf("相册已生成，共 %d 页", len(album.Pages)))
		return nil
	})
}

// UpdateAlbum applies editor changes and stores an undo checkpoint.
func (s *Service) UpdateAlbum(projectID string, input UpdateAlbumInput) (*domain.Album, error) {
	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		if project.Album == nil {
			return errors.New("album has not been generated")
		}
		pushAlbumHistory(project.Album, input.Reason)
		if input.Title != nil {
			project.Album.Title = strings.TrimSpace(*input.Title)
			if project.Album.Title == "" {
				project.Album.Title = "未命名相册"
			}
		}
		if input.Intro != nil {
			project.Album.Intro = strings.TrimSpace(*input.Intro)
		}
		if input.ThemeID != nil && strings.TrimSpace(*input.ThemeID) != "" {
			project.Album.ThemeID = strings.TrimSpace(*input.ThemeID)
			project.ThemeID = project.Album.ThemeID
		}
		if input.Pages != nil {
			project.Album.Pages = normalizeAlbumPages(input.Pages)
		}
		project.Album.Version++
		project.Album.Status = domain.AlbumStatusEdited
		project.Album.RedoStack = nil
		project.Album.UpdatedAt = time.Now()
		project.Status = domain.ProjectStatusEditing
		project.CurrentStep = "相册编辑中"
		return nil
	}); err != nil {
		return nil, err
	}
	updated, err := s.store.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	return updated.Album, nil
}

// UndoAlbum restores the previous editor checkpoint.
func (s *Service) UndoAlbum(projectID string) (*domain.Album, error) {
	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		if project.Album == nil {
			return errors.New("album has not been generated")
		}
		if len(project.Album.EditHistory) == 0 {
			return errors.New("no undo history")
		}
		current := snapshotAlbum(project.Album, "redo")
		last := project.Album.EditHistory[len(project.Album.EditHistory)-1]
		project.Album.EditHistory = project.Album.EditHistory[:len(project.Album.EditHistory)-1]
		project.Album.RedoStack = append(project.Album.RedoStack, current)
		restoreAlbumSnapshot(project.Album, last)
		project.Album.UpdatedAt = time.Now()
		project.CurrentStep = "已撤销上一步编辑"
		project.Status = domain.ProjectStatusEditing
		return nil
	}); err != nil {
		return nil, err
	}
	updated, err := s.store.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	return updated.Album, nil
}

// RedoAlbum reapplies the last undone editor checkpoint.
func (s *Service) RedoAlbum(projectID string) (*domain.Album, error) {
	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		if project.Album == nil {
			return errors.New("album has not been generated")
		}
		if len(project.Album.RedoStack) == 0 {
			return errors.New("no redo history")
		}
		current := snapshotAlbum(project.Album, "undo")
		next := project.Album.RedoStack[len(project.Album.RedoStack)-1]
		project.Album.RedoStack = project.Album.RedoStack[:len(project.Album.RedoStack)-1]
		project.Album.EditHistory = append(project.Album.EditHistory, current)
		restoreAlbumSnapshot(project.Album, next)
		project.Album.UpdatedAt = time.Now()
		project.CurrentStep = "已重做上一步编辑"
		project.Status = domain.ProjectStatusEditing
		return nil
	}); err != nil {
		return nil, err
	}
	updated, err := s.store.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	return updated.Album, nil
}

// ExportAlbum exports the current album in a chosen format.
func (s *Service) ExportAlbum(projectID string, exportType string) (*domain.AlbumExport, error) {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	if project.Album == nil {
		return nil, errors.New("album has not been generated")
	}

	exportType = strings.TrimSpace(strings.ToLower(exportType))
	if exportType == "" {
		exportType = "html"
	}
	exportTypeName := exportTypeDisplayName(exportType)

	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		project.Status = domain.ProjectStatusExporting
		project.CurrentStep = fmt.Sprintf("正在生成%s", exportTypeName)
		project.LastError = ""
		startProjectTask(project, domain.TaskTypeExport, fmt.Sprintf("准备生成%s", exportTypeName))
		return nil
	}); err != nil {
		return nil, err
	}

	export := &domain.AlbumExport{
		ID:        id.New("exp"),
		AlbumID:   project.Album.ID,
		ProjectID: projectID,
		Type:      exportType,
		CreatedAt: time.Now(),
	}

	switch exportType {
	case "html":
		rel := filepath.Join("exports", projectID, export.ID+".html")
		abs := filepath.Join(s.media.Root(), rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(abs, []byte(renderAlbumHTML(project, nil)), 0o644); err != nil {
			return nil, err
		}
		export.URL = s.media.PublicURL(rel)
		export.Message = "HTML 相册可直接打开"
	case "long_image":
		url, msg, err := s.exportAlbumLongImage(project, projectID, export.ID)
		if err != nil {
			_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
				project.Status = domain.ProjectStatusFailed
				project.CurrentStep = "图片导出失败"
				project.LastError = err.Error()
				failProjectTask(project, err)
				return nil
			})
			return nil, err
		}
		export.URL = url
		export.Message = msg
	case "share_link":
		url, msg, err := s.exportAlbumShare(project, projectID, export.ID)
		if err != nil {
			_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
				project.Status = domain.ProjectStatusFailed
				project.CurrentStep = "分享链接导出失败"
				project.LastError = err.Error()
				failProjectTask(project, err)
				return nil
			})
			return nil, err
		}
		export.URL = url
		export.Message = msg
	case "github_pages":
		url, msg, err := s.exportAlbumGitHubPages(project, projectID, export.ID)
		if err != nil {
			_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
				project.Status = domain.ProjectStatusFailed
				project.CurrentStep = "GitHub Pages 发布失败"
				project.LastError = err.Error()
				failProjectTask(project, err)
				return nil
			})
			return nil, err
		}
		export.URL = url
		export.Message = msg
	default:
		err := fmt.Errorf("unsupported export type: %s", exportType)
		_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
			project.Status = domain.ProjectStatusFailed
			project.CurrentStep = "导出类型不支持"
			project.LastError = err.Error()
			failProjectTask(project, err)
			return nil
		})
		return nil, err
	}

	if err := s.store.UpdateProject(projectID, func(project *domain.Project) error {
		project.Status = domain.ProjectStatusDone
		project.CurrentStep = fmt.Sprintf("%s已导出", exportTypeName)
		if project.Album != nil {
			project.Album.Status = domain.AlbumStatusExported
			project.Album.UpdatedAt = time.Now()
		}
		// Persist the export record (keep last 50)
		project.Exports = append(project.Exports, export)
		if len(project.Exports) > 50 {
			project.Exports = project.Exports[len(project.Exports)-50:]
		}
		completeProjectTask(project, fmt.Sprintf("%s已导出", exportTypeName))
		return nil
	}); err != nil {
		return nil, err
	}

	return export, nil
}

// ExportAlbumHTML keeps the old HTML export API for compatibility.
func (s *Service) ExportAlbumHTML(projectID string) (*domain.AlbumExport, error) {
	return s.ExportAlbum(projectID, "html")
}

func (s *Service) exportAlbumLongImage(project *domain.Project, projectID, exportID string) (string, string, error) {
	htmlRel := filepath.Join("exports", projectID, exportID+"_image_source.html")
	htmlAbs := filepath.Join(s.media.Root(), htmlRel)
	if err := os.MkdirAll(filepath.Dir(htmlAbs), 0o755); err != nil {
		return "", "", err
	}
	pageHeight := albumImageExportPageHeight(project)
	if err := os.WriteFile(htmlAbs, []byte(renderAlbumImageExportHTML(project, s.httpAlbumImageURL, pageHeight)), 0o644); err != nil {
		return "", "", err
	}

	imageRel := filepath.Join("exports", projectID, exportID+"_image.png")
	imageAbs := filepath.Join(s.media.Root(), imageRel)
	imageAbs, err := filepath.Abs(imageAbs)
	if err != nil {
		return "", "", err
	}
	sourceURL := s.absoluteMediaURL(s.media.PublicURL(htmlRel))
	chrome, err := findChromeExecutable()
	if err != nil {
		return s.media.PublicURL(htmlRel), "未找到 Chrome/Chromium，已生成同款 HTML，可用浏览器打开后保存为图片。", nil
	}
	cmd := exec.Command(
		chrome,
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--hide-scrollbars",
		fmt.Sprintf("--window-size=1440,%d", maxInt(pageHeight*albumPageCount(project), pageHeight)),
		"--screenshot="+imageAbs,
		sourceURL,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return s.media.PublicURL(htmlRel), "图片渲染失败，已保留同款 HTML：" + strings.TrimSpace(string(output)), nil
	}
	if _, err := os.Stat(imageAbs); err != nil {
		return "", "", err
	}
	return s.media.PublicURL(imageRel), "图片已按 HTML 相册样式生成", nil
}

func (s *Service) exportAlbumShare(project *domain.Project, projectID, exportID string) (string, string, error) {
	rel := filepath.Join("shares", projectID, exportID+".html")
	abs := filepath.Join(s.media.Root(), rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(abs, []byte(renderAlbumHTML(project, nil)), 0o644); err != nil {
		return "", "", err
	}
	return s.media.PublicURL(rel), "分享链接已生成，可在同一服务地址下打开", nil
}

func (s *Service) GetPublishProgress(projectID string) GitHubPublishProgress {
	s.publishProgressMu.Lock()
	defer s.publishProgressMu.Unlock()
	if p, ok := s.publishProgress[projectID]; ok {
		return p
	}
	return GitHubPublishProgress{ProjectID: projectID}
}

func (s *Service) setPublishProgress(p GitHubPublishProgress) {
	s.publishProgressMu.Lock()
	defer s.publishProgressMu.Unlock()
	s.publishProgress[p.ProjectID] = p
}

func (s *Service) exportAlbumGitHubPages(project *domain.Project, projectID, exportID string) (string, string, error) {
	gh := s.cfg.GitHubSettings()
	if gh.Owner == "" || gh.Repo == "" || gh.Token == "" {
		return "", "", errors.New("请先在设置中配置 GitHub 仓库信息（所有者、仓库名和 Token）")
	}

	progress := GitHubPublishProgress{
		ProjectID: projectID,
		Active:    true,
		Phase:     "preparing",
		Message:   "正在准备发布...",
	}
	s.setPublishProgress(progress)
	defer func() {
		progress.Active = false
		s.setPublishProgress(progress)
	}()

	client := newGitHubClient(gh.Owner, gh.Repo, gh.Branch, gh.Token)
	slug := slugify(project.Album.Title)
	if slug == "album" {
		slug = slug + "-" + projectID[:8]
	}
	albumPath := "albums/" + slug

	// Collect all image IDs used in album pages
	imageIDs := collectAlbumImageIDs(project)
	progress.Total = len(imageIDs)
	progress.Phase = "uploading_images"

	// Upload each image
	uploaded := 0
	for _, imgID := range imageIDs {
		image := findImageByID(project, imgID)
		if image == nil {
			continue
		}
		imgPath := imageFilePath(image)
		if imgPath == "" {
			continue
		}
		progress.Current = uploaded
		progress.Message = fmt.Sprintf("正在上传图片 %d/%d: %s", uploaded+1, progress.Total, image.FileName)
		s.setPublishProgress(progress)

		rel := strings.TrimPrefix(imgPath, "/media/")
		absPath := filepath.Join(s.media.Root(), rel)
		data, err := os.ReadFile(absPath)
		if err != nil {
			progress.Phase = "error"
			progress.Error = fmt.Sprintf("读取图片 %s 失败: %v", image.FileName, err)
			progress.Message = progress.Error
			s.setPublishProgress(progress)
			return "", "", fmt.Errorf("读取图片 %s 失败: %w", image.FileName, err)
		}
		destPath := albumPath + "/images/" + filepath.Base(imgPath)
		msg := fmt.Sprintf("Upload album image %d/%d: %s", uploaded+1, progress.Total, image.FileName)
		if err := client.putFile(destPath, msg, data); err != nil {
			progress.Phase = "error"
			progress.Error = fmt.Sprintf("上传图片 %s 失败: %v", image.FileName, err)
			progress.Message = progress.Error
			s.setPublishProgress(progress)
			return "", "", fmt.Errorf("上传图片 %s 失败: %w", image.FileName, err)
		}
		uploaded++
	}
	progress.Current = uploaded
	progress.Message = fmt.Sprintf("图片上传完成，共 %d 张", uploaded)
	s.setPublishProgress(progress)

	// Generate HTML with relative image paths pointing to images/ subdirectory
	resolveImageURL := func(image *domain.ImageAsset) string {
		imgPath := imageFilePath(image)
		if imgPath == "" {
			return ""
		}
		return "images/" + filepath.Base(imgPath)
	}
	albumHTML := renderAlbumHTML(project, resolveImageURL)

	// Upload album HTML
	progress.Phase = "uploading_html"
	progress.Message = "正在上传相册页面..."
	s.setPublishProgress(progress)

	htmlPath := albumPath + "/index.html"
	if err := client.putFile(htmlPath, "Publish album: "+project.Album.Title, []byte(albumHTML)); err != nil {
		progress.Phase = "error"
		progress.Error = fmt.Sprintf("上传相册页面失败: %v", err)
		progress.Message = progress.Error
		s.setPublishProgress(progress)
		return "", "", fmt.Errorf("上传相册页面失败: %w", err)
	}

	// Update root index.html (album listing)
	progress.Phase = "updating_listing"
	progress.Message = "正在更新相册列表..."
	s.setPublishProgress(progress)

	if err := client.publishAlbumListing(); err != nil {
		// Non-fatal: album is published but listing is stale
		_ = err
	}

	progress.Phase = "done"
	progress.Message = "发布完成"
	s.setPublishProgress(progress)

	publishedURL := fmt.Sprintf("https://%s.github.io/%s/%s/", gh.Owner, gh.Repo, albumPath)
	return publishedURL, "相册已发布到 GitHub Pages", nil
}

func collectAlbumImageIDs(project *domain.Project) []string {
	if project.Album == nil {
		return nil
	}
	seen := make(map[string]bool)
	var ids []string
	for _, page := range project.Album.Pages {
		for _, imgID := range page.ImageIDs {
			if !seen[imgID] {
				seen[imgID] = true
				ids = append(ids, imgID)
			}
		}
	}
	return ids
}

func findImageByID(project *domain.Project, imageID string) *domain.ImageAsset {
	for _, img := range project.Images {
		if img.ID == imageID {
			return img
		}
	}
	return nil
}

func imageFilePath(image *domain.ImageAsset) string {
	if image == nil {
		return ""
	}
	if strings.TrimSpace(image.DerivedURL) != "" {
		return image.DerivedURL
	}
	return image.OriginalURL
}

func (s *Service) httpAlbumImageURL(image *domain.ImageAsset) string {
	publicURL := imageDisplayURL(image)
	if publicURL == "" {
		return ""
	}
	return s.absoluteMediaURL(publicURL)
}

func (s *Service) absoluteMediaURL(publicURL string) string {
	value := strings.TrimSpace(publicURL)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	baseURL := strings.TrimRight(strings.TrimSpace(s.cfg.InternalBaseURL), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8090"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return baseURL + value
}

func exportTypeDisplayName(exportType string) string {
	switch strings.TrimSpace(strings.ToLower(exportType)) {
	case "html":
		return "HTML"
	case "long_image":
		return "images"
	case "share_link":
		return "分享链接"
	case "github_pages":
		return "GitHub Pages"
	default:
		return "导出结果"
	}
}

func (s *Service) analyzeProject(ctx context.Context, projectID string) {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
			project.Status = domain.ProjectStatusFailed
			project.AnalysisStatus = "failed"
			project.LastError = err.Error()
			return nil
		})
		return
	}

	versions := s.currentAnalyzerVersions()
	pending := pendingAnalysisImages(project.Images, versions)
	if len(pending) == 0 {
		_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
			project.Status = domain.ProjectStatusReviewing
			project.AnalysisStatus = "done"
			project.AnalysisProgress = 100
			project.CurrentStep = "分析完成"
			project.AnalysisModelVersion = versions.AnalysisModelVersion
			project.AnalysisPromptVersion = versions.AnalysisPromptVersion
			return nil
		})
		return
	}

	workers := minInt(runtime.NumCPU(), 4)
	if workers < 1 {
		workers = 1
	}
	type result struct {
		index    int
		imageID  string
		analysis *domain.ImageAnalysis
		err      error
	}
	jobs := make(chan struct {
		index int
		image *domain.ImageAsset
	})
	results := make(chan result, len(pending))
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				saved := *job.image
				imageBase64, imageMIMEType, err := s.media.EncodeForModel(job.image.OriginalURL, 1280, 86)
				if err != nil {
					results <- result{index: job.index, imageID: job.image.ID, err: fmt.Errorf("读取图片供大模型分析失败: %w", err)}
					continue
				}
				analysis, err := s.analyzer.AnalyzeImage(ctx, ai.AnalyzeImageInput{
					Project:       *project,
					Image:         saved,
					Metrics:       ensureMetrics(saved),
					ImageBase64:   imageBase64,
					ImageMIMEType: imageMIMEType,
					Index:         job.index + 1,
					Total:         len(pending),
				})
				if err != nil {
					results <- result{index: job.index, imageID: job.image.ID, err: err}
					continue
				}
				analysis.CompletedAt = time.Now()
				if analysis.ModelVersion == "" {
					analysis.ModelVersion = versions.AnalysisModelVersion
				}
				if analysis.PromptVersion == "" {
					analysis.PromptVersion = versions.AnalysisPromptVersion
				}
				results <- result{index: job.index, imageID: job.image.ID, analysis: analysis}
			}
		}()
	}

	go func() {
		for index, img := range pending {
			jobs <- struct {
				index int
				image *domain.ImageAsset
			}{index: index, image: img}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	completed := 0
	for item := range results {
		if item.err != nil {
			_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
				project.Status = domain.ProjectStatusFailed
				project.AnalysisStatus = "failed"
				project.LastError = item.err.Error()
				project.CurrentStep = "AI 分析失败"
				failProjectTask(project, item.err)
				return nil
			})
			return
		}
		_, _ = s.store.UpdateImage(item.imageID, func(image *domain.ImageAsset) error {
			previousDecision := image.UserDecision
			image.Analysis = item.analysis
			image.Status = statusFromRecommendation(item.analysis.Recommendation)
			if isManualImageDecision(previousDecision) {
				switch previousDecision {
				case "keep":
					image.Status = domain.ImageStatusApproved
				case "exclude":
					image.Status = domain.ImageStatusExcluded
				}
				image.UserDecision = previousDecision
			} else if image.Status == domain.ImageStatusKeep || image.Status == domain.ImageStatusImproveThenKeep {
				image.UserDecision = "ai_preselected"
			}
			return nil
		})
		completed++
		progress := int(float64(completed) / float64(len(pending)) * 100)
		_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
			project.CurrentStep = fmt.Sprintf("分析 %d / %d", completed, len(pending))
			project.AnalysisProgress = progress
			updateProjectTask(project, progress, project.CurrentStep)
			return nil
		})
	}

	if updated, err := s.store.GetProject(projectID); err == nil {
		_ = s.organizeReviewSet(ctx, updated)
	}

	_ = s.store.UpdateProject(projectID, func(project *domain.Project) error {
		project.Status = domain.ProjectStatusReviewing
		project.AnalysisStatus = "done"
		project.AnalysisProgress = 100
		project.CurrentStep = "AI 筛选与相似组整理完成"
		project.AnalysisModelVersion = versions.AnalysisModelVersion
		project.AnalysisPromptVersion = versions.AnalysisPromptVersion
		completeProjectTask(project, "AI 筛选与相似组整理完成")
		return nil
	})
}

func (s *Service) organizeReviewSet(ctx context.Context, project *domain.Project) error {
	if project == nil || len(project.Images) == 0 {
		return nil
	}
	const maxReviewBatchSize = 40
	batches := batchImagesForReview(project.Images, maxReviewBatchSize)
	var merged ai.ReviewOrganization
	for _, batch := range batches {
		review, err := s.analyzer.OrganizeReview(ctx, ai.ReviewOrganizationInput{
			Project: *project,
			Images:  batch,
		})
		if err != nil {
			return err
		}
		if review == nil {
			continue
		}
		merged.SelectionStrategy = review.SelectionStrategy
		merged.Groups = append(merged.Groups, review.Groups...)
	}
	if len(merged.Groups) == 0 {
		return nil
	}
	return s.applyReviewOrganization(project.ID, &merged)
}

func (s *Service) applyReviewOrganization(projectID string, review *ai.ReviewOrganization) error {
	return s.store.UpdateProject(projectID, func(project *domain.Project) error {
		imageByID := make(map[string]*domain.ImageAsset, len(project.Images))
		for _, image := range project.Images {
			imageByID[image.ID] = image
		}

		for gi, decision := range review.Groups {
			groupID := strings.TrimSpace(decision.ID)
			if groupID == "" {
				groupID = fmt.Sprintf("grp_%d", gi+1)
			}
			imageIDs := decision.ImageIDs
			if len(imageIDs) == 0 {
				continue
			}
			bestID := decision.BestImageID
			if bestID == "" {
				bestID = imageIDs[0]
			}
			keepSet := make(map[string]bool, len(decision.KeepImageIDs))
			for _, id := range decision.KeepImageIDs {
				keepSet[id] = true
			}
			if len(keepSet) == 0 && bestID != "" {
				keepSet[bestID] = true
			}
			rejectSet := make(map[string]bool, len(decision.RejectImageIDs))
			for _, id := range decision.RejectImageIDs {
				rejectSet[id] = true
			}
			socialID := decision.SocialImageID
			if socialID == "" {
				socialID = bestID
			}
			label := strings.TrimSpace(decision.Title)
			reason := strings.TrimSpace(decision.Reason)
			for rank, imageID := range imageIDs {
				image := imageByID[imageID]
				if image == nil {
					continue
				}
				if image.Analysis == nil {
					image.Analysis = &domain.ImageAnalysis{}
				}
				image.Analysis.SimilarGroupID = groupID
				image.Analysis.SimilarGroupLabel = label
				image.Analysis.SimilarGroupReason = reason
				image.Analysis.SimilarGroupBest = imageID == bestID
				if imageID == bestID {
					image.Analysis.SimilarGroupRank = 1
				} else if keepSet[imageID] {
					image.Analysis.SimilarGroupRank = 2 + rank
				} else {
					image.Analysis.SimilarGroupRank = 20 + rank
				}
				image.Analysis.SelectionRank = image.Analysis.SimilarGroupRank
				if imageID == socialID && image.Analysis.SocialCaption == "" && len(image.Analysis.CaptionSeeds) > 0 {
					image.Analysis.SocialCaption = image.Analysis.CaptionSeeds[0]
				}
				if isManualDecision(image.UserDecision) {
					continue
				}
				switch {
				case rejectSet[imageID]:
					image.Status = domain.ImageStatusRejectSuggested
					if image.UserDecision == "" {
						image.UserDecision = "similar_alternative"
					}
				case keepSet[imageID]:
					if image.Status != domain.ImageStatusApproved && image.Status != domain.ImageStatusExcluded {
						if image.Analysis.Recommendation == "improve_then_keep" {
							image.Status = domain.ImageStatusImproveThenKeep
						} else {
							image.Status = domain.ImageStatusKeep
						}
						if image.UserDecision == "" || image.UserDecision == "candidate" {
							image.UserDecision = "ai_preselected"
						}
					}
				default:
					if imageID != bestID && len(imageIDs) > 1 {
						image.Status = domain.ImageStatusRejectSuggested
						if image.UserDecision == "" {
							image.UserDecision = "similar_alternative"
						}
					}
				}
			}
		}

		return nil
	})
}

func ensureMetrics(image domain.ImageAsset) domain.ImageMetrics {
	if image.Analysis != nil {
		return image.Analysis.Metrics
	}
	return domain.ImageMetrics{
		AspectRatio: float64(image.Width) / maxFloat(float64(image.Height), 1),
		FileSize:    image.FileSize,
		Width:       image.Width,
		Height:      image.Height,
	}
}

func isManualDecision(decision string) bool {
	switch decision {
	case "keep", "exclude", "crop_applied", "cleanup_applied", "rotate_applied", "color_applied", "expand_applied":
		return true
	default:
		return false
	}
}

func selectAlbumImages(images []*domain.ImageAsset) []*domain.ImageAsset {
	items := make([]*domain.ImageAsset, 0, len(images))
	seenGroups := map[string]bool{}
	for _, image := range images {
		if image.Status == domain.ImageStatusApproved || image.Status == domain.ImageStatusKeep || image.Status == domain.ImageStatusImproveThenKeep {
			if image.Analysis != nil && image.Analysis.SimilarGroupID != "" && !image.Analysis.SimilarGroupBest && image.Status != domain.ImageStatusApproved {
				continue
			}
			if image.Analysis != nil && image.Analysis.SimilarGroupID != "" {
				if seenGroups[image.Analysis.SimilarGroupID] && image.Status != domain.ImageStatusApproved {
					continue
				}
				seenGroups[image.Analysis.SimilarGroupID] = true
			}
			items = append(items, image)
		}
	}
	if len(items) == 0 {
		for _, image := range images {
			if image.Analysis != nil && image.Analysis.Recommendation != "reject_suggested" {
				items = append(items, image)
			}
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Analysis != nil && items[j].Analysis != nil {
			ri, rj := items[i].Analysis.SelectionRank, items[j].Analysis.SelectionRank
			if ri > 0 && rj > 0 && ri != rj {
				return ri < rj
			}
			if items[i].Analysis.SimilarGroupBest != items[j].Analysis.SimilarGroupBest {
				return items[i].Analysis.SimilarGroupBest
			}
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items
}

func batchImagesForReview(images []*domain.ImageAsset, maxSize int) [][]*domain.ImageAsset {
	if len(images) <= maxSize {
		return [][]*domain.ImageAsset{images}
	}
	sort.Slice(images, func(i, j int) bool {
		return images[i].CreatedAt.Before(images[j].CreatedAt)
	})
	var batches [][]*domain.ImageAsset
	for i := 0; i < len(images); i += maxSize {
		end := i + maxSize
		if end > len(images) {
			end = len(images)
		}
		batches = append(batches, images[i:end])
	}
	return batches
}
func statusFromRecommendation(recommendation string) domain.ImageStatus {
	switch recommendation {
	case "keep":
		return domain.ImageStatusKeep
	case "improve_then_keep":
		return domain.ImageStatusImproveThenKeep
	case "reject_suggested":
		return domain.ImageStatusRejectSuggested
	default:
		return domain.ImageStatusReview
	}
}

func isManualImageDecision(decision string) bool {
	switch decision {
	case "keep", "exclude", "crop_applied", "cleanup_applied", "rotate_applied", "color_applied", "expand_applied":
		return true
	default:
		return false
	}
}

func needsAnalysis(image *domain.ImageAsset, versions ai.AnalyzerVersions) bool {
	if image == nil {
		return false
	}
	if image.Analysis == nil {
		return true
	}
	if image.Analysis.CompletedAt.IsZero() {
		return true
	}
	return isAnalysisStale(image.Analysis, versions)
}

func isAnalysisStale(analysis *domain.ImageAnalysis, versions ai.AnalyzerVersions) bool {
	if analysis == nil {
		return true
	}
	if versions.AnalysisModelVersion != "" && analysis.ModelVersion != "" && analysis.ModelVersion != versions.AnalysisModelVersion {
		return true
	}
	if versions.AnalysisPromptVersion != "" && analysis.PromptVersion != "" && analysis.PromptVersion != versions.AnalysisPromptVersion {
		return true
	}
	if versions.AnalysisModelVersion != "" && analysis.ModelVersion == "" {
		return true
	}
	if versions.AnalysisPromptVersion != "" && analysis.PromptVersion == "" {
		return true
	}
	return false
}

func analysisCounts(images []*domain.ImageAsset, versions ai.AnalyzerVersions) (pending int, stale int) {
	for _, image := range images {
		if image == nil {
			continue
		}
		if image.Analysis == nil || image.Analysis.CompletedAt.IsZero() {
			pending++
			continue
		}
		if isAnalysisStale(image.Analysis, versions) {
			pending++
			stale++
		}
	}
	return pending, stale
}

func pendingAnalysisImages(images []*domain.ImageAsset, versions ai.AnalyzerVersions) []*domain.ImageAsset {
	out := make([]*domain.ImageAsset, 0, len(images))
	for _, image := range images {
		if needsAnalysis(image, versions) {
			out = append(out, image)
		}
	}
	return out
}

func countStaleAnalysisImages(images []*domain.ImageAsset, versions ai.AnalyzerVersions) int {
	_, stale := analysisCounts(images, versions)
	return stale
}

func removeEmptyAlbumPages(pages []*domain.AlbumPage) []*domain.AlbumPage {
	out := make([]*domain.AlbumPage, 0, len(pages))
	for _, page := range pages {
		pageType := normalizePageType(page.PageType)
		if pageType == "cover" && len(page.ImageIDs) == 0 {
			continue
		}
		if pageType != "ending" && pageType != "pause" && len(page.ImageIDs) == 0 {
			continue
		}
		out = append(out, page)
	}
	return out
}

func normalizeAlbumPages(inputs []AlbumPageInput) []*domain.AlbumPage {
	pages := make([]*domain.AlbumPage, 0, len(inputs))
	for _, input := range inputs {
		pageType := normalizePageType(input.PageType)
		page := &domain.AlbumPage{
			ID:       input.ID,
			Order:    input.Order,
			PageType: pageType,
			LayoutID: normalizeAlbumLayout(input.LayoutID, len(input.ImageIDs), pageType),
			ImageIDs: append([]string{}, input.ImageIDs...),
			Title:    input.Title,
			Body:     input.Body,
			Caption:  input.Caption,
		}
		pages = append(pages, page)
	}
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Order == pages[j].Order {
			return pages[i].ID < pages[j].ID
		}
		return pages[i].Order < pages[j].Order
	})
	for index, page := range pages {
		page.Order = index + 1
		page.PageType = normalizePageType(page.PageType)
	}
	return pages
}

func pushAlbumHistory(album *domain.Album, reason string) {
	if album == nil {
		return
	}
	album.EditHistory = append(album.EditHistory, snapshotAlbum(album, reason))
	if len(album.EditHistory) > 20 {
		album.EditHistory = album.EditHistory[len(album.EditHistory)-20:]
	}
}

func snapshotAlbum(album *domain.Album, reason string) *domain.AlbumSnapshot {
	if album == nil {
		return nil
	}
	return &domain.AlbumSnapshot{
		Title:     album.Title,
		Intro:     album.Intro,
		ThemeID:   album.ThemeID,
		Pages:     cloneAlbumPages(album.Pages),
		Version:   album.Version,
		Reason:    reason,
		CreatedAt: time.Now(),
	}
}

func restoreAlbumSnapshot(album *domain.Album, snapshot *domain.AlbumSnapshot) {
	if album == nil || snapshot == nil {
		return
	}
	album.Title = snapshot.Title
	album.Intro = snapshot.Intro
	album.ThemeID = snapshot.ThemeID
	album.Pages = cloneAlbumPages(snapshot.Pages)
	album.Version = snapshot.Version
}

func cloneAlbumPages(pages []*domain.AlbumPage) []*domain.AlbumPage {
	out := make([]*domain.AlbumPage, 0, len(pages))
	for _, page := range pages {
		if page == nil {
			continue
		}
		cloned := *page
		cloned.ImageIDs = append([]string{}, page.ImageIDs...)
		out = append(out, &cloned)
	}
	return out
}

func buildAlbumPages(selected []*domain.ImageAsset, narrative *ai.AlbumNarrative) []*domain.AlbumPage {
	validIDs := make(map[string]*domain.ImageAsset, len(selected))
	for _, image := range selected {
		validIDs[image.ID] = image
	}
	pages := make([]*domain.AlbumPage, 0, len(narrative.Pages)+2)
	usedLayouts := map[string]bool{}
	for _, plan := range narrative.Pages {
		imageIDs := filterValidImageIDs(plan.ImageIDs, validIDs)
		pageType := normalizePageType(plan.PageType)
		if len(imageIDs) == 0 && pageType != "ending" && pageType != "pause" && len(selected) > 0 {
			imageIDs = []string{selected[0].ID}
		}
		layoutID := normalizeAlbumLayout(defaultIfEmpty(plan.LayoutID, layoutForImageCount(len(imageIDs), pageType)), len(imageIDs), pageType)
		usedLayouts[layoutID] = true
		title := plan.Title
		if title == "" {
			switch pageType {
			case "cover":
				title = narrative.Title
			case "ending":
				title = "结尾"
			default:
				title = fallbackPageTitle(pageType, layoutID, len(pages)+1)
			}
		}
		title = sanitizeAlbumPageTitle(title, pageType, layoutID, len(pages)+1)
		pages = append(pages, &domain.AlbumPage{
			ID:       id.New("page"),
			Order:    len(pages) + 1,
			PageType: pageType,
			LayoutID: layoutID,
			ImageIDs: imageIDs,
			Title:    title,
			Body:     plan.Body,
			Caption:  defaultIfEmpty(plan.Caption, title),
		})
	}
	if len(pages) == 0 {
		return nil
	}
	if !hasPageType(pages, "cover") && len(selected) > 0 {
		cover := &domain.AlbumPage{
			ID:       id.New("page"),
			Order:    1,
			PageType: "cover",
			LayoutID: "cover_full_bleed",
			ImageIDs: []string{selected[0].ID},
			Title:    narrative.Title,
			Body:     narrative.Intro,
			Caption:  narrative.Intro,
		}
		pages = append([]*domain.AlbumPage{cover}, pages...)
		usedLayouts["cover_full_bleed"] = true
	}
	if !hasPageType(pages, "ending") && len(selected) > 0 {
		pages = append(pages, &domain.AlbumPage{
			ID:       id.New("page"),
			Order:    len(pages) + 1,
			PageType: "ending",
			LayoutID: "ending_text",
			ImageIDs: []string{selected[len(selected)-1].ID},
			Title:    "结尾",
			Body:     narrative.Ending,
			Caption:  narrative.Ending,
		})
	}
	for index, page := range pages {
		page.Order = index + 1
	}
	return pages
}

func insertBeforeEnding(pages []*domain.AlbumPage, page *domain.AlbumPage) []*domain.AlbumPage {
	if page == nil {
		return pages
	}
	for index, existing := range pages {
		if existing != nil && existing.PageType == "ending" {
			out := append([]*domain.AlbumPage{}, pages[:index]...)
			out = append(out, page)
			out = append(out, pages[index:]...)
			return out
		}
	}
	return append(pages, page)
}

func buildSocialPosts(selected []*domain.ImageAsset, narrative *ai.AlbumNarrative) []*domain.AlbumSocialPost {
	validIDs := make(map[string]*domain.ImageAsset, len(selected))
	for _, image := range selected {
		validIDs[image.ID] = image
	}
	posts := make([]*domain.AlbumSocialPost, 0, len(narrative.SocialPosts))
	for _, post := range narrative.SocialPosts {
		imageIDs := filterValidImageIDs(post.ImageIDs, validIDs)
		if len(imageIDs) == 0 && len(selected) > 0 {
			for i := 0; i < minInt(len(selected), 9); i++ {
				imageIDs = append(imageIDs, selected[i].ID)
			}
		}
		if strings.TrimSpace(post.Title) == "" && strings.TrimSpace(post.Body) == "" {
			continue
		}
		platform := normalizeSocialPlatform(post.Platform, len(posts))
		body := strings.TrimSpace(post.Body)
		hook := strings.TrimSpace(post.Hook)
		if hook == "" {
			hook = ""
		}
		posts = append(posts, &domain.AlbumSocialPost{
			Platform: platform,
			Title:    strings.TrimSpace(post.Title),
			Body:     body,
			Hook:     hook,
			ImageIDs: imageIDs,
			Hashtags: normalizeSocialHashtags(post.Hashtags, platform),
		})
	}
	if len(posts) == 0 && len(selected) > 0 {
		ids := make([]string, 0, minInt(len(selected), 9))
		for i := 0; i < minInt(len(selected), 9); i++ {
			ids = append(ids, selected[i].ID)
		}
		posts = append(posts, &domain.AlbumSocialPost{
			Platform: "moments",
			Title:    narrative.Title + " · 朋友圈",
			Hook:     "把这些瞬间整理成一本相册。",
			Body:     "把这些瞬间整理成一本相册，留下的不只是照片，也是那段时间的气息。以后再翻到这里，应该还能想起当时的光、风和心情。",
			ImageIDs: ids,
			Hashtags: normalizeSocialHashtags(nil, "moments"),
		})
		posts = append(posts, &domain.AlbumSocialPost{
			Platform: "xiaohongshu",
			Title:    narrative.Title + " · 小红书",
			Hook:     "把一组照片做成了有故事线的纪念相册。",
			Body:     "这组照片没有直接堆在一起，而是按记忆点、画面关系和情绪节奏做了筛选。相册里有封面、主视觉、细节、留白和结尾，更像一本可以长期保存的私人摄影书。",
			ImageIDs: ids,
			Hashtags: normalizeSocialHashtags(nil, "xiaohongshu"),
		})
	}
	return posts
}

func normalizeSocialPlatform(platform string, index int) string {
	value := strings.ToLower(strings.TrimSpace(platform))
	switch value {
	case "moments", "wechat", "wechat_moments", "朋友圈":
		return "moments"
	case "xiaohongshu", "redbook", "red_note", "小红书":
		return "xiaohongshu"
	default:
		if index == 1 {
			return "xiaohongshu"
		}
		return "moments"
	}
}

func normalizeSocialHashtags(tags []string, platform string) []string {
	_ = platform
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !strings.HasPrefix(tag, "#") {
			tag = "#" + tag
		}
		if !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	return out
}

func filterValidImageIDs(ids []string, valid map[string]*domain.ImageAsset) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, imageID := range ids {
		imageID = strings.TrimSpace(imageID)
		if imageID == "" || valid[imageID] == nil || seen[imageID] {
			continue
		}
		seen[imageID] = true
		out = append(out, imageID)
	}
	return out
}

func hasPageType(pages []*domain.AlbumPage, pageType string) bool {
	for _, page := range pages {
		if page.PageType == pageType {
			return true
		}
	}
	return false
}

func layoutForImageCount(count int, pageType string) string {
	pageType = normalizePageType(pageType)
	if pageType == "cover" {
		return "cover_full_bleed"
	}
	if pageType == "ending" {
		return "ending_text"
	}
	if pageType == "pause" && count == 0 {
		return "quote_break"
	}
	switch {
	case count <= 1:
		return "hero_story"
	case count == 2:
		return "diptych"
	case count == 3:
		return "triptych"
	case count <= 6:
		return "mosaic"
	default:
		return "contact_sheet"
	}
}

func normalizeAlbumLayout(layoutID string, count int, pageType string) string {
	pageType = normalizePageType(pageType)
	switch layoutID {
	case "cover_full_bleed", "hero_story", "single_photo_caption", "diptych", "triptych", "mosaic", "gallery_wall", "timeline_ribbon", "full_bleed_quote", "scrapbook", "contact_sheet", "quote_break", "ending_text":
	default:
		layoutID = layoutForImageCount(count, pageType)
	}
	if pageType == "cover" {
		return "cover_full_bleed"
	}
	if pageType == "ending" {
		return "ending_text"
	}
	if (pageType == "pause" || pageType == "opener") && count == 0 {
		return "quote_break"
	}
	if count == 0 && layoutID != "quote_break" {
		return "quote_break"
	}
	if count == 1 && (layoutID == "diptych" || layoutID == "triptych" || layoutID == "mosaic" || layoutID == "gallery_wall" || layoutID == "timeline_ribbon" || layoutID == "scrapbook" || layoutID == "contact_sheet") {
		return "hero_story"
	}
	if count == 2 && layoutID == "triptych" {
		return "diptych"
	}
	return layoutID
}

func normalizePageType(pageType string) string {
	switch strings.TrimSpace(pageType) {
	case "cover":
		return "cover"
	case "opener", "opening":
		return "opener"
	case "hero", "chapter":
		return "hero"
	case "detail":
		return "detail"
	case "sequence":
		return "sequence"
	case "collage":
		return "collage"
	case "gallery":
		return "gallery"
	case "pause", "transition", "quote":
		return "pause"
	case "index":
		return "index"
	case "ending":
		return "ending"
	default:
		return pageTypeForLayout("")
	}
}

func pageTypeForLayout(layoutID string) string {
	switch layoutID {
	case "cover_full_bleed":
		return "cover"
	case "hero_story", "full_bleed_quote":
		return "hero"
	case "single_photo_caption", "diptych", "triptych":
		return "detail"
	case "timeline_ribbon":
		return "sequence"
	case "mosaic", "scrapbook":
		return "collage"
	case "gallery_wall":
		return "gallery"
	case "quote_break":
		return "pause"
	case "contact_sheet":
		return "index"
	case "ending_text":
		return "ending"
	default:
		return "detail"
	}
}

func fallbackPageTitle(pageType string, layoutID string, index int) string {
	switch normalizePageType(pageType) {
	case "opener":
		return "开场白"
	case "hero":
		return "第一眼记住的画面"
	case "detail":
		if layoutID == "diptych" || layoutID == "triptych" {
			return "并置的回声"
		}
		return "被看见的细节"
	case "sequence":
		return "时间经过这里"
	case "collage":
		return "记忆碎片"
	case "gallery":
		return "一面记忆墙"
	case "pause":
		return "留白"
	case "index":
		return "这一卷的索引"
	case "ending":
		return "结尾"
	default:
		return fmt.Sprintf("页面 %d", index)
	}
}

func sanitizeAlbumPageTitle(title string, pageType string, layoutID string, index int) string {
	trimmed := strings.TrimSpace(title)
	forbidden := []string{"无题", "章节", "第一章", "第二章", "第三章", "第四章", "第五章", "第六章", "第七章", "第八章", "第九章"}
	if trimmed == "" {
		return fallbackPageTitle(pageType, layoutID, index)
	}
	for _, token := range forbidden {
		if strings.Contains(trimmed, token) {
			return fallbackPageTitle(pageType, layoutID, index)
		}
	}
	return trimmed
}

func imageIDs(images []*domain.ImageAsset) []string {
	out := make([]string, 0, len(images))
	for _, image := range images {
		out = append(out, image.ID)
	}
	return out
}

func extractCaptions(images []*domain.ImageAsset) []string {
	out := make([]string, 0, len(images))
	for _, image := range images {
		if image.Analysis != nil && len(image.Analysis.CaptionSeeds) > 0 {
			out = append(out, image.Analysis.CaptionSeeds[0])
		}
	}
	return out
}

func layoutForImages(images []*domain.ImageAsset, pageOrder int) string {
	switch {
	case len(images) <= 1:
		if pageOrder%2 == 0 {
			return "hero_story"
		}
		return "single_photo_caption"
	case len(images) == 2:
		return "diptych"
	case len(images) == 3:
		return "triptych"
	case len(images) <= 6:
		return "mosaic"
	default:
		return "contact_sheet"
	}
}

type albumImageURLResolver func(*domain.ImageAsset) string

func renderAlbumHTML(project *domain.Project, resolveImageURL albumImageURLResolver) string {
	return renderAlbumHTMLWithExtraCSS(project, resolveImageURL, "")
}

func renderAlbumImageExportHTML(project *domain.Project, resolveImageURL albumImageURLResolver, pageHeight int) string {
	extraCSS := fmt.Sprintf(`html,body{width:1440px;overflow:hidden}.album{width:1440px;max-width:none}.page{height:%dpx;min-height:%dpx}`, pageHeight, pageHeight)
	return renderAlbumHTMLWithExtraCSS(project, resolveImageURL, extraCSS)
}

func renderAlbumHTMLWithExtraCSS(project *domain.Project, resolveImageURL albumImageURLResolver, extraCSS string) string {
	album := project.Album
	images := mapImages(project.Images)
	var b strings.Builder

	b.WriteString("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(album.Title))
	b.WriteString(" - Memoir</title><style>")
	b.WriteString(albumThemeCSS(album.ThemeID))
	if extraCSS != "" {
		b.WriteString(extraCSS)
	}
	b.WriteString("</style></head><body>")
	b.WriteString("<main class=\"album album-")
	b.WriteString(html.EscapeString(album.ThemeID))
	b.WriteString("\">")

	for _, page := range album.Pages {
		b.WriteString("<section class=\"page page-")
		b.WriteString(html.EscapeString(page.PageType))
		b.WriteString(" layout-")
		b.WriteString(html.EscapeString(page.LayoutID))
		b.WriteString("\">")

		if page.PageType == "cover" {
			writePageImages(&b, page, images, true, resolveImageURL)
			b.WriteString("<div class=\"cover-copy\"><p class=\"kicker\">Memoir / 集忆</p><h1>")
			b.WriteString(html.EscapeString(page.Title))
			b.WriteString("</h1><p>")
			b.WriteString(html.EscapeString(page.Body))
			b.WriteString("</p></div>")
		} else {
			b.WriteString("<div class=\"page-copy\"><p class=\"kicker\">")
			b.WriteString(html.EscapeString(exportPageTypeLabel(page.PageType)))
			b.WriteString("</p><h2>")
			b.WriteString(html.EscapeString(page.Title))
			b.WriteString("</h2>")
			if page.Body != "" {
				b.WriteString("<p>")
				b.WriteString(html.EscapeString(page.Body))
				b.WriteString("</p>")
			}
			b.WriteString("</div>")
			writePageImages(&b, page, images, false, resolveImageURL)
		}

		b.WriteString("</section>")
	}

	b.WriteString("</main><div class=\"lightbox\" data-open=\"false\" aria-hidden=\"true\"><button class=\"lightbox-close\" type=\"button\" aria-label=\"关闭大图\">×</button><img alt=\"\"><p></p></div><script>")
	b.WriteString(albumLightboxScript())
	b.WriteString("</script></body></html>")
	return b.String()
}

func writePageImages(b *strings.Builder, page *domain.AlbumPage, images map[string]*domain.ImageAsset, cover bool, resolveImageURL albumImageURLResolver) {
	if len(page.ImageIDs) == 0 {
		return
	}
	className := "photo-grid"
	if cover {
		className = "cover-image"
	}
	b.WriteString("<div class=\"")
	b.WriteString(className)
	b.WriteString("\">")
	for _, imageID := range page.ImageIDs {
		image := images[imageID]
		if image == nil {
			continue
		}
		src := imageDisplayURL(image)
		if resolveImageURL != nil {
			src = resolveImageURL(image)
		}
		caption := image.FileName
		if image.Analysis != nil && len(image.Analysis.CaptionSeeds) > 0 {
			caption = image.Analysis.CaptionSeeds[0]
		}
		b.WriteString("<figure><button class=\"photo-zoom\" type=\"button\" data-src=\"")
		b.WriteString(html.EscapeString(src))
		b.WriteString("\" data-caption=\"")
		b.WriteString(html.EscapeString(caption))
		b.WriteString("\" aria-label=\"放大查看 ")
		b.WriteString(html.EscapeString(image.FileName))
		b.WriteString("\"><img src=\"")
		b.WriteString(html.EscapeString(src))
		b.WriteString("\" alt=\"")
		b.WriteString(html.EscapeString(image.FileName))
		b.WriteString("\"></button><figcaption>")
		b.WriteString(html.EscapeString(caption))
		b.WriteString("</figcaption></figure>")
	}
	b.WriteString("</div>")
}

func imageDisplayURL(image *domain.ImageAsset) string {
	if image == nil {
		return ""
	}
	if strings.TrimSpace(image.DerivedURL) != "" {
		return image.DerivedURL
	}
	return image.OriginalURL
}

func albumPageCount(project *domain.Project) int {
	if project == nil || project.Album == nil || len(project.Album.Pages) == 0 {
		return 1
	}
	return len(project.Album.Pages)
}

func albumImageExportPageHeight(project *domain.Project) int {
	const pageHeight = 1080
	const maxScreenshotHeight = 30000
	count := albumPageCount(project)
	if count < 1 {
		count = 1
	}
	if pageHeight*count <= maxScreenshotHeight {
		return pageHeight
	}
	return maxInt(720, maxScreenshotHeight/count)
}

func exportPageTypeLabel(pageType string) string {
	switch normalizePageType(pageType) {
	case "cover":
		return "封面"
	case "opener":
		return "开场"
	case "hero":
		return "主视觉"
	case "detail":
		return "细节"
	case "sequence":
		return "时间带"
	case "collage":
		return "拼贴"
	case "gallery":
		return "画廊"
	case "pause":
		return "留白"
	case "index":
		return "索引"
	case "ending":
		return "结尾"
	default:
		if pageType == "" {
			return "页面"
		}
		return pageType
	}
}

func albumLightboxScript() string {
	return `
(() => {
  const box = document.querySelector('.lightbox');
  const img = box?.querySelector('img');
  const caption = box?.querySelector('p');
  const close = () => {
    if (!box || !img || !caption) return;
    box.dataset.open = 'false';
    box.setAttribute('aria-hidden', 'true');
    img.removeAttribute('src');
    caption.textContent = '';
  };
  document.querySelectorAll('.photo-zoom').forEach((button) => {
    button.addEventListener('click', () => {
      if (!box || !img || !caption) return;
      img.src = button.dataset.src || '';
      caption.textContent = button.dataset.caption || '';
      box.dataset.open = 'true';
      box.setAttribute('aria-hidden', 'false');
    });
  });
  box?.addEventListener('click', (event) => {
    if (event.target === box || event.target.closest('.lightbox-close')) close();
  });
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') close();
  });
})();`
}

func mapImages(images []*domain.ImageAsset) map[string]*domain.ImageAsset {
	out := make(map[string]*domain.ImageAsset, len(images))
	for _, image := range images {
		out[image.ID] = image
	}
	return out
}

type albumThemeTokens struct {
	bg           string
	bg2          string
	text         string
	muted        string
	accent       string
	panel        string
	frame        string
	soft         string
	coverOverlay string
	coverFilter  string
	photoFilter  string
	shadow       string
}

func albumThemeTokensFor(themeID string) albumThemeTokens {
	tokens := albumThemeTokens{
		bg:           "#f6f1e8",
		bg2:          "#f4f7f3",
		text:         "#181818",
		muted:        "rgba(18,24,24,.74)",
		accent:       "#c94c2c",
		panel:        "#fffaf3",
		frame:        "rgba(255,255,255,.88)",
		soft:         "rgba(18,24,24,.12)",
		coverOverlay: "linear-gradient(90deg,rgba(4,6,8,.76),rgba(4,6,8,.08)),linear-gradient(0deg,rgba(4,6,8,.48),transparent 45%)",
		coverFilter:  "saturate(1.04) contrast(1.02)",
		photoFilter:  "saturate(1.02) contrast(1.01)",
		shadow:       "rgba(12,18,20,.16)",
	}
	switch themeID {
	case "warm_family":
		tokens = albumThemeTokens{bg: "#fff1e9", bg2: "#f8dccc", text: "#2a1f1b", muted: "rgba(42,31,27,.74)", accent: "#d86f57", panel: "#fff8f2", frame: "rgba(255,248,242,.94)", soft: "rgba(160,85,64,.16)", coverOverlay: "linear-gradient(90deg,rgba(54,25,17,.72),rgba(54,25,17,.18)),linear-gradient(0deg,rgba(72,34,25,.52),transparent 48%)", coverFilter: "saturate(1.08) contrast(.98) sepia(.1)", photoFilter: "saturate(1.06) contrast(.98) sepia(.08)", shadow: "rgba(94,45,31,.16)"}
	case "editorial":
		tokens = albumThemeTokens{bg: "#f6f6f1", bg2: "#ffffff", text: "#101010", muted: "rgba(16,16,16,.72)", accent: "#111111", panel: "#ffffff", frame: "#ffffff", soft: "rgba(0,0,0,.16)", coverOverlay: "linear-gradient(90deg,rgba(0,0,0,.82),rgba(0,0,0,.08)),linear-gradient(0deg,rgba(0,0,0,.54),transparent 48%)", coverFilter: "grayscale(.72) contrast(1.18)", photoFilter: "grayscale(.12) contrast(1.08)", shadow: "rgba(0,0,0,.14)"}
	case "minimal_gallery":
		tokens = albumThemeTokens{bg: "#fbfbf8", bg2: "#ecefed", text: "#151717", muted: "rgba(21,23,23,.64)", accent: "#245f73", panel: "#ffffff", frame: "#fdfdfb", soft: "rgba(36,95,115,.12)", coverOverlay: "linear-gradient(90deg,rgba(10,18,19,.62),rgba(10,18,19,.02)),linear-gradient(0deg,rgba(10,18,19,.42),transparent 45%)", coverFilter: "saturate(.92) contrast(1.03)", photoFilter: "saturate(.94) contrast(1.02)", shadow: "rgba(15,25,26,.12)"}
	case "nocturne":
		tokens = albumThemeTokens{bg: "#10171f", bg2: "#1a2230", text: "#f5efe3", muted: "rgba(245,239,227,.68)", accent: "#f0b15f", panel: "#0b1017", frame: "rgba(255,255,255,.1)", soft: "rgba(240,177,95,.22)", coverOverlay: "linear-gradient(90deg,rgba(3,6,10,.82),rgba(14,24,34,.2)),linear-gradient(0deg,rgba(3,6,10,.72),transparent 55%)", coverFilter: "saturate(1.15) contrast(1.12) brightness(.84)", photoFilter: "saturate(1.08) contrast(1.08) brightness(.92)", shadow: "rgba(0,0,0,.34)"}
	case "botanical":
		tokens = albumThemeTokens{bg: "#eef4e9", bg2: "#dbe9d5", text: "#17231a", muted: "rgba(23,35,26,.72)", accent: "#4f7f52", panel: "#f7fbf2", frame: "#fbfff7", soft: "rgba(79,127,82,.17)", coverOverlay: "linear-gradient(90deg,rgba(12,35,19,.74),rgba(12,35,19,.08)),linear-gradient(0deg,rgba(12,35,19,.46),transparent 47%)", coverFilter: "saturate(1.02) contrast(.98) hue-rotate(-6deg)", photoFilter: "saturate(.98) contrast(.98) hue-rotate(-4deg)", shadow: "rgba(29,64,35,.16)"}
	case "postcard":
		tokens = albumThemeTokens{bg: "#eef7fb", bg2: "#d5ecf6", text: "#17324a", muted: "rgba(23,50,74,.72)", accent: "#e85d4f", panel: "#fffefa", frame: "#fffefa", soft: "rgba(232,93,79,.2)", coverOverlay: "linear-gradient(90deg,rgba(8,36,60,.74),rgba(8,36,60,.04)),linear-gradient(0deg,rgba(8,36,60,.48),transparent 45%)", coverFilter: "saturate(1.12) contrast(1.02)", photoFilter: "saturate(1.06) contrast(1.01)", shadow: "rgba(14,55,74,.15)"}
	case "archive":
		tokens = albumThemeTokens{bg: "#eee6d4", bg2: "#d9c9ad", text: "#251f18", muted: "rgba(37,31,24,.72)", accent: "#8c4b35", panel: "#f8f0df", frame: "#f8f0df", soft: "rgba(140,75,53,.18)", coverOverlay: "linear-gradient(90deg,rgba(38,28,18,.78),rgba(38,28,18,.14)),linear-gradient(0deg,rgba(38,28,18,.52),transparent 50%)", coverFilter: "sepia(.22) saturate(.92) contrast(1.03)", photoFilter: "sepia(.18) saturate(.9) contrast(1.02)", shadow: "rgba(52,37,21,.17)"}
	case "cinematic_bw":
		tokens = albumThemeTokens{bg: "#0e0e0f", bg2: "#232324", text: "#f3f3ef", muted: "rgba(243,243,239,.68)", accent: "#d7d7d2", panel: "#151515", frame: "rgba(255,255,255,.12)", soft: "rgba(215,215,210,.2)", coverOverlay: "linear-gradient(90deg,rgba(0,0,0,.86),rgba(0,0,0,.15)),radial-gradient(circle at 70% 32%,rgba(255,255,255,.16),transparent 26%)", coverFilter: "grayscale(1) contrast(1.24) brightness(.86)", photoFilter: "grayscale(1) contrast(1.18)", shadow: "rgba(0,0,0,.38)"}
	case "summer_diary":
		tokens = albumThemeTokens{bg: "#f2fbf8", bg2: "#d8f3ed", text: "#163235", muted: "rgba(22,50,53,.7)", accent: "#f2a33a", panel: "#fffdf0", frame: "#fffdf0", soft: "rgba(38,165,154,.18)", coverOverlay: "linear-gradient(90deg,rgba(8,45,52,.68),rgba(8,45,52,.04)),linear-gradient(0deg,rgba(8,45,52,.42),transparent 45%)", coverFilter: "saturate(1.12) contrast(.98) brightness(1.04)", photoFilter: "saturate(1.08) contrast(.98) brightness(1.02)", shadow: "rgba(19,92,88,.14)"}
	}
	return tokens
}

func albumThemeCSS(themeID string) string {
	tokens := albumThemeTokensFor(themeID)
	vars := fmt.Sprintf(`.album{--album-bg:%s;--album-bg-2:%s;--album-text:%s;--album-muted:%s;--album-accent:%s;--album-panel:%s;--album-frame:%s;--album-soft:%s;--cover-overlay:%s;--cover-filter:%s;--photo-filter:%s;--album-shadow:%s;}`,
		tokens.bg, tokens.bg2, tokens.text, tokens.muted, tokens.accent, tokens.panel, tokens.frame, tokens.soft, tokens.coverOverlay, tokens.coverFilter, tokens.photoFilter, tokens.shadow)
	return vars + `
*{box-sizing:border-box}body{margin:0;background:#0c0f11;color:#181818;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}.album{max-width:1440px;margin:0 auto;background:var(--album-bg);color:var(--album-text)}.page{min-height:100vh;padding:clamp(2rem,5vw,5rem);display:grid;grid-template-columns:minmax(260px,.62fr) minmax(0,1fr);gap:clamp(1.4rem,4vw,4.5rem);align-items:center;border-bottom:1px solid var(--album-soft);position:relative;overflow:hidden;background:radial-gradient(circle at 12% 15%,color-mix(in srgb,var(--album-accent) 13%,transparent),transparent 28%),linear-gradient(135deg,var(--album-bg),var(--album-bg-2));color:var(--album-text)}.page:after{content:"";position:absolute;left:clamp(2rem,5vw,5rem);right:clamp(2rem,5vw,5rem);bottom:1.2rem;height:1px;background:linear-gradient(90deg,transparent,var(--album-soft),transparent)}.page-cover{display:grid;grid-template-columns:1fr;overflow:hidden;color:white;background:#111}.cover-image{position:absolute;inset:0}.cover-image figure{margin:0;width:100%;height:100%}.cover-image figcaption{display:none}.cover-image:after{content:"";position:absolute;inset:0;background:var(--cover-overlay);pointer-events:none}.cover-image img{width:100%;height:100%;object-fit:cover;filter:var(--cover-filter)}figure{margin:0}img{display:block;width:100%;height:100%;object-fit:cover}.photo-zoom{display:block;width:100%;height:100%;padding:0;border:0;background:transparent;cursor:zoom-in;color:inherit}.cover-copy{position:relative;max-width:780px;align-self:end;margin-bottom:4vh;z-index:1}.cover-copy h1{font-size:clamp(3.2rem,9vw,8rem);line-height:.92;margin:.18em 0;letter-spacing:0}.cover-copy p{font-size:clamp(1rem,1.6vw,1.25rem);max-width:48rem;line-height:1.8;color:rgba(255,255,255,.84)}.kicker{color:var(--album-accent);text-transform:uppercase;letter-spacing:.12em;font-size:.78rem;font-weight:800}.page-cover .kicker{color:color-mix(in srgb,var(--album-accent) 42%,white 58%)}.page-copy{max-width:680px;align-self:center;position:relative;z-index:1}.page-copy h2{font-size:clamp(2rem,5.4vw,5.2rem);line-height:.98;margin:.12em 0;letter-spacing:0}.page-copy p{font-size:1rem;line-height:1.95;color:var(--album-muted)}.photo-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:clamp(.8rem,1.8vw,1.4rem);align-items:stretch;position:relative;z-index:1}.photo-grid figure{background:var(--album-panel);padding:.65rem;box-shadow:0 22px 70px var(--album-shadow);min-width:0}.photo-grid img{aspect-ratio:4/3;filter:var(--photo-filter)}.photo-grid figcaption{font-size:.82rem;line-height:1.55;margin-top:.55rem;color:var(--album-muted)}.layout-hero_story{grid-template-columns:minmax(0,1.1fr) minmax(300px,.9fr)}.layout-hero_story .photo-grid{grid-column:1;grid-row:1;grid-template-columns:1fr}.layout-hero_story .page-copy{grid-column:2;grid-row:1}.layout-hero_story .photo-grid img{aspect-ratio:4/5}.layout-single_photo_caption .photo-grid{grid-template-columns:minmax(0,760px)}.layout-diptych .photo-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.layout-triptych .photo-grid{grid-template-columns:repeat(3,minmax(0,1fr))}.layout-mosaic .photo-grid{grid-template-columns:1.1fr .9fr .9fr;grid-auto-rows:minmax(150px,22vh)}.layout-mosaic .photo-grid figure:first-child{grid-row:span 2}.layout-mosaic .photo-grid img,.layout-gallery_wall .photo-grid img,.layout-scrapbook .photo-grid img{height:100%;aspect-ratio:auto}.layout-gallery_wall{grid-template-columns:minmax(220px,.42fr) minmax(0,1fr)}.layout-gallery_wall .photo-grid{grid-template-columns:repeat(3,minmax(0,1fr));grid-auto-rows:minmax(130px,1fr)}.layout-gallery_wall .photo-grid figure:nth-child(1),.layout-gallery_wall .photo-grid figure:nth-child(5){grid-row:span 2}.layout-timeline_ribbon{grid-template-columns:1fr;align-content:center}.layout-timeline_ribbon .page-copy{max-width:900px}.layout-timeline_ribbon .photo-grid{grid-template-columns:repeat(auto-fit,minmax(120px,1fr));border-top:1px solid var(--album-soft);border-bottom:1px solid var(--album-soft);padding:1rem 0}.layout-timeline_ribbon .photo-grid figure{padding:.35rem;box-shadow:0 10px 28px var(--album-shadow)}.layout-timeline_ribbon .photo-grid figure:nth-child(2n){margin-top:2rem}.layout-full_bleed_quote{grid-template-columns:1fr;padding:0;color:white;background:#070707}.layout-full_bleed_quote .page-copy{align-self:end;max-width:780px;padding:clamp(2rem,6vw,5rem);z-index:2}.layout-full_bleed_quote .page-copy p{color:rgba(255,255,255,.78)}.layout-full_bleed_quote .photo-grid{position:absolute;inset:0;display:block}.layout-full_bleed_quote .photo-grid figure{width:100%;height:100%;padding:0;border:0;box-shadow:none;background:#000}.layout-full_bleed_quote .photo-grid figure:not(:first-child){display:none}.layout-full_bleed_quote .photo-grid:after{content:"";position:absolute;inset:0;background:var(--cover-overlay);pointer-events:none}.layout-full_bleed_quote .photo-grid img{height:100%;aspect-ratio:auto;filter:var(--cover-filter)}.layout-full_bleed_quote figcaption{display:none}.layout-scrapbook{grid-template-columns:minmax(220px,.46fr) minmax(0,1fr)}.layout-scrapbook .photo-grid{grid-template-columns:repeat(4,minmax(0,1fr));grid-auto-rows:minmax(105px,1fr)}.layout-scrapbook .photo-grid figure{padding:.42rem;transform:rotate(-1.2deg)}.layout-scrapbook .photo-grid figure:nth-child(2n){transform:rotate(1.4deg)}.layout-scrapbook .photo-grid figure:nth-child(1),.layout-scrapbook .photo-grid figure:nth-child(6){grid-column:span 2;grid-row:span 2}.layout-contact_sheet{grid-template-columns:1fr}.layout-contact_sheet .page-copy{max-width:920px}.layout-contact_sheet .photo-grid{grid-template-columns:repeat(auto-fit,minmax(130px,1fr));gap:.55rem}.layout-contact_sheet .photo-grid figure{padding:.35rem;box-shadow:none;border:1px solid var(--album-soft)}.layout-contact_sheet .photo-grid figcaption{font-size:.68rem}.layout-quote_break,.page-notes{grid-template-columns:minmax(0,820px);justify-content:center;text-align:center}.layout-quote_break .page-copy h2,.layout-ending_text .page-copy h2{font-size:clamp(2.6rem,7vw,6.5rem)}.layout-ending_text{grid-template-columns:minmax(0,.8fr) minmax(260px,.72fr)}.album-editorial .page-copy h2,.album-cinematic_bw .page-copy h2,.album-editorial .cover-copy h1,.album-cinematic_bw .cover-copy h1{text-transform:uppercase}.album-archive .kicker:before{content:"No. "}.album-postcard .photo-grid figure{transform:rotate(-.8deg)}.album-postcard .photo-grid figure:nth-child(2n){transform:rotate(.8deg)}.lightbox{position:fixed;inset:0;z-index:20;display:none;place-items:center;padding:2rem;background:rgba(6,8,10,.86);backdrop-filter:blur(18px)}.lightbox[data-open="true"]{display:grid}.lightbox img{max-width:min(92vw,1280px);max-height:82vh;width:auto;height:auto;object-fit:contain;box-shadow:0 30px 90px rgba(0,0,0,.45)}.lightbox p{margin:.9rem 0 0;color:white;max-width:760px;text-align:center;line-height:1.6}.lightbox-close{position:fixed;right:1.2rem;top:1rem;width:2.8rem;height:2.8rem;border:1px solid rgba(255,255,255,.25);border-radius:999px;background:rgba(255,255,255,.08);color:white;font-size:2rem;line-height:1;cursor:pointer}@media print{.lightbox{display:none!important}.photo-zoom{cursor:default}}@media(max-width:860px){.page,.layout-hero_story,.layout-ending_text,.layout-gallery_wall,.layout-scrapbook{grid-template-columns:1fr;padding:2rem 1rem}.layout-full_bleed_quote{padding:0}.layout-hero_story .photo-grid,.layout-hero_story .page-copy{grid-column:auto;grid-row:auto}.layout-diptych .photo-grid,.layout-triptych .photo-grid,.layout-mosaic .photo-grid,.layout-gallery_wall .photo-grid,.layout-scrapbook .photo-grid{grid-template-columns:1fr}.layout-mosaic .photo-grid figure:first-child,.layout-gallery_wall .photo-grid figure:nth-child(1),.layout-gallery_wall .photo-grid figure:nth-child(5),.layout-scrapbook .photo-grid figure:nth-child(1),.layout-scrapbook .photo-grid figure:nth-child(6){grid-column:auto;grid-row:auto}.layout-timeline_ribbon .photo-grid figure:nth-child(2n){margin-top:0}.cover-copy{margin-bottom:2rem}}`
}

func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func findChromeExecutable() (string, error) {
	if override := strings.TrimSpace(os.Getenv("CHROME_PATH")); override != "" {
		if strings.HasPrefix(override, "/") {
			if _, err := os.Stat(override); err == nil {
				return override, nil
			}
		} else if path, err := exec.LookPath(override); err == nil {
			return path, nil
		}
	}
	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"google-chrome",
		"chromium",
		"chromium-browser",
		"chrome",
	}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, "/") {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", errors.New("chrome executable not found")
}

func cropImage(src image.Image, box domain.CropBox) image.Image {
	rect := normalizeCropBox(src.Bounds(), box)
	if rect.Empty() {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(dst, dst.Bounds(), src, rect.Min, draw.Src)
	return dst
}

func expandCanvas(src image.Image, ratio int) image.Image {
	if ratio <= 0 {
		ratio = 18
	}
	bounds := src.Bounds()
	addW := bounds.Dx() * ratio / 100
	addH := bounds.Dy() * ratio / 100
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx()+addW*2, bounds.Dy()+addH*2))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)
	center := image.Rect(addW, addH, addW+bounds.Dx(), addH+bounds.Dy())
	xdraw.CatmullRom.Scale(dst, center, src, bounds, xdraw.Over, nil)
	return dst
}

func normalizeCropBox(bounds image.Rectangle, box domain.CropBox) image.Rectangle {
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	x := int(math.Round(box.X * w))
	y := int(math.Round(box.Y * h))
	cw := int(math.Round(box.W * w))
	ch := int(math.Round(box.H * h))
	rect := image.Rect(x, y, x+cw, y+ch).Intersect(bounds)
	return rect
}

func uniqueSuffix(prefix string) string {
	return fmt.Sprintf("%s_%d", strings.TrimSpace(prefix), time.Now().UnixNano())
}
