package postgres

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"gorm.io/gorm"
)

type AudienceRepository struct {
	db *gorm.DB
}

func NewAudienceRepository(db *gorm.DB) *AudienceRepository {
	return &AudienceRepository{db: db}
}

func (r *AudienceRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.Audience, error) {

	audience, err := gorm.G[domain.Audience](r.db).Where("project_id = ?", projectID).Find(ctx)

	if err != nil {
		return []domain.Audience{}, err
	}

	return audience, nil
}

// FindSubscribedByProjectID returns only the recipients who have not unsubscribed.
func (r *AudienceRepository) FindSubscribedByProjectID(ctx context.Context, projectID string) ([]domain.Audience, error) {

	audience, err := gorm.G[domain.Audience](r.db).Where("project_id = ? AND subscribed", projectID).Find(ctx)

	if err != nil {
		return []domain.Audience{}, repositoryError(err)
	}

	return audience, nil
}

func (r *AudienceRepository) FindByID(ctx context.Context, id string) (*domain.Audience, error) {

	audience, err := gorm.G[domain.Audience](r.db).Where("id = ?", id).First(ctx)

	if err != nil {
		return nil, err
	}

	return &audience, nil
}

// FindByEmailAndProjectID locates one project's subscription for an address,
// matching case-insensitively so "A@x.com" and "a@x.com" are one subscription.
func (r *AudienceRepository) FindByEmailAndProjectID(ctx context.Context, email, projectID string) (*domain.Audience, error) {

	audience, err := gorm.G[domain.Audience](r.db).Where("project_id = ? AND lower(email) = lower(?)", projectID, email).First(ctx)

	if err != nil {
		return nil, repositoryError(err)
	}

	return &audience, nil
}

// FindByUnsubscribeToken looks a subscription up by the stored hash of its token.
func (r *AudienceRepository) FindByUnsubscribeToken(ctx context.Context, tokenHash string) (*domain.Audience, error) {

	if tokenHash == "" {
		return nil, domain.ErrNotFound
	}

	audience, err := gorm.G[domain.Audience](r.db).Where("unsubscribe_token = ?", tokenHash).First(ctx)

	if err != nil {
		return nil, repositoryError(err)
	}

	return &audience, nil
}

func (r *AudienceRepository) DeleteByID(ctx context.Context, id string) error {

	rowsAffected, err := gorm.G[domain.Audience](r.db).Where("id = ?", id).Delete(ctx)

	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	if err != nil {
		return err
	}

	return nil
}

func (r *AudienceRepository) SaveAll(ctx context.Context, audience []domain.Audience) error {

	err := gorm.G[domain.Audience](r.db).CreateInBatches(ctx, &audience, 100)

	if err != nil {
		return err
	}

	return nil
}

// SetSubscribed flips one subscription's consent without touching the row's token.
func (r *AudienceRepository) SetSubscribed(ctx context.Context, id string, subscribed bool) error {

	rows, err := gorm.G[domain.Audience](r.db).Where("id = ?", id).Update(ctx, "subscribed", subscribed)

	return changed(int64(rows), err)
}

// SetUnsubscribeToken backfills the stored token hash for an existing subscription.
func (r *AudienceRepository) SetUnsubscribeToken(ctx context.Context, id, tokenHash string) error {

	rows, err := gorm.G[domain.Audience](r.db).Where("id = ?", id).Update(ctx, "unsubscribe_token", tokenHash)

	return changed(int64(rows), err)
}
