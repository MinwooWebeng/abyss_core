package host

import (
	"sync"

	"github.com/google/uuid"
)

type SimplePathResolver struct {
	pathMap map[string]uuid.UUID
	mtx     *sync.Mutex
}

func NewSimplePathResolver() *SimplePathResolver {
	return &SimplePathResolver{
		pathMap: make(map[string]uuid.UUID),
		mtx:     new(sync.Mutex),
	}
}

func (r *SimplePathResolver) SetMapping(path string, dest uuid.UUID) {
	r.mtx.Lock()
	r.pathMap[path] = dest
	r.mtx.Unlock()
}

func (r *SimplePathResolver) DeleteMapping(path string) {
	r.mtx.Lock()
	delete(r.pathMap, path)
	r.mtx.Unlock()
}

func (r *SimplePathResolver) PathToSessionID(path string, _ string) (uuid.UUID, bool) {
	r.mtx.Lock()
	res, ok := r.pathMap[path]
	r.mtx.Unlock()

	return res, ok
}
