package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"memoir/internal/domain"
)

func TestNewFileStoreRecoversInterruptedRunningTasks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state.json")
	started := time.Now().Add(-2 * time.Minute).UTC()

	initial := state{
		Projects: map[string]*domain.Project{
			"analysis": {
				ID:               "analysis",
				Title:            "analysis project",
				Status:           domain.ProjectStatusAnalyzing,
				AnalysisStatus:   "running",
				AnalysisProgress: 42,
				CurrentStep:      "分析 2 / 5",
				CreatedAt:        started,
				UpdatedAt:        started,
				Images: []*domain.ImageAsset{
					{ID: "image", ProjectID: "analysis", Status: domain.ImageStatusAnalyzing, CreatedAt: started, UpdatedAt: started},
				},
				ActiveTask: runningTask("task_analysis", domain.TaskTypeAnalysis, 42, started),
			},
			"album": {
				ID:               "album",
				Title:            "album project",
				Status:           domain.ProjectStatusGeneratingAlbum,
				AnalysisStatus:   "done",
				AnalysisProgress: 100,
				CurrentStep:      "生成相册草稿",
				CreatedAt:        started,
				UpdatedAt:        started,
				Images:           []*domain.ImageAsset{{ID: "album_image", ProjectID: "album", CreatedAt: started, UpdatedAt: started}},
				ActiveTask:       runningTask("task_album", domain.TaskTypeAlbumGeneration, 15, started),
			},
			"export": {
				ID:               "export",
				Title:            "export project",
				Status:           domain.ProjectStatusExporting,
				AnalysisStatus:   "done",
				AnalysisProgress: 100,
				CurrentStep:      "导出 pdf 相册",
				CreatedAt:        started,
				UpdatedAt:        started,
				Images:           []*domain.ImageAsset{{ID: "export_image", ProjectID: "export", CreatedAt: started, UpdatedAt: started}},
				Album:            &domain.Album{ID: "album_export", ProjectID: "export", Status: domain.AlbumStatusGenerated, CreatedAt: started, UpdatedAt: started},
				ActiveTask:       runningTask("task_export", domain.TaskTypeExport, 8, started),
			},
		},
	}
	raw, err := json.MarshalIndent(initial, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}

	analysis := recoveredProject(t, store, "analysis")
	if analysis.Status != domain.ProjectStatusReviewing || analysis.AnalysisStatus != "failed" {
		t.Fatalf("expected analysis project to be recoverable review state, got status=%s analysis=%s", analysis.Status, analysis.AnalysisStatus)
	}
	if analysis.CurrentStep != "分析中断，可重新发起" {
		t.Fatalf("unexpected analysis current step: %s", analysis.CurrentStep)
	}
	assertInterruptedTask(t, analysis.ActiveTask, 42)

	album := recoveredProject(t, store, "album")
	if album.Status != domain.ProjectStatusReviewing {
		t.Fatalf("expected album generation project to return to review state, got %s", album.Status)
	}
	if album.CurrentStep != "相册生成中断，可重新生成" {
		t.Fatalf("unexpected album current step: %s", album.CurrentStep)
	}
	assertInterruptedTask(t, album.ActiveTask, 15)

	export := recoveredProject(t, store, "export")
	if export.Status != domain.ProjectStatusEditing {
		t.Fatalf("expected export project to return to editing state, got %s", export.Status)
	}
	if export.CurrentStep != "导出中断，可重新导出" {
		t.Fatalf("unexpected export current step: %s", export.CurrentStep)
	}
	assertInterruptedTask(t, export.ActiveTask, 8)

	var persisted state
	persistedRaw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read recovered state: %v", err)
	}
	if err := json.Unmarshal(persistedRaw, &persisted); err != nil {
		t.Fatalf("decode recovered state: %v", err)
	}
	if persisted.Projects["analysis"].ActiveTask.Status != domain.TaskStatusInterrupted {
		t.Fatalf("expected recovered task state to be persisted, got %s", persisted.Projects["analysis"].ActiveTask.Status)
	}
}

func TestNewFileStoreNormalizesLegacyNullCollections(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state.json")
	now := time.Now().UTC()

	raw := []byte(`{
  "projects": {
    "legacy": {
      "id": "legacy",
      "title": "legacy project",
      "description": "",
      "tone": "film",
      "themeId": "film_travel",
      "status": "editing",
      "analysisStatus": "done",
      "analysisProgress": 100,
      "currentStep": "legacy",
      "createdAt": "` + now.Format(time.RFC3339Nano) + `",
      "updatedAt": "` + now.Format(time.RFC3339Nano) + `",
      "images": [
        {
          "id": "img_1",
          "projectId": "legacy",
          "fileName": "legacy.jpg",
          "mimeType": "image/jpeg",
          "fileSize": 123,
          "width": 1200,
          "height": 800,
          "originalUrl": "/media/original.jpg",
          "thumbnailUrl": "/media/thumb.jpg",
          "averageHash": "abc",
          "status": "keep",
          "createdAt": "` + now.Format(time.RFC3339Nano) + `",
          "updatedAt": "` + now.Format(time.RFC3339Nano) + `",
          "analysis": {
            "qualityScore": 80,
            "preservationScore": 82,
            "storyScore": 77,
            "recommendation": "keep",
            "reasons": null,
            "detectedContent": {
              "peopleCount": 1,
              "scenes": null,
              "objects": null,
              "mood": null,
              "tags": null
            },
            "metrics": {
              "aspectRatio": 1.5,
              "brightness": 0.4,
              "contrast": 0.3,
              "sharpness": 0.6,
              "averageHash": "abc",
              "fileSize": 123,
              "width": 1200,
              "height": 800
            },
            "issues": null,
            "cropSuggestions": null,
            "captionSeeds": null,
            "editSuggestions": null,
            "modelVersion": "legacy",
            "promptVersion": "legacy",
            "completedAt": "` + now.Format(time.RFC3339Nano) + `"
          }
        }
      ],
      "album": {
        "id": "alb_1",
        "projectId": "legacy",
        "themeId": "film_travel",
        "title": "legacy album",
        "intro": "intro",
        "status": "generated",
        "version": 1,
        "pages": [
          {
            "id": "page_1",
            "order": 1,
            "pageType": "cover",
            "layoutId": "cover_full_bleed",
            "imageIds": null,
            "title": "cover",
            "body": "",
            "caption": ""
          }
        ],
        "socialPosts": [
          {
            "title": "post",
            "body": "body",
            "imageIds": null,
            "hashtags": null
          }
        ],
        "editHistory": [
          {
            "title": "snap",
            "intro": "intro",
            "themeId": "film_travel",
            "pages": [
              {
                "id": "snap_page",
                "order": 1,
                "pageType": "chapter",
                "layoutId": "single_photo_caption",
                "imageIds": null,
                "title": "title",
                "body": "",
                "caption": ""
              }
            ],
            "version": 1,
            "reason": "legacy",
            "createdAt": "` + now.Format(time.RFC3339Nano) + `"
          }
        ],
        "redoStack": null,
        "createdAt": "` + now.Format(time.RFC3339Nano) + `",
        "updatedAt": "` + now.Format(time.RFC3339Nano) + `"
      },
      "taskHistory": null
    }
  }
}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write legacy state: %v", err)
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}

	project, err := store.GetProject("legacy")
	if err != nil {
		t.Fatalf("get legacy project: %v", err)
	}
	if project.Images == nil || project.TaskHistory == nil {
		t.Fatalf("expected normalized project slices, got images=%#v taskHistory=%#v", project.Images, project.TaskHistory)
	}
	if project.Place != nil {
		t.Fatalf("expected legacy project without place to load with nil place, got %#v", project.Place)
	}
	if project.Images[0].Analysis == nil {
		t.Fatalf("expected legacy analysis to survive")
	}
	analysis := project.Images[0].Analysis
	if analysis.Reasons == nil || analysis.Issues == nil || analysis.CropSuggestions == nil || analysis.CaptionSeeds == nil || analysis.EditSuggestions == nil {
		t.Fatalf("expected normalized analysis slices, got %#v", analysis)
	}
	if analysis.DetectedContent.Scenes == nil || analysis.DetectedContent.Objects == nil || analysis.DetectedContent.Mood == nil || analysis.DetectedContent.Tags == nil {
		t.Fatalf("expected normalized detected content, got %#v", analysis.DetectedContent)
	}
	if project.Album == nil || project.Album.Pages == nil || project.Album.SocialPosts == nil || project.Album.EditHistory == nil || project.Album.RedoStack == nil {
		t.Fatalf("expected normalized album slices, got %#v", project.Album)
	}
	if project.Album.Pages[0].ImageIDs == nil {
		t.Fatalf("expected normalized page image ids")
	}
	if project.Album.SocialPosts[0].ImageIDs == nil || project.Album.SocialPosts[0].Hashtags == nil {
		t.Fatalf("expected normalized social post arrays, got %#v", project.Album.SocialPosts[0])
	}
	if project.Album.EditHistory[0].Pages[0].ImageIDs == nil {
		t.Fatalf("expected normalized snapshot page image ids")
	}
}

func runningTask(id string, taskType domain.TaskType, progress int, now time.Time) *domain.ProjectTask {
	return &domain.ProjectTask{
		ID:        id,
		Type:      taskType,
		Status:    domain.TaskStatusRunning,
		Progress:  progress,
		Message:   "running",
		StartedAt: now,
		UpdatedAt: now,
	}
}

func recoveredProject(t *testing.T, store *FileStore, projectID string) *domain.Project {
	t.Helper()
	project, err := store.GetProject(projectID)
	if err != nil {
		t.Fatalf("get recovered project %s: %v", projectID, err)
	}
	if project.LastError == "" || !strings.Contains(project.LastError, "中断") {
		t.Fatalf("expected recovery error on project %s, got %q", projectID, project.LastError)
	}
	return project
}

func assertInterruptedTask(t *testing.T, task *domain.ProjectTask, progress int) {
	t.Helper()
	if task == nil {
		t.Fatalf("expected active task")
	}
	if task.Status != domain.TaskStatusInterrupted {
		t.Fatalf("expected interrupted task, got %s", task.Status)
	}
	if task.Progress != progress {
		t.Fatalf("expected progress %d, got %d", progress, task.Progress)
	}
	if task.Error == "" || !strings.Contains(task.Error, "中断") {
		t.Fatalf("expected interrupted task error, got %q", task.Error)
	}
	if task.CompletedAt.IsZero() {
		t.Fatalf("expected interrupted task completion timestamp")
	}
}
