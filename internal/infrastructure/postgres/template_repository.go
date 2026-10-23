package postgres

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"gorm.io/gorm"
)

type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) FindByID(ctx context.Context, id string) (*domain.Template, error) {
	template, err := gorm.G[domain.Template](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, repositoryError(err)
	}
	return &template, nil
}

func (r *TemplateRepository) FindByName(ctx context.Context, name string) (*domain.Template, error) {
	template, err := gorm.G[domain.Template](r.db).Where("name = ?", name).First(ctx)
	if err != nil {
		return nil, repositoryError(err)
	}
	return &template, nil
}

func (r *TemplateRepository) FindAll(ctx context.Context) ([]domain.Template, error) {
	templates, err := gorm.G[domain.Template](r.db).Order("created_at, id").Find(ctx)
	return templates, repositoryError(err)
}

func (r *TemplateRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.Template, error) {
	templates, err := gorm.G[domain.Template](r.db).Where("project_id = ?", projectID).Order("created_at, id").Find(ctx)
	return templates, repositoryError(err)
}

func (r *TemplateRepository) Save(ctx context.Context, template *domain.Template) error {
	return repositoryError(gorm.G[domain.Template](r.db).Create(ctx, template))
}

func (r *TemplateRepository) Update(ctx context.Context, template *domain.Template) error {
	result := r.db.WithContext(ctx).Model(template).Select("Name", "Subject", "Data", "Content").Updates(template)
	return changed(result.RowsAffected, result.Error)
}

func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	count, err := gorm.G[domain.Template](r.db).Where("id = ?", id).Delete(ctx)
	return changed(int64(count), err)
}
