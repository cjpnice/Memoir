package mock

import (
	"context"
	"fmt"
	"time"

	"memoir/internal/ai"
	"memoir/internal/domain"
)

// Analyzer is a minimal test stub that satisfies the ai.Analyzer interface.
type Analyzer struct{}

// New returns a test stub analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// AnalyzeImage returns a fixed mid-range analysis for testing.
func (a *Analyzer) AnalyzeImage(ctx context.Context, input ai.AnalyzeImageInput) (*domain.ImageAnalysis, error) {
	_ = ctx
	return &domain.ImageAnalysis{
		QualityScore:      50,
		PreservationScore: 50,
		StoryScore:        50,
		Recommendation:    "keep",
		Reasons:           []string{"test stub"},
		DetectedContent: domain.DetectedContent{
			Scenes:  []string{},
			Objects: []string{},
			Mood:    []string{},
			Tags:    []string{},
		},
		Metrics: input.Metrics,
		CropSuggestions: []domain.CropSuggestion{
			{ID: "crop_1", AspectRatio: "4:5", Box: domain.CropBox{X: 0.1, Y: 0.1, W: 0.8, H: 0.8}, Reason: "test stub crop"},
		},
		EditSuggestions: []domain.ImageEditSuggestion{
			{Type: "crop", Strength: "medium", Reason: "test stub", Execution: "local", ActionLabel: "本地裁剪"},
			{Type: "color", Strength: "light", Reason: "test stub", Execution: "local", ActionLabel: "本地调色"},
			{Type: "rotate", Strength: "light", Reason: "test stub", Execution: "local", ActionLabel: "本地旋转"},
			{Type: "cleanup", Strength: "medium", Reason: "test stub", Execution: "local_approximation", ActionLabel: "本地使用裁剪弱化干扰物"},
			{Type: "expand", Strength: "medium", Reason: "test stub", Execution: "local_approximation", ActionLabel: "本地扩展画布"},
		},
		CaptionSeeds:  []string{"test stub caption"},
		AlbumRole:     "review_candidate",
		SocialCaption: "test stub social caption",
		ModelVersion:  ai.LocalModelVersion,
		PromptVersion: ai.AnalysisPromptVersion,
		CompletedAt:   time.Now(),
	}, nil
}

// OrganizeReview returns each image as its own single-image group.
func (a *Analyzer) OrganizeReview(ctx context.Context, input ai.ReviewOrganizationInput) (*ai.ReviewOrganization, error) {
	_ = ctx
	groups := make([]ai.ReviewGroupDecision, 0, len(input.Images))
	for _, image := range input.Images {
		if image == nil {
			continue
		}
		groups = append(groups, ai.ReviewGroupDecision{
			ID:            "grp_" + image.ID,
			Title:         "test stub",
			ImageIDs:      []string{image.ID},
			BestImageID:   image.ID,
			KeepImageIDs:  []string{image.ID},
			Reason:        "test stub",
			SocialImageID: image.ID,
		})
	}
	return &ai.ReviewOrganization{
		SelectionStrategy: "test stub: each image in its own group",
		Groups:            groups,
	}, nil
}

// WriteAlbumNarrative returns a minimal album narrative for testing.
func (a *Analyzer) WriteAlbumNarrative(ctx context.Context, input ai.AlbumNarrativeInput) (*ai.AlbumNarrative, error) {
	_ = ctx
	title := input.Project.Title
	if title == "" {
		title = "集忆"
	}
	pages := []ai.AlbumPagePlan{
		{PageType: "cover", LayoutID: "cover_full_bleed", Title: title, Body: "test stub intro", Caption: "test stub intro"},
		{PageType: "hero", LayoutID: "hero_story", Title: "主视觉", Body: "test stub hero body", Caption: "test stub hero caption"},
		{PageType: "pause", LayoutID: "quote_break", Title: "留白", Body: "test stub pause body", Caption: "test stub pause caption"},
		{PageType: "ending", LayoutID: "ending_text", Title: "结尾", Body: "test stub ending", Caption: "test stub ending"},
	}
	if len(input.Images) > 0 {
		pages[0].ImageIDs = []string{input.Images[0].ID}
		pages[1].ImageIDs = []string{input.Images[minInt(len(input.Images)-1, 1)].ID}
		pages[3].ImageIDs = []string{input.Images[len(input.Images)-1].ID}
	}
	return &ai.AlbumNarrative{
		Title:         title,
		Intro:         "test stub intro",
		ChapterTitles: []string{"片段"},
		Ending:        "test stub ending",
		DesignNotes:   "test stub design notes",
		Pages:         pages,
		SocialPosts: []ai.SocialPost{
			{Platform: "moments", Title: title + " · 朋友圈", Hook: "test stub hook", Body: "test stub moments body", ImageIDs: pages[0].ImageIDs, Hashtags: []string{"#test"}},
			{Platform: "xiaohongshu", Title: title + " · 小红书", Hook: "test stub hook", Body: "test stub xiaohongshu body", ImageIDs: pages[0].ImageIDs, Hashtags: []string{"#test"}},
		},
		ModelVersion:  ai.LocalModelVersion,
		PromptVersion: ai.AlbumPromptVersion,
	}, nil
}

// Versions reports the active test stub version bundle.
func (a *Analyzer) Versions() ai.AnalyzerVersions {
	return ai.AnalyzerVersions{
		AnalysisModelVersion:  ai.LocalModelVersion,
		AnalysisPromptVersion: ai.AnalysisPromptVersion,
		ReviewModelVersion:    ai.LocalModelVersion,
		ReviewPromptVersion:   ai.ReviewPromptVersion,
		AlbumModelVersion:     ai.LocalModelVersion,
		AlbumPromptVersion:    ai.AlbumPromptVersion,
	}
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Ensure Analyzer satisfies ai.Analyzer at compile time.
var _ ai.Analyzer = (*Analyzer)(nil)

// FormatVersion returns a formatted version string for testing.
func FormatVersion(major, minor int) string {
	return fmt.Sprintf("mock-v%d.%d", major, minor)
}
