package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"memoir/internal/domain"
)

// FileStore persists Memoir state in a JSON file.
type FileStore struct {
	mu   sync.RWMutex
	path string
	data *state
}

var _ ProjectStore = (*FileStore)(nil)

type state struct {
	Projects map[string]*domain.Project `json:"projects"`
}

// NewFileStore loads or creates the JSON store at path.
func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{
		path: path,
		data: &state{Projects: map[string]*domain.Project{}},
	}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

// ListProjects returns all projects sorted by update time.
func (s *FileStore) ListProjects() ([]*domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*domain.Project, 0, len(s.data.Projects))
	for _, project := range s.data.Projects {
		items = append(items, cloneProject(project))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

// GetProject returns a single project by id.
func (s *FileStore) GetProject(id string) (*domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.data.Projects[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return cloneProject(project), nil
}

// CreateProject stores a new project.
func (s *FileStore) CreateProject(project *domain.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.Projects[project.ID]; ok {
		return errors.New("project already exists")
	}
	s.data.Projects[project.ID] = cloneProject(project)
	return s.persistLocked()
}

// DeleteProject removes a project and its images.
func (s *FileStore) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data.Projects, id)
	return s.persistLocked()
}

// UpdateProject mutates a project inside the store.
func (s *FileStore) UpdateProject(id string, mutate func(*domain.Project) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.data.Projects[id]
	if !ok {
		return os.ErrNotExist
	}
	if err := mutate(project); err != nil {
		return err
	}
	project.UpdatedAt = time.Now()
	return s.persistLocked()
}

// AddImage appends an image to a project.
func (s *FileStore) AddImage(projectID string, image *domain.ImageAsset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.data.Projects[projectID]
	if !ok {
		return os.ErrNotExist
	}
	project.Images = append(project.Images, cloneImage(image))
	project.UpdatedAt = time.Now()
	return s.persistLocked()
}

// DeleteImage removes one image from its project.
func (s *FileStore) DeleteImage(imageID string) (*domain.ImageAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, project := range s.data.Projects {
		for index, image := range project.Images {
			if image.ID != imageID {
				continue
			}
			deleted := cloneImage(image)
			project.Images = append(project.Images[:index], project.Images[index+1:]...)
			if project.Album != nil {
				project.Album.Pages = removeImageFromPages(project.Album.Pages, imageID)
				project.Album.UpdatedAt = time.Now()
			}
			if len(project.Images) == 0 {
				project.Status = domain.ProjectStatusDraft
				project.AnalysisStatus = "idle"
				project.AnalysisProgress = 0
				project.CurrentStep = "等待导入"
				project.Album = nil
			}
			project.UpdatedAt = time.Now()
			if err := s.persistLocked(); err != nil {
				return nil, err
			}
			return deleted, nil
		}
	}
	return nil, os.ErrNotExist
}

// UpdateImage mutates a single image in a project.
func (s *FileStore) UpdateImage(imageID string, mutate func(*domain.ImageAsset) error) (*domain.ImageAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, project := range s.data.Projects {
		for _, image := range project.Images {
			if image.ID == imageID {
				if err := mutate(image); err != nil {
					return nil, err
				}
				image.UpdatedAt = time.Now()
				project.UpdatedAt = time.Now()
				if err := s.persistLocked(); err != nil {
					return nil, err
				}
				return cloneImage(image), nil
			}
		}
	}
	return nil, os.ErrNotExist
}

// FindImage locates an image and returns its project and image.
func (s *FileStore) FindImage(imageID string) (*domain.Project, *domain.ImageAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, project := range s.data.Projects {
		for _, image := range project.Images {
			if image.ID == imageID {
				return cloneProject(project), cloneImage(image), nil
			}
		}
	}
	return nil, nil, os.ErrNotExist
}

// SetAlbum stores a generated album on a project.
func (s *FileStore) SetAlbum(projectID string, album *domain.Album) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.data.Projects[projectID]
	if !ok {
		return os.ErrNotExist
	}
	project.Album = cloneAlbum(album)
	project.UpdatedAt = time.Now()
	return s.persistLocked()
}

func (s *FileStore) load() error {
	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	var loaded state
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return err
	}
	if loaded.Projects == nil {
		loaded.Projects = map[string]*domain.Project{}
	}
	changed := normalizeProjects(loaded.Projects)
	s.data = &loaded
	changed = recoverInterruptedTasks(s.data.Projects) || changed
	if changed {
		return s.persistLocked()
	}
	return nil
}

func (s *FileStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func cloneProject(project *domain.Project) *domain.Project {
	if project == nil {
		return nil
	}
	raw, _ := json.Marshal(project)
	var out domain.Project
	_ = json.Unmarshal(raw, &out)
	normalizeProject(&out)
	return &out
}

func cloneImage(image *domain.ImageAsset) *domain.ImageAsset {
	if image == nil {
		return nil
	}
	raw, _ := json.Marshal(image)
	var out domain.ImageAsset
	_ = json.Unmarshal(raw, &out)
	normalizeImage(&out)
	return &out
}

func cloneAlbum(album *domain.Album) *domain.Album {
	if album == nil {
		return nil
	}
	raw, _ := json.Marshal(album)
	var out domain.Album
	_ = json.Unmarshal(raw, &out)
	normalizeAlbum(&out)
	return &out
}

func normalizeProjects(projects map[string]*domain.Project) bool {
	changed := false
	for key, project := range projects {
		if project == nil {
			delete(projects, key)
			changed = true
			continue
		}
		if normalizeProject(project) {
			changed = true
		}
	}
	return changed
}

func normalizeProject(project *domain.Project) bool {
	if project == nil {
		return false
	}
	changed := false
	if project.Images == nil {
		project.Images = []*domain.ImageAsset{}
		changed = true
	}
	for index, image := range project.Images {
		if image == nil {
			project.Images = append(project.Images[:index], project.Images[index+1:]...)
			changed = true
			return normalizeProject(project) || changed
		}
		if normalizeImage(image) {
			changed = true
		}
	}
	if project.TaskHistory == nil {
		project.TaskHistory = []*domain.ProjectTask{}
		changed = true
	}
	for index, task := range project.TaskHistory {
		if task == nil {
			project.TaskHistory = append(project.TaskHistory[:index], project.TaskHistory[index+1:]...)
			changed = true
			return normalizeProject(project) || changed
		}
	}
	if project.Album != nil && normalizeAlbum(project.Album) {
		changed = true
	}
	return changed
}

func normalizeImage(image *domain.ImageAsset) bool {
	if image == nil {
		return false
	}
	changed := false
	if image.EditHistory == nil {
		image.EditHistory = []*domain.ImageSnapshot{}
		changed = true
	}
	for index, snapshot := range image.EditHistory {
		if snapshot == nil {
			image.EditHistory = append(image.EditHistory[:index], image.EditHistory[index+1:]...)
			changed = true
			return normalizeImage(image) || changed
		}
	}
	if image.Analysis != nil && normalizeAnalysis(image.Analysis) {
		changed = true
	}
	return changed
}

func normalizeAnalysis(analysis *domain.ImageAnalysis) bool {
	if analysis == nil {
		return false
	}
	changed := false
	if analysis.Reasons == nil {
		analysis.Reasons = []string{}
		changed = true
	}
	if analysis.Issues == nil {
		analysis.Issues = []domain.ImageIssue{}
		changed = true
	}
	if analysis.CropSuggestions == nil {
		analysis.CropSuggestions = []domain.CropSuggestion{}
		changed = true
	}
	if analysis.CaptionSeeds == nil {
		analysis.CaptionSeeds = []string{}
		changed = true
	}
	if analysis.EditSuggestions == nil {
		analysis.EditSuggestions = []domain.ImageEditSuggestion{}
		changed = true
	}
	if analysis.DetectedContent.Scenes == nil {
		analysis.DetectedContent.Scenes = []string{}
		changed = true
	}
	if analysis.DetectedContent.Objects == nil {
		analysis.DetectedContent.Objects = []string{}
		changed = true
	}
	if analysis.DetectedContent.Mood == nil {
		analysis.DetectedContent.Mood = []string{}
		changed = true
	}
	if analysis.DetectedContent.Tags == nil {
		analysis.DetectedContent.Tags = []string{}
		changed = true
	}
	return changed
}

func normalizeAlbum(album *domain.Album) bool {
	if album == nil {
		return false
	}
	changed := false
	if album.Pages == nil {
		album.Pages = []*domain.AlbumPage{}
		changed = true
	}
	for index, page := range album.Pages {
		if page == nil {
			album.Pages = append(album.Pages[:index], album.Pages[index+1:]...)
			changed = true
			return normalizeAlbum(album) || changed
		}
		if normalizeAlbumPage(page) {
			changed = true
		}
	}
	if album.SocialPosts == nil {
		album.SocialPosts = []*domain.AlbumSocialPost{}
		changed = true
	}
	for index, post := range album.SocialPosts {
		if post == nil {
			album.SocialPosts = append(album.SocialPosts[:index], album.SocialPosts[index+1:]...)
			changed = true
			return normalizeAlbum(album) || changed
		}
		if normalizeAlbumSocialPost(post) {
			changed = true
		}
	}
	if album.EditHistory == nil {
		album.EditHistory = []*domain.AlbumSnapshot{}
		changed = true
	}
	for index, snapshot := range album.EditHistory {
		if snapshot == nil {
			album.EditHistory = append(album.EditHistory[:index], album.EditHistory[index+1:]...)
			changed = true
			return normalizeAlbum(album) || changed
		}
		if normalizeAlbumSnapshot(snapshot) {
			changed = true
		}
	}
	if album.RedoStack == nil {
		album.RedoStack = []*domain.AlbumSnapshot{}
		changed = true
	}
	for index, snapshot := range album.RedoStack {
		if snapshot == nil {
			album.RedoStack = append(album.RedoStack[:index], album.RedoStack[index+1:]...)
			changed = true
			return normalizeAlbum(album) || changed
		}
		if normalizeAlbumSnapshot(snapshot) {
			changed = true
		}
	}
	return changed
}

func normalizeAlbumPage(page *domain.AlbumPage) bool {
	if page == nil {
		return false
	}
	if page.ImageIDs == nil {
		page.ImageIDs = []string{}
		return true
	}
	return false
}

func normalizeAlbumSocialPost(post *domain.AlbumSocialPost) bool {
	if post == nil {
		return false
	}
	changed := false
	if post.ImageIDs == nil {
		post.ImageIDs = []string{}
		changed = true
	}
	if post.Hashtags == nil {
		post.Hashtags = []string{}
		changed = true
	}
	return changed
}

func normalizeAlbumSnapshot(snapshot *domain.AlbumSnapshot) bool {
	if snapshot == nil {
		return false
	}
	changed := false
	if snapshot.Pages == nil {
		snapshot.Pages = []*domain.AlbumPage{}
		changed = true
	}
	for index, page := range snapshot.Pages {
		if page == nil {
			snapshot.Pages = append(snapshot.Pages[:index], snapshot.Pages[index+1:]...)
			changed = true
			return normalizeAlbumSnapshot(snapshot) || changed
		}
		if normalizeAlbumPage(page) {
			changed = true
		}
	}
	return changed
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func recoverInterruptedTasks(projects map[string]*domain.Project) bool {
	changed := false
	for _, project := range projects {
		if project == nil || project.ActiveTask == nil || project.ActiveTask.Status != domain.TaskStatusRunning {
			continue
		}
		now := time.Now()
		project.ActiveTask.Status = domain.TaskStatusInterrupted
		project.ActiveTask.Progress = minInt(project.ActiveTask.Progress, 99)
		project.ActiveTask.Error = "任务在服务重启或进程退出时中断"
		project.ActiveTask.UpdatedAt = now
		if project.ActiveTask.CompletedAt.IsZero() {
			project.ActiveTask.CompletedAt = now
		}
		project.LastError = project.ActiveTask.Error
		switch project.ActiveTask.Type {
		case domain.TaskTypeAnalysis:
			if len(project.Images) == 0 {
				project.Status = domain.ProjectStatusDraft
				project.AnalysisStatus = "idle"
				project.AnalysisProgress = 0
				project.CurrentStep = "等待导入"
			} else {
				project.Status = domain.ProjectStatusReviewing
				project.AnalysisStatus = "failed"
				project.CurrentStep = "分析中断，可重新发起"
			}
		case domain.TaskTypeAlbumGeneration:
			if project.Album != nil {
				project.Status = domain.ProjectStatusEditing
				project.CurrentStep = "相册生成中断，可重新生成"
			} else {
				project.Status = domain.ProjectStatusReviewing
				project.CurrentStep = "相册生成中断，可重新生成"
			}
		case domain.TaskTypeExport:
			if project.Album != nil {
				project.Status = domain.ProjectStatusEditing
			} else {
				project.Status = domain.ProjectStatusReviewing
			}
			project.CurrentStep = "导出中断，可重新导出"
		}
		project.UpdatedAt = now
		changed = true
	}
	return changed
}

func removeImageFromPages(pages []*domain.AlbumPage, imageID string) []*domain.AlbumPage {
	out := make([]*domain.AlbumPage, 0, len(pages))
	for _, page := range pages {
		ids := make([]string, 0, len(page.ImageIDs))
		for _, id := range page.ImageIDs {
			if id != imageID {
				ids = append(ids, id)
			}
		}
		page.ImageIDs = ids
		if page.PageType == "cover" && len(page.ImageIDs) == 0 {
			continue
		}
		out = append(out, page)
	}
	return out
}
