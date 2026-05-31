package domain

import "time"

// ProjectStatus describes the lifecycle of a project.
type ProjectStatus string

const (
	ProjectStatusDraft           ProjectStatus = "draft"
	ProjectStatusUploading       ProjectStatus = "uploading"
	ProjectStatusAnalyzing       ProjectStatus = "analyzing"
	ProjectStatusReviewing       ProjectStatus = "reviewing"
	ProjectStatusGeneratingAlbum ProjectStatus = "generating_album"
	ProjectStatusEditing         ProjectStatus = "editing"
	ProjectStatusExporting       ProjectStatus = "exporting"
	ProjectStatusDone            ProjectStatus = "done"
	ProjectStatusFailed          ProjectStatus = "failed"
)

// ImageStatus describes the current decision for an image.
type ImageStatus string

const (
	ImageStatusUploaded        ImageStatus = "uploaded"
	ImageStatusAnalyzing       ImageStatus = "analyzing"
	ImageStatusAnalyzed        ImageStatus = "analyzed"
	ImageStatusKeep            ImageStatus = "keep"
	ImageStatusImproveThenKeep ImageStatus = "improve_then_keep"
	ImageStatusReview          ImageStatus = "review"
	ImageStatusRejectSuggested ImageStatus = "reject_suggested"
	ImageStatusApproved        ImageStatus = "approved"
	ImageStatusExcluded        ImageStatus = "excluded"
)

// AlbumStatus describes the lifecycle of a generated album.
type AlbumStatus string

const (
	AlbumStatusDraft     AlbumStatus = "draft"
	AlbumStatusGenerated AlbumStatus = "generated"
	AlbumStatusEdited    AlbumStatus = "edited"
	AlbumStatusExported  AlbumStatus = "exported"
)

// TaskStatus describes a persisted long-running task state.
type TaskStatus string

const (
	TaskStatusRunning     TaskStatus = "running"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusFailed      TaskStatus = "failed"
	TaskStatusInterrupted TaskStatus = "interrupted"
)

// TaskType identifies a long-running project workflow.
type TaskType string

const (
	TaskTypeAnalysis        TaskType = "analysis"
	TaskTypeAlbumGeneration TaskType = "album_generation"
	TaskTypeExport          TaskType = "export"
)

// Project is the top-level album workspace.
type Project struct {
	ID                           string         `json:"id"`
	Title                        string         `json:"title"`
	Description                  string         `json:"description"`
	Location                     string         `json:"location,omitempty"`
	Place                        *ProjectPlace  `json:"place,omitempty"`
	Tone                         string         `json:"tone"`
	ThemeID                      string         `json:"themeId"`
	Status                       ProjectStatus  `json:"status"`
	AnalysisStatus               string         `json:"analysisStatus"`
	AnalysisProgress             int            `json:"analysisProgress"`
	AnalysisModelVersion         string         `json:"analysisModelVersion,omitempty"`
	AnalysisPromptVersion        string         `json:"analysisPromptVersion,omitempty"`
	CurrentAnalysisModelVersion  string         `json:"currentAnalysisModelVersion,omitempty"`
	CurrentAnalysisPromptVersion string         `json:"currentAnalysisPromptVersion,omitempty"`
	PendingAnalysisCount         int            `json:"pendingAnalysisCount,omitempty"`
	StaleAnalysisCount           int            `json:"staleAnalysisCount,omitempty"`
	CurrentStep                  string         `json:"currentStep"`
	LastError                    string         `json:"lastError,omitempty"`
	CreatedAt                    time.Time      `json:"createdAt"`
	UpdatedAt                    time.Time      `json:"updatedAt"`
	Images                       []*ImageAsset  `json:"images"`
	Album                        *Album         `json:"album,omitempty"`
	ActiveTask                   *ProjectTask   `json:"activeTask,omitempty"`
	TaskHistory                  []*ProjectTask `json:"taskHistory,omitempty"`
	Exports                      []*AlbumExport `json:"exports,omitempty"`
}

// ProjectPlace stores structured location metadata for map-based memory views.
type ProjectPlace struct {
	City       string  `json:"city"`
	Region     string  `json:"region,omitempty"`
	Country    string  `json:"country"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence,omitempty"`
}

// ProjectTask stores persisted progress for analysis, generation, and export work.
type ProjectTask struct {
	ID          string     `json:"id"`
	Type        TaskType   `json:"type"`
	Status      TaskStatus `json:"status"`
	Progress    int        `json:"progress"`
	Message     string     `json:"message"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt time.Time  `json:"completedAt,omitempty"`
}

// ImageAsset is a stored photo plus its analysis state.
type ImageAsset struct {
	ID           string           `json:"id"`
	ProjectID    string           `json:"projectId"`
	FileName     string           `json:"fileName"`
	MimeType     string           `json:"mimeType"`
	FileSize     int64            `json:"fileSize"`
	Width        int              `json:"width"`
	Height       int              `json:"height"`
	OriginalURL  string           `json:"originalUrl"`
	ThumbnailURL string           `json:"thumbnailUrl"`
	DerivedURL   string           `json:"derivedUrl,omitempty"`
	Status       ImageStatus      `json:"status"`
	UserDecision string           `json:"userDecision,omitempty"`
	EditHistory  []*ImageSnapshot `json:"editHistory,omitempty"`
	Analysis     *ImageAnalysis   `json:"analysis,omitempty"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// ImageSnapshot stores an undoable image edit state.
type ImageSnapshot struct {
	DerivedURL   string      `json:"derivedUrl,omitempty"`
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	Status       ImageStatus `json:"status"`
	UserDecision string      `json:"userDecision,omitempty"`
	Reason       string      `json:"reason,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
}

// ImageAnalysis is the structured result produced by the AI layer.
type ImageAnalysis struct {
	QualityScore       int                   `json:"qualityScore"`
	PreservationScore  int                   `json:"preservationScore"`
	StoryScore         int                   `json:"storyScore"`
	Recommendation     string                `json:"recommendation"`
	Reasons            []string              `json:"reasons"`
	DetectedContent    DetectedContent       `json:"detectedContent"`
	Metrics            ImageMetrics          `json:"metrics"`
	Issues             []ImageIssue          `json:"issues"`
	CropSuggestions    []CropSuggestion      `json:"cropSuggestions"`
	CaptionSeeds       []string              `json:"captionSeeds"`
	EditSuggestions    []ImageEditSuggestion `json:"editSuggestions,omitempty"`
	SimilarGroupID     string                `json:"similarGroupId,omitempty"`
	SimilarGroupLabel  string                `json:"similarGroupLabel,omitempty"`
	SimilarGroupRank   int                   `json:"similarGroupRank,omitempty"`
	SimilarGroupBest   bool                  `json:"similarGroupBest,omitempty"`
	SimilarGroupReason string                `json:"similarGroupReason,omitempty"`
	AlbumRole          string                `json:"albumRole,omitempty"`
	SocialCaption      string                `json:"socialCaption,omitempty"`
	SelectionRank      int                   `json:"selectionRank,omitempty"`
	ModelVersion       string                `json:"modelVersion"`
	PromptVersion      string                `json:"promptVersion"`
	CompletedAt        time.Time             `json:"completedAt"`
}

// ImageMetrics contains low-level heuristics computed locally.
type ImageMetrics struct {
	AspectRatio float64 `json:"aspectRatio"`
	Brightness  float64 `json:"brightness"`
	Contrast    float64 `json:"contrast"`
	Sharpness   float64 `json:"sharpness"`
	FileSize    int64   `json:"fileSize"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
}

// DetectedContent captures coarse semantic tags.
type DetectedContent struct {
	PeopleCount int      `json:"peopleCount"`
	Scenes      []string `json:"scenes"`
	Objects     []string `json:"objects"`
	Mood        []string `json:"mood"`
	Tags        []string `json:"tags"`
}

// ImageIssue describes a photo problem that might be worth correcting.
type ImageIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// ImageEditSuggestion describes an AI-recommended non-destructive edit.
type ImageEditSuggestion struct {
	Type           string `json:"type"`
	Strength       string `json:"strength,omitempty"`
	Reason         string `json:"reason"`
	Execution      string `json:"execution,omitempty"`
	ProviderBacked bool   `json:"providerBacked,omitempty"`
	ActionLabel    string `json:"actionLabel,omitempty"`
}

// CropSuggestion recommends a non-destructive crop.
type CropSuggestion struct {
	ID          string  `json:"id"`
	AspectRatio string  `json:"aspectRatio"`
	Box         CropBox `json:"box"`
	Reason      string  `json:"reason"`
}

// CropBox stores normalized crop coordinates.
type CropBox struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Album is the generated output for a project.
type Album struct {
	ID            string             `json:"id"`
	ProjectID     string             `json:"projectId"`
	ThemeID       string             `json:"themeId"`
	Title         string             `json:"title"`
	Intro         string             `json:"intro"`
	DesignNotes   string             `json:"designNotes,omitempty"`
	Status        AlbumStatus        `json:"status"`
	Version       int                `json:"version"`
	ModelVersion  string             `json:"modelVersion,omitempty"`
	PromptVersion string             `json:"promptVersion,omitempty"`
	Pages         []*AlbumPage       `json:"pages"`
	SocialPosts   []*AlbumSocialPost `json:"socialPosts,omitempty"`
	EditHistory   []*AlbumSnapshot   `json:"editHistory,omitempty"`
	RedoStack     []*AlbumSnapshot   `json:"redoStack,omitempty"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

// AlbumSnapshot stores an undoable album editing state.
type AlbumSnapshot struct {
	Title     string       `json:"title"`
	Intro     string       `json:"intro"`
	ThemeID   string       `json:"themeId"`
	Pages     []*AlbumPage `json:"pages"`
	Version   int          `json:"version"`
	Reason    string       `json:"reason"`
	CreatedAt time.Time    `json:"createdAt"`
}

// AlbumExport describes a generated album export artifact.
type AlbumExport struct {
	ID        string    `json:"id"`
	AlbumID   string    `json:"albumId"`
	ProjectID string    `json:"projectId"`
	Type      string    `json:"type"`
	URL       string    `json:"url"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// AlbumPage is a single album page or spread.
type AlbumPage struct {
	ID       string   `json:"id"`
	Order    int      `json:"order"`
	PageType string   `json:"pageType"`
	LayoutID string   `json:"layoutId"`
	ImageIDs []string `json:"imageIds"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Caption  string   `json:"caption"`
}

// AlbumSocialPost is a share-ready post derived from the album.
type AlbumSocialPost struct {
	Platform string   `json:"platform,omitempty"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Hook     string   `json:"hook,omitempty"`
	ImageIDs []string `json:"imageIds"`
	Hashtags []string `json:"hashtags,omitempty"`
}
