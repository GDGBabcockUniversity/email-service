// Package events tells dashboard streams when a project's emails change.
package events

import (
	"context"
	"sync"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

// EmailRepository wraps a repository and signals subscribers after every write.
// shortcut: in-process only, so it assumes the worker runs in the same binary as the API; use Redis pub/sub if they are ever split.
type EmailRepository struct {
	domain.EmailRepository
	mu   sync.Mutex
	subs map[string]map[chan struct{}]struct{}
}

func NewEmailRepository(inner domain.EmailRepository) *EmailRepository {
	return &EmailRepository{EmailRepository: inner, subs: map[string]map[chan struct{}]struct{}{}}
}

// Subscribe returns a channel that receives (coalesced) whenever the project's emails change.
func (r *EmailRepository) Subscribe(projectID string) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	r.mu.Lock()
	if r.subs[projectID] == nil {
		r.subs[projectID] = map[chan struct{}]struct{}{}
	}
	r.subs[projectID][ch] = struct{}{}
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		delete(r.subs[projectID], ch)
		if len(r.subs[projectID]) == 0 {
			delete(r.subs, projectID)
		}
		r.mu.Unlock()
	}
}

func (r *EmailRepository) notify(projectID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.subs[projectID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (r *EmailRepository) Save(ctx context.Context, email *domain.Email) error {
	err := r.EmailRepository.Save(ctx, email)
	if err == nil {
		r.notify(email.ProjectID)
	}
	return err
}

func (r *EmailRepository) SaveAll(ctx context.Context, emails []domain.Email) error {
	err := r.EmailRepository.SaveAll(ctx, emails)
	if err == nil {
		for _, e := range emails {
			r.notify(e.ProjectID)
		}
	}
	return err
}

func (r *EmailRepository) Update(ctx context.Context, email *domain.Email) error {
	err := r.EmailRepository.Update(ctx, email)
	if err == nil {
		r.notify(email.ProjectID)
	}
	return err
}
