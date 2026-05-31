package store

import "memoir/internal/domain"

// ProjectStore defines the persistence contract used by the app layer.
// FileStore is the current implementation, but other backends can satisfy
// this interface without changing the rest of the application.
type ProjectStore interface {
	ListProjects() ([]*domain.Project, error)
	GetProject(id string) (*domain.Project, error)
	CreateProject(project *domain.Project) error
	DeleteProject(id string) error
	UpdateProject(id string, mutate func(*domain.Project) error) error
	AddImage(projectID string, image *domain.ImageAsset) error
	DeleteImage(imageID string) (*domain.ImageAsset, error)
	UpdateImage(imageID string, mutate func(*domain.ImageAsset) error) (*domain.ImageAsset, error)
	FindImage(imageID string) (*domain.Project, *domain.ImageAsset, error)
	SetAlbum(projectID string, album *domain.Album) error
}
