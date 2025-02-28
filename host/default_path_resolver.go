package host

import (
	"sync"

	"github.com/google/uuid"
)

type DefaultPathResolver struct {
	pathMap map[string]uuid.UUID
	mtx     *sync.Mutex
}

func NewDefaultPathResolver() *DefaultPathResolver {
	return &DefaultPathResolver{
		pathMap: make(map[string]uuid.UUID),
		mtx:     new(sync.Mutex),
	}
}

func (r *DefaultPathResolver) SetMapping(path string, dest uuid.UUID) {
	if dest == uuid.Nil {
		r.mtx.Lock()
		delete(r.pathMap, path)
		r.mtx.Unlock()
		return
	}

	r.mtx.Lock()
	r.pathMap[path] = dest
	r.mtx.Unlock()
}

func (r *DefaultPathResolver) PathToSessionID(path string, _ string) (uuid.UUID, bool) {
	r.mtx.Lock()
	res, ok := r.pathMap[path]
	r.mtx.Unlock()

	return res, ok
}
