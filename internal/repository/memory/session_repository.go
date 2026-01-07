package memory

import (
	"ai-notetaking-be/pkg/store"
	"time"

	"github.com/patrickmn/go-cache"
)

type SessionRepository struct {
	cache *cache.Cache
}

func NewSessionRepository() *SessionRepository {
	// Create a cache with a default expiration time of 1 hour, and which
	// purges expired items every 10 minutes
	c := cache.New(1*time.Hour, 10*time.Minute)
	return &SessionRepository{
		cache: c,
	}
}

func (r *SessionRepository) Save(session *store.Session) {
	r.cache.Set(session.ID, session, cache.DefaultExpiration)
}

func (r *SessionRepository) Get(sessionID string) (*store.Session, bool) {
	if x, found := r.cache.Get(sessionID); found {
		return x.(*store.Session), true
	}
	return nil, false
}

func (r *SessionRepository) Delete(sessionID string) {
	r.cache.Delete(sessionID)
}
