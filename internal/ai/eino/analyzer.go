package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"memoir/internal/ai"
	"memoir/internal/config"
	"memoir/internal/domain"
)

// Analyzer wraps an Eino chat agent for photo curation and album copy.
type Analyzer struct {
	chatModel  model.BaseChatModel
	modelLabel string
}

// New builds an Eino-backed analyzer.
func New(cfg config.Config) (*Analyzer, error) {
	ctx := context.Background()
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.OpenAIAPIKey,
		BaseURL: cfg.OpenAIBaseURL,
		Model:   cfg.OpenAIModel,
	})
	if err != nil {
		return nil, err
	}

	return &Analyzer{
		chatModel:  chatModel,
		modelLabel: "eino/" + cfg.OpenAIModel,
	}, nil
}

// AnalyzeImage sends the actual image to the multimodal model and asks for structured curation output.
func (a *Analyzer) AnalyzeImage(ctx context.Context, input ai.AnalyzeImageInput) (*domain.ImageAnalysis, error) {
	if input.ImageBase64 == "" {
		return nil, fmt.Errorf("图片数据为空，无法分析")
	}

	prompt := buildAnalysisPrompt(input)
	msg, err := a.generateWithRetry(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是 Memoir / 集忆 的资深照片策展大模型。你必须直接观察图片，替用户做严格筛选、保留理由、优化建议和回忆价值判断。只返回 JSON，不要使用 Markdown 代码块。",
		},
		buildImageAnalysisMessage(prompt, input),
	}, 2)
	if err != nil {
		return nil, fmt.Errorf("图片分析失败: %w", err)
	}

	var analysis domain.ImageAnalysis
	if err := decodeJSONMessage(msg.Content, &analysis); err != nil {
		return nil, fmt.Errorf("图片分析结果解析失败: %w", err)
	}
	normalizeAnalysis(&analysis)
	analysis.Metrics = input.Metrics
	analysis.ModelVersion = a.modelLabel
	analysis.PromptVersion = ai.AnalysisPromptVersion
	analysis.CompletedAt = time.Now()
	return &analysis, nil
}

// OrganizeReview receives all analyzed images and asks the LLM to identify similar groups and rank them.
func (a *Analyzer) OrganizeReview(ctx context.Context, input ai.ReviewOrganizationInput) (*ai.ReviewOrganization, error) {
	prompt := buildReviewOrganizationPrompt(input)
	msg, err := a.generateWithRetry(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是 Memoir / 集忆 的照片总编。你会收到一个项目的所有照片分析摘要。你需要：1）识别哪些照片是视觉相似的（同一场景、连拍、近重复）；2）将相似照片分组，为每组挑选最值得长期保存的一张；3）不相似的照片各自成为单图组。只返回 JSON。",
		},
		{Role: schema.User, Content: prompt},
	}, 2)
	if err != nil {
		return nil, fmt.Errorf("照片分组分析失败: %w", err)
	}

	var review ai.ReviewOrganization
	if err := decodeJSONMessage(msg.Content, &review); err != nil {
		return nil, fmt.Errorf("照片分组结果解析失败: %w", err)
	}
	normalizeReviewOrganization(&review)
	if len(review.Groups) == 0 {
		return nil, fmt.Errorf("照片分组结果为空")
	}
	return &review, nil
}

// WriteAlbumNarrative asks the model for a title and short narration bundle.
func (a *Analyzer) WriteAlbumNarrative(ctx context.Context, input ai.AlbumNarrativeInput) (*ai.AlbumNarrative, error) {
	prompt := buildAlbumPrompt(input)
	msg, err := a.generateWithRetry(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是 Memoir / 集忆 的艺术相册策划、叙事编辑和视觉设计总监。你要自动生成可长期保存的人生经历相册，像一本完成度高的摄影书，而不是照片清单；同时单独给出可发朋友圈/小红书的图文草稿。只返回 JSON，不要使用 Markdown 代码块。",
		},
		{Role: schema.User, Content: prompt},
	}, 2)
	if err != nil {
		return nil, fmt.Errorf("相册叙事生成失败: %w", err)
	}

	var narrative ai.AlbumNarrative
	if err := decodeJSONMessage(msg.Content, &narrative); err != nil {
		return nil, fmt.Errorf("相册叙事结果解析失败: %w", err)
	}
	narrative.ModelVersion = a.modelLabel
	narrative.PromptVersion = ai.AlbumPromptVersion
	return &narrative, nil
}

// Versions reports the active model and prompt bundle used by this analyzer.
func (a *Analyzer) Versions() ai.AnalyzerVersions {
	return ai.AnalyzerVersions{
		AnalysisModelVersion:  a.modelLabel,
		AnalysisPromptVersion: ai.AnalysisPromptVersion,
		ReviewModelVersion:    a.modelLabel,
		ReviewPromptVersion:   ai.ReviewPromptVersion,
		AlbumModelVersion:     a.modelLabel,
		AlbumPromptVersion:    ai.AlbumPromptVersion,
	}
}

func (a *Analyzer) generateWithRetry(ctx context.Context, messages []*schema.Message, maxRetries int) (*schema.Message, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		msg, err := a.chatModel.Generate(ctx, messages)
		if err == nil {
			return msg, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("大模型调用失败（重试 %d 次后）: %w", maxRetries, lastErr)
}

type reviewPromptImage struct {
	ID                string              `json:"id"`
	FileName          string              `json:"fileName"`
	Status            string              `json:"status"`
	Width             int                 `json:"width"`
	Height            int                 `json:"height"`
	UserDecision      string              `json:"userDecision,omitempty"`
	CreatedAt         string              `json:"createdAt,omitempty"`
	QualityScore      int                 `json:"qualityScore"`
	PreservationScore int                 `json:"preservationScore"`
	StoryScore        int                 `json:"storyScore"`
	Recommendation    string              `json:"recommendation"`
	Reasons           []string            `json:"reasons,omitempty"`
	CaptionSeeds      []string            `json:"captionSeeds,omitempty"`
	Issues            []domain.ImageIssue `json:"issues,omitempty"`
	Tags              []string            `json:"tags,omitempty"`
	AlbumRole         string              `json:"albumRole,omitempty"`
}

type albumPromptImage struct {
	ID                string   `json:"id"`
	FileName          string   `json:"fileName"`
	Status            string   `json:"status"`
	QualityScore      int      `json:"qualityScore"`
	PreservationScore int      `json:"preservationScore"`
	StoryScore        int      `json:"storyScore"`
	Reasons           []string `json:"reasons,omitempty"`
	CaptionSeeds      []string `json:"captionSeeds,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	GroupLabel        string   `json:"groupLabel,omitempty"`
	GroupBest         bool     `json:"groupBest,omitempty"`
	AlbumRole         string   `json:"albumRole,omitempty"`
	SocialCaption     string   `json:"socialCaption,omitempty"`
}

func buildImageAnalysisMessage(prompt string, input ai.AnalyzeImageInput) *schema.Message {
	return &schema.Message{
		Role: schema.User,
		UserInputMultiContent: []schema.MessageInputPart{
			{
				Type: schema.ChatMessagePartTypeText,
				Text: prompt,
			},
			{
				Type: schema.ChatMessagePartTypeImageURL,
				Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{
						Base64Data: &input.ImageBase64,
						MIMEType:   input.ImageMIMEType,
					},
					Detail: schema.ImageURLDetailHigh,
				},
			},
		},
	}
}

func buildAnalysisPrompt(input ai.AnalyzeImageInput) string {
	return fmt.Sprintf(`
	你是 Memoir/集忆 的照片筛选器。请直接观察随消息附带的图片，再结合给定的结构化信息，输出严格 JSON，不要输出解释文字。

	输出字段：
	{
	  "qualityScore": 0-100,
	  "preservationScore": 0-100,
	  "storyScore": 0-100,
	  "recommendation": "keep|improve_then_keep|review|reject_suggested",
	  "reasons": ["..."],
	  "detectedContent": {"peopleCount":0,"scenes":["..."],"objects":["..."],"mood":["..."],"tags":["..."]},
	  "issues": [{"type":"blur","severity":"low|medium|high","description":"..."}],
	  "cropSuggestions": [{"id":"crop_1","aspectRatio":"4:5","box":{"x":0.1,"y":0.1,"w":0.8,"h":0.8},"reason":"..."}],
	  "captionSeeds": ["..."],
	  "editSuggestions": [{"type":"crop|rotate|color|cleanup|expand","strength":"light|medium|strong","reason":"...","execution":"local|local_approximation|provider_generative","providerBacked":false,"actionLabel":"..."}],
	  "albumRole": "cover|chapter_anchor|detail|transition|social|exclude|review_candidate",
	  "socialCaption": "适合朋友圈/小红书的一句真实短文"
	}

	项目标题: %s
	项目语气: %s
	照片文件名: %s
	尺寸: %d x %d
	比例: %.4f
	文件大小: %d
	模型图片输入: %s
	局部指标:
	- brightness: %.3f
	- contrast: %.3f
	- sharpness: %.3f

	规则：
	1. 判断必须以画面本身为主，包括清晰度、主体、人物表情、构图、背景干扰、事件信息、时间感和情绪价值。
	2. 请站在"帮用户减少筛图工作量"的角度严格筛选：模糊、闭眼、重复、无主体、信息弱、背景干扰严重的照片优先 reject_suggested。
	3. 如果你判断这张图是多张相似连拍中的一张，在 reasons 中说明并相应调整 preservationScore。
	4. 有长期回忆价值但需要裁剪/调色/去干扰物时推荐 improve_then_keep，并在 editSuggestions 写出优化动作。
	5. editSuggestions 必须区分能力边界：crop/rotate/color 通常是 local；基于裁剪的去干扰物或非生成式扩图是 local_approximation；真正需要修复背景、移除物体、生成边缘内容、重绘人物/场景时标为 provider_generative 且 providerBacked=true。execution 和 actionLabel 是必填字段。
	6. preservationScore 代表多年后是否还值得回看，storyScore 代表是否能承载经历叙事，qualityScore 代表画面技术质量；三者不要机械相同。
	7. cropSuggestions 要基于你看到的画面主体，使用归一化坐标，尽量保守，优先保留主体和能产生回忆感的环境线索。
	8. captionSeeds 要像相册配文，不要营销腔；socialCaption 可以更适合朋友圈或小红书，但不能编造事实。
	9. albumRole 用于后续自动生成相册，请判断这张图更适合当封面、章节主图、细节、过渡、社交分享图，还是应排除。
	`, input.Project.Title, input.Project.Tone, input.Image.FileName, input.Metrics.Width, input.Metrics.Height, input.Metrics.AspectRatio, input.Metrics.FileSize, input.ImageMIMEType, input.Metrics.Brightness, input.Metrics.Contrast, input.Metrics.Sharpness)
}

func buildReviewOrganizationPrompt(input ai.ReviewOrganizationInput) string {
	payload := struct {
		ProjectTitle string              `json:"projectTitle"`
		ProjectTone  string              `json:"projectTone"`
		Images       []reviewPromptImage `json:"images"`
	}{
		ProjectTitle: input.Project.Title,
		ProjectTone:  input.Project.Tone,
		Images:       make([]reviewPromptImage, 0, len(input.Images)),
	}
	for _, image := range input.Images {
		if image == nil {
			continue
		}
		summary := reviewPromptImage{
			ID:           image.ID,
			FileName:     image.FileName,
			Status:       string(image.Status),
			Width:        image.Width,
			Height:       image.Height,
			UserDecision: image.UserDecision,
		}
		if !image.CreatedAt.IsZero() {
			summary.CreatedAt = image.CreatedAt.Format(time.RFC3339)
		}
		if image.Analysis != nil {
			summary.QualityScore = image.Analysis.QualityScore
			summary.PreservationScore = image.Analysis.PreservationScore
			summary.StoryScore = image.Analysis.StoryScore
			summary.Recommendation = image.Analysis.Recommendation
			summary.Reasons = image.Analysis.Reasons
			summary.CaptionSeeds = image.Analysis.CaptionSeeds
			summary.Issues = image.Analysis.Issues
			summary.Tags = append(summary.Tags, image.Analysis.DetectedContent.Scenes...)
			summary.Tags = append(summary.Tags, image.Analysis.DetectedContent.Objects...)
			summary.Tags = append(summary.Tags, image.Analysis.DetectedContent.Mood...)
			summary.Tags = append(summary.Tags, image.Analysis.DetectedContent.Tags...)
			summary.AlbumRole = image.Analysis.AlbumRole
		}
		payload.Images = append(payload.Images, summary)
	}
	raw, _ := json.Marshal(payload)
	return fmt.Sprintf(`
	请审查以下项目中的所有照片分析摘要，识别视觉上相似的照片并分组。输出严格 JSON。

	输出字段：
	{
	  "selectionStrategy": "整体筛选策略",
	  "groups": [
	    {
	      "id": "唯一分组标识",
	      "title": "给这组相似照片起一个短标题",
	      "imageIds": ["组内全部图片 id"],
	      "bestImageId": "最值得长期保存和入册的图片 id",
	      "keepImageIds": ["本组保留图片 id，通常 1 张，最多 2 张"],
	      "rejectImageIds": ["本组可自动弱化/不入册的相似替代图 id"],
	      "reason": "为什么这张最好",
	      "storyBeat": "它在相册故事线里的作用",
	      "socialImageId": "本组最适合朋友圈/小红书的图片 id"
	    }
	  ]
	}

	识别相似照片的方法：
	1. 相同或极度相近的 detected content（scenes、objects、mood、tags）——说明拍摄了同一场景或同一人物。
	2. 相近的创建时间（createdAt 接近）——说明可能是连拍。
	3. 相似的尺寸和比例——同一拍摄角度或裁剪。
	4. 相似的 captionSeeds 或 reasons——同样的内容被重复拍摄。
	5. 单张照片没有相似配对时，也要输出一个只包含自身的 group。

	要求：
	1. 你不是简单按分数排序，要综合清晰度、表情/主体、情绪、构图、稀缺性、能否多年后唤起回忆。
	2. 相似照片组里默认只留最好的一张，除非两张承担明显不同叙事作用。
	3. 不要把明显废片放进 keepImageIds。
	4. title、reason、storyBeat 要帮助用户快速理解为什么可以少看少选。
	5. 必须只使用输入中出现的 id。
	6. 每张图片都必须出现在某个 group 的 imageIds 中，不能遗漏。

	输入 JSON：
	%s
	`, string(raw))
}

func buildAlbumPrompt(input ai.AlbumNarrativeInput) string {
	payload := struct {
		ProjectTitle  string             `json:"projectTitle"`
		Description   string             `json:"description"`
		Location      string             `json:"location"`
		ThemeID       string             `json:"themeId"`
		Tone          string             `json:"tone"`
		SelectedCount int                `json:"selectedCount"`
		Images        []albumPromptImage `json:"images"`
	}{
		ProjectTitle:  input.Project.Title,
		Description:   input.Project.Description,
		Location:      input.Project.Location,
		ThemeID:       input.ThemeID,
		Tone:          input.Project.Tone,
		SelectedCount: input.SelectedCount,
		Images:        make([]albumPromptImage, 0, len(input.Images)),
	}
	for _, image := range input.Images {
		if image == nil {
			continue
		}
		item := albumPromptImage{
			ID:       image.ID,
			FileName: image.FileName,
			Status:   string(image.Status),
		}
		if image.Analysis != nil {
			item.QualityScore = image.Analysis.QualityScore
			item.PreservationScore = image.Analysis.PreservationScore
			item.StoryScore = image.Analysis.StoryScore
			item.Reasons = image.Analysis.Reasons
			item.CaptionSeeds = image.Analysis.CaptionSeeds
			item.GroupLabel = image.Analysis.SimilarGroupLabel
			item.GroupBest = image.Analysis.SimilarGroupBest
			item.AlbumRole = image.Analysis.AlbumRole
			item.SocialCaption = image.Analysis.SocialCaption
			item.Tags = append(item.Tags, image.Analysis.DetectedContent.Scenes...)
			item.Tags = append(item.Tags, image.Analysis.DetectedContent.Objects...)
			item.Tags = append(item.Tags, image.Analysis.DetectedContent.Mood...)
			item.Tags = append(item.Tags, image.Analysis.DetectedContent.Tags...)
		}
		payload.Images = append(payload.Images, item)
	}
	raw, _ := json.Marshal(payload)
	return fmt.Sprintf(`
	你是 Memoir/集忆 的艺术相册策划。请基于照片分析摘要，自动生成一本"可长期保存的人生经历记录"相册，同时给出适合朋友圈/小红书的图文草稿。输出严格 JSON，不要输出解释文字。

	这本相册需要有明确设计感：像一本私人摄影书、杂志专题或艺术家手工书，有开场、主视觉、细节观察、时间序列、拼贴、留白、索引和收束。不要把每一页都叫章节，也不要生成"第一章/第二章/章节 6"这类目录式标题。请把"社交图文"和"相册正文"完全分开，socialPosts 只作为导出分享页素材，不能混入 pages。

	主题风格说明：
	- film_travel：胶片旅行。暖色、底片索引、路途感，标题像旅途中写下的章节名。
	- warm_family：温暖家庭记忆。亲密、柔软、具体，像写给家人的长期记忆册。
	- editorial：纪实杂志。强网格、专题报道感，标题短促有力量。
	- minimal_gallery：极简画廊。大留白、克制、单张作品被认真观看。
	- nocturne：夜色电影。深色电影感、情绪镜头、适合夜景和安静片段。
	- botanical：植物手札。自然、纸张肌理、细节观察，像夹着叶片的旅行笔记。
	- postcard：明信片旅行。城市、远方、票据和邮戳感，轻快但真实。
	- archive：旧物档案。编号、证物、时间标本，把照片整理成可保存的档案。
	- cinematic_bw：黑白剧场。高反差、人物和关键瞬间优先，文字像幕间字幕。
	- summer_diary：夏日清单。清亮、日记式、适合朋友、天气、海边、街道和轻松片段。

	输出字段：
	{
	  "title": "相册标题",
	  "intro": "一段开篇导语",
	  "chapterTitles": ["可选的叙事段落名，不要用于每页标题"],
	  "ending": "结尾短句",
	  "designNotes": "整体设计意图、叙事线和视觉风格",
	  "pages": [
	    {
	      "pageType": "cover|opener|hero|detail|sequence|collage|gallery|pause|index|ending",
	      "layoutId": "cover_full_bleed|hero_story|single_photo_caption|diptych|triptych|mosaic|gallery_wall|timeline_ribbon|full_bleed_quote|scrapbook|contact_sheet|quote_break|ending_text",
	      "imageIds": ["images id"],
	      "title": "页面标题",
	      "body": "页面正文，真实、克制、有回忆感",
	      "caption": "页面短配文"
	    }
	  ],
	  "socialPosts": [
	    {
	      "platform": "moments|xiaohongshu",
	      "title": "朋友圈或小红书标题",
	      "hook": "第一眼吸引人的短句",
	      "body": "可直接发布的图文正文",
	      "imageIds": ["建议配图 id"],
	      "hashtags": ["#话题"]
	    }
	  ]
	}

	要求：
	1. 相册不是照片集合，而是一段人生经历的长期存档；优先真实、克制、具体、能多年后唤起回忆。
	2. 你要自动完成选图顺序、封面、页面角色、版式、正文和配文，用户只需要微调；生成结果要像"已经被设计师排过版"的相册。
	3. 必须把 themeId 转化成叙事和排版选择：同一组照片在不同主题下应有不同标题气质、页面节奏、留白程度和版式偏好。
	4. pageType 不是目录层级，而是设计页角色：cover=封面，opener=开场，hero=主视觉，detail=细节观察，sequence=时间序列，collage=拼贴，gallery=画廊墙，pause=留白停顿，index=照片索引，ending=收束。
	5. 版式语义：cover_full_bleed 做强封面；hero_story 做主视觉；single_photo_caption 做强记忆单图；diptych/triptych 做对照；mosaic 做碎片叙事；gallery_wall 做作品墙；timeline_ribbon 做时间推进；full_bleed_quote 做情绪高潮；scrapbook 做手账/票据/细节拼贴；contact_sheet 做索引；quote_break 可不放图，用一句有记忆感的短句停顿；ending_text 用于收束。
	6. 页面组织按"叙事节奏"而不是"第几章"：开场 -> 关键画面 -> 细节/对照/时间推进 -> 留白 -> 索引或收束；不要机械平均分组，不要把每张照片都塞进相册，宁可少而好。
	7. 每页 imageIds 只能使用输入里出现的 id；相似组只用最佳图，避免重复。
	8. 封面、第一张主视觉和结尾要形成完整回忆弧线；中间至少安排 1 个明显有设计感的页面（gallery_wall、timeline_ribbon、full_bleed_quote 或 scrapbook）。
	9. 页面标题要短，有文学性但不矫饰；禁止"第一章/第二章/第三章/章节 6/无题"这类标题；正文 30-90 字，像写给多年后的自己；caption 12-28 字，适合放在照片旁。
	10. 输出 5-14 个页面，照片少时页数可少；必须包含 cover 和 ending；至少使用 4 种不同 layoutId，且至少使用 4 种不同 pageType。
	11. 朋友圈文案和小红书文案必须明显不同，但都不能编造不存在的地点、人物或事件。
	12. socialPosts 给 2 条且顺序固定：第一条 platform="moments"，第二条 platform="xiaohongshu"；每条配 1-9 张图；socialPosts 不属于相册 pages。
	13. 朋友圈：像真实朋友动态，轻松、具体、不过度营销，body 80-160 字，可以有 1-2 个自然换行，不要堆话题。
	14. 小红书：标题更像笔记标题，hook 有收藏/分享欲，body 160-280 字，结构可以是 2-4 小段，分享"怎么筛选/为什么这样排版/适合怎样保存"，但不要教程腔；hashtags 4-8 个。
	15. 每条 socialPosts 的 imageIds 要优先选择最适合社交发布的 3-9 张图：封面/主视觉/细节/氛围各有代表，避免重复相似图。

	输入 JSON：
	%s
	`, string(raw))
}

func decodeJSONMessage(content any, out any) error {
	raw := strings.TrimSpace(fmt.Sprint(content))
	raw = trimCodeFences(raw)
	start := strings.Index(raw, "{")
	if start < 0 {
		start = strings.Index(raw, "[")
	}
	end := strings.LastIndex(raw, "}")
	if end < 0 {
		end = strings.LastIndex(raw, "]")
	}
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}
	return json.Unmarshal([]byte(raw), out)
}

func trimCodeFences(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	return strings.TrimSpace(raw)
}

func normalizeAnalysis(analysis *domain.ImageAnalysis) {
	analysis.QualityScore = clampScore(analysis.QualityScore)
	analysis.PreservationScore = clampScore(analysis.PreservationScore)
	analysis.StoryScore = clampScore(analysis.StoryScore)
	if analysis.Recommendation == "" {
		analysis.Recommendation = "review"
	}
	normalizeDetectedContent(&analysis.DetectedContent)
	for i := range analysis.EditSuggestions {
		normalizeEditSuggestion(&analysis.EditSuggestions[i])
	}
}

func normalizeDetectedContent(content *domain.DetectedContent) {
	if content == nil {
		return
	}
	if content.Scenes == nil {
		content.Scenes = []string{}
	}
	if content.Objects == nil {
		content.Objects = []string{}
	}
	if content.Mood == nil {
		content.Mood = []string{}
	}
	if content.Tags == nil {
		content.Tags = []string{}
	}
}

func normalizeEditSuggestion(suggestion *domain.ImageEditSuggestion) {
	suggestion.Type = strings.TrimSpace(strings.ToLower(suggestion.Type))
	suggestion.Strength = strings.TrimSpace(strings.ToLower(suggestion.Strength))
	suggestion.Execution = strings.TrimSpace(strings.ToLower(suggestion.Execution))
	switch suggestion.Execution {
	case "local", "local_approximation", "provider_generative":
	default:
		if suggestion.ProviderBacked {
			suggestion.Execution = "provider_generative"
		}
	}
	suggestion.ProviderBacked = suggestion.Execution == "provider_generative"
}

func normalizeReviewOrganization(review *ai.ReviewOrganization) {
	for i := range review.Groups {
		group := &review.Groups[i]
		group.ID = strings.TrimSpace(group.ID)
		group.BestImageID = strings.TrimSpace(group.BestImageID)
		if len(group.KeepImageIDs) == 0 && group.BestImageID != "" {
			group.KeepImageIDs = []string{group.BestImageID}
		}
		if group.SocialImageID == "" {
			group.SocialImageID = group.BestImageID
		}
	}
}

func clampScore(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
