package ai

import (
	"context"

	"memoir/internal/domain"
)

// Analyzer produces image analysis and album narration.
type Analyzer interface {
	AnalyzeImage(ctx context.Context, input AnalyzeImageInput) (*domain.ImageAnalysis, error)
	OrganizeReview(ctx context.Context, input ReviewOrganizationInput) (*ReviewOrganization, error)
	WriteAlbumNarrative(ctx context.Context, input AlbumNarrativeInput) (*AlbumNarrative, error)
	Versions() AnalyzerVersions
}

const (
	AnalysisPromptVersion = "analysis-v3"
	ReviewPromptVersion   = "review-v2"
	AlbumPromptVersion    = "album-v6"
	LocalModelVersion     = "mock-v2"
)

// AnalyzerVersions identifies the model and prompt bundle currently in use.
type AnalyzerVersions struct {
	AnalysisModelVersion  string `json:"analysisModelVersion"`
	AnalysisPromptVersion string `json:"analysisPromptVersion"`
	ReviewModelVersion    string `json:"reviewModelVersion"`
	ReviewPromptVersion   string `json:"reviewPromptVersion"`
	AlbumModelVersion     string `json:"albumModelVersion"`
	AlbumPromptVersion    string `json:"albumPromptVersion"`
}

// AlbumNarrative is the short text bundle used when generating albums.
type AlbumNarrative struct {
	Title         string          `json:"title"`
	Intro         string          `json:"intro"`
	ChapterTitles []string        `json:"chapterTitles"`
	Ending        string          `json:"ending"`
	DesignNotes   string          `json:"designNotes,omitempty"`
	Pages         []AlbumPagePlan `json:"pages,omitempty"`
	SocialPosts   []SocialPost    `json:"socialPosts,omitempty"`
	ModelVersion  string          `json:"modelVersion"`
	PromptVersion string          `json:"promptVersion"`
}

// AlbumPagePlan is the model-authored structure for one album page.
type AlbumPagePlan struct {
	PageType string   `json:"pageType"`
	LayoutID string   `json:"layoutId"`
	ImageIDs []string `json:"imageIds"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Caption  string   `json:"caption"`
}

// SocialPost is a share-ready WeChat/Xiaohongshu-style post.
type SocialPost struct {
	Platform string   `json:"platform,omitempty"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Hook     string   `json:"hook,omitempty"`
	ImageIDs []string `json:"imageIds"`
	Hashtags []string `json:"hashtags,omitempty"`
}

// AnalyzeImageInput describes one image and its local metrics.
type AnalyzeImageInput struct {
	Project       domain.Project      `json:"project"`
	Image         domain.ImageAsset   `json:"image"`
	Metrics       domain.ImageMetrics `json:"metrics"`
	ImageBase64   string              `json:"-"`
	ImageMIMEType string              `json:"imageMimeType"`
	Index         int                 `json:"index"`
	Total         int                 `json:"total"`
}

// AlbumNarrativeInput describes the selected photo set.
type AlbumNarrativeInput struct {
	Project       domain.Project       `json:"project"`
	Images        []*domain.ImageAsset `json:"images"`
	ThemeID       string               `json:"themeId"`
	SelectedCount int                  `json:"selectedCount"`
}

// ReviewOrganizationInput asks the model to identify similar images and organize them into groups.
type ReviewOrganizationInput struct {
	Project domain.Project       `json:"project"`
	Images  []*domain.ImageAsset `json:"images"`
}

// ReviewOrganization is the model's set-level curation decision.
type ReviewOrganization struct {
	SelectionStrategy string                `json:"selectionStrategy,omitempty"`
	Groups            []ReviewGroupDecision `json:"groups"`
}

// ReviewGroupDecision ranks one similar-photo group and chooses the best image.
type ReviewGroupDecision struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	ImageIDs       []string `json:"imageIds"`
	BestImageID    string   `json:"bestImageId"`
	KeepImageIDs   []string `json:"keepImageIds,omitempty"`
	RejectImageIDs []string `json:"rejectImageIds,omitempty"`
	Reason         string   `json:"reason"`
	StoryBeat      string   `json:"storyBeat,omitempty"`
	SocialImageID  string   `json:"socialImageId,omitempty"`
}
