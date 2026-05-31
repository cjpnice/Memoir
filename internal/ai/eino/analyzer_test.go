package eino

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"memoir/internal/ai"
	"memoir/internal/domain"
)

func TestBuildAnalysisPromptIncludesCoreRules(t *testing.T) {
	input := ai.AnalyzeImageInput{
		Project: domain.Project{
			Title: "周末旅行",
			Tone:  "film",
		},
		Image: domain.ImageAsset{
			FileName: "cover.heic",
			Status:   domain.ImageStatusUploaded,
		},
		Metrics: domain.ImageMetrics{
			AspectRatio: 1.5,
			Brightness:  0.42,
			Contrast:    0.18,
			Sharpness:   0.63,
			FileSize:    128000,
			Width:       3000,
			Height:      2000,
		},
		ImageMIMEType: "image/heic",
	}

	prompt := buildAnalysisPrompt(input)

	for _, want := range []string{
		"严格 JSON",
		"帮用户减少筛图工作量",
		"cropSuggestions",
		"socialCaption",
		"周末旅行",
		"cover.heic",
		"image/heic",
		"execution 和 actionLabel 是必填字段",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("analysis prompt missing %q", want)
		}
	}
}

func TestBuildReviewAndAlbumPromptsIncludeSetLevelConstraints(t *testing.T) {
	project := domain.Project{
		Title: "家庭记忆",
		Tone:  "warm",
	}
	images := []*domain.ImageAsset{
		{
			ID:       "img_1",
			FileName: "a.jpg",
			Status:   domain.ImageStatusKeep,
			Analysis: &domain.ImageAnalysis{
				QualityScore:      91,
				PreservationScore: 94,
				StoryScore:        88,
				Recommendation:    "keep",
				Reasons:           []string{"sharp"},
				CaptionSeeds:      []string{"one"},
				AlbumRole:         "chapter_anchor",
				SocialCaption:     "shareable",
			},
		},
	}
	reviewPrompt := buildReviewOrganizationPrompt(ai.ReviewOrganizationInput{
		Project: project,
		Images:  images,
	})
	for _, want := range []string{
		"只使用输入中出现的 id",
		"img_1",
		"识别相似照片",
		"家庭记忆",
	} {
		if !strings.Contains(reviewPrompt, want) {
			t.Fatalf("review prompt missing %q", want)
		}
	}

	albumPrompt := buildAlbumPrompt(ai.AlbumNarrativeInput{
		Project:       project,
		Images:        images,
		ThemeID:       "warm_family",
		SelectedCount: len(images),
	})
	for _, want := range []string{
		"朋友圈/小红书",
		"5-14 个页面",
		"hero_story",
		"quote_break",
		"gallery_wall",
		"full_bleed_quote",
		"主题风格说明",
		"pageType 不是目录层级",
		"禁止",
		"opener|hero|detail|sequence|collage|gallery|pause|index",
		"socialPosts",
		"platform=\"moments\"",
		"platform=\"xiaohongshu\"",
		"hook",
		"warm_family",
		"家庭记忆",
	} {
		if !strings.Contains(albumPrompt, want) {
			t.Fatalf("album prompt missing %q", want)
		}
	}
}

func TestDecodeAndNormalizeModelResponse(t *testing.T) {
	var analysis domain.ImageAnalysis
	content := "模型返回：\n```json\n{\"qualityScore\":120,\"preservationScore\":-5,\"storyScore\":88,\"recommendation\":\"\",\"reasons\":[\"ok\"],\"detectedContent\":{\"peopleCount\":1,\"scenes\":null,\"objects\":[\"tree\"],\"mood\":null,\"tags\":null},\"editSuggestions\":[{\"type\":\"cleanup\",\"reason\":\"路人闯入边缘\",\"execution\":\"local_approximation\"},{\"type\":\"expand\",\"reason\":\"画面左右留白不够\",\"execution\":\"local_approximation\"},{\"type\":\"unknown_fix\",\"reason\":\"需要重绘背景\",\"execution\":\"provider_generative\",\"providerBacked\":true},{\"type\":\"rotate\",\"reason\":\"地平线略歪\",\"execution\":\"local\",\"actionLabel\":\"本地旋转校正\"}]}\n```"
	if err := decodeJSONMessage(content, &analysis); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	normalizeAnalysis(&analysis)
	if analysis.QualityScore != 100 {
		t.Fatalf("expected clamped quality score 100, got %d", analysis.QualityScore)
	}
	if analysis.PreservationScore != 0 {
		t.Fatalf("expected clamped preservation score 0, got %d", analysis.PreservationScore)
	}
	if analysis.Recommendation != "review" {
		t.Fatalf("expected default recommendation review, got %q", analysis.Recommendation)
	}
	if analysis.DetectedContent.Scenes == nil || analysis.DetectedContent.Mood == nil || analysis.DetectedContent.Tags == nil {
		t.Fatalf("expected detected content slices to be normalized, got %#v", analysis.DetectedContent)
	}
	if len(analysis.DetectedContent.Objects) != 1 || analysis.DetectedContent.Objects[0] != "tree" {
		t.Fatalf("expected detected content objects to survive normalization, got %#v", analysis.DetectedContent.Objects)
	}
	if len(analysis.EditSuggestions) != 4 {
		t.Fatalf("expected normalized edit suggestions, got %d", len(analysis.EditSuggestions))
	}
	if analysis.EditSuggestions[0].Execution != "local_approximation" {
		t.Fatalf("expected cleanup execution local_approximation, got %q", analysis.EditSuggestions[0].Execution)
	}
	if analysis.EditSuggestions[1].Execution != "local_approximation" {
		t.Fatalf("expected expand execution local_approximation, got %q", analysis.EditSuggestions[1].Execution)
	}
	if analysis.EditSuggestions[2].Execution != "provider_generative" || !analysis.EditSuggestions[2].ProviderBacked {
		t.Fatalf("expected provider-backed edit to normalize as provider generative, got %#v", analysis.EditSuggestions[2])
	}
	if analysis.EditSuggestions[3].Execution != "local" {
		t.Fatalf("expected explicit execution to survive normalization, got %#v", analysis.EditSuggestions[3])
	}
	if analysis.EditSuggestions[3].ActionLabel != "本地旋转校正" {
		t.Fatalf("expected explicit action label to survive normalization, got %q", analysis.EditSuggestions[3].ActionLabel)
	}

	review := ai.ReviewOrganization{
		Groups: []ai.ReviewGroupDecision{
			{ID: "grp_1", ImageIDs: []string{"img_1"}, BestImageID: "img_1"},
			{ID: "grp_2", ImageIDs: []string{"img_2"}, BestImageID: "img_2", Title: "", SocialImageID: ""},
		},
	}
	normalizeReviewOrganization(&review)
	if len(review.Groups[1].KeepImageIDs) != 1 || review.Groups[1].KeepImageIDs[0] != "img_2" {
		t.Fatalf("expected fallback keep ids, got %#v", review.Groups[1].KeepImageIDs)
	}
	if review.Groups[1].SocialImageID != "img_2" {
		t.Fatalf("expected fallback social image id, got %q", review.Groups[1].SocialImageID)
	}
}

func TestBuildImageAnalysisMessageIncludesImagePart(t *testing.T) {
	input := ai.AnalyzeImageInput{
		ImageBase64:   "base64-image-data",
		ImageMIMEType: "image/jpeg",
	}

	message := buildImageAnalysisMessage("look carefully", input)

	if len(message.UserInputMultiContent) != 2 {
		t.Fatalf("expected text and image parts, got %d", len(message.UserInputMultiContent))
	}
	if message.UserInputMultiContent[0].Type != schema.ChatMessagePartTypeText {
		t.Fatalf("expected text part first, got %s", message.UserInputMultiContent[0].Type)
	}
	if message.UserInputMultiContent[0].Text != "look carefully" {
		t.Fatalf("unexpected text prompt: %q", message.UserInputMultiContent[0].Text)
	}
	imagePart := message.UserInputMultiContent[1]
	if imagePart.Type != schema.ChatMessagePartTypeImageURL {
		t.Fatalf("expected image part second, got %s", imagePart.Type)
	}
	if imagePart.Image == nil || imagePart.Image.Base64Data == nil {
		t.Fatalf("expected image base64 payload")
	}
	if *imagePart.Image.Base64Data != "base64-image-data" {
		t.Fatalf("unexpected image payload: %q", *imagePart.Image.Base64Data)
	}
	if imagePart.Image.MIMEType != "image/jpeg" {
		t.Fatalf("unexpected image MIME type: %q", imagePart.Image.MIMEType)
	}
}
