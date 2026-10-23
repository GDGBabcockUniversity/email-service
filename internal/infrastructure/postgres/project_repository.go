package postgres

import (
	"context"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"gorm.io/gorm"
)

type ProjectRepository struct{ db *gorm.DB }

func NewProjectRepository(db *gorm.DB) *ProjectRepository { return &ProjectRepository{db: db} }

func (r *ProjectRepository) FindByID(ctx context.Context, id string) (*domain.Project, error) {
	project, err := gorm.G[domain.Project](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, repositoryError(err)
	}
	return &project, nil
}

func (r *ProjectRepository) FindAll(ctx context.Context) ([]domain.Project, error) {
	projects, err := gorm.G[domain.Project](r.db).Order("created_at, id").Find(ctx)
	return projects, repositoryError(err)
}

func (r *ProjectRepository) Save(ctx context.Context, project *domain.Project) error {
	return repositoryError(gorm.G[domain.Project](r.db).Create(ctx, project))
}

func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	result := r.db.WithContext(ctx).Model(&domain.Project{}).Where("id = ?", project.ID).
		Updates(map[string]any{"name": project.Name, "description": project.Description})
	return changed(result.RowsAffected, result.Error)
}

func (r *ProjectRepository) DeleteById(ctx context.Context, id string) error {
	return repositoryError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ?", id).Delete(&domain.Project{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("project_id = ?", id).Delete(&domain.Template{}).Error; err != nil {
			return err
		}
		return tx.Model(&domain.ApiKey{}).Where("project_id = ? AND revoked = ?", id, false).
			Updates(map[string]any{"revoked": true, "revoked_at": time.Now().UTC()}).Error
	}))
}

func (r *ProjectRepository) ExistsByNameAndUserID(ctx context.Context, name, userID string) (bool, error) {
	count, err := gorm.G[domain.Project](r.db).Where("name = ? AND user_id = ?", name, userID).Count(ctx, "id")
	return count > 0, repositoryError(err)
}

func (r *ProjectRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	count, err := gorm.G[domain.Project](r.db).Where("id = ?", id).Count(ctx, "id")
	return count > 0, repositoryError(err)
}

func (r *ProjectRepository) AddAudience(ctx context.Context, projectID string, audience []domain.Audience) error {
	for _, a := range audience {
		if err := r.db.WithContext(ctx).Create(&domain.Audience{ProjectID: projectID, Email: a.Email}).Error; err != nil {
			return err
		}
	}
	return nil
}
