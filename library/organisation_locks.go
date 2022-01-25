package library

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type organisationLocks struct {
	// To allow multiple organisation (aka file moving) threads to run concurrently we need a way of locking a _titleID_ for each thread.
	lockedTitleIDs sync.Map
}

func (org *organisationLocks) Lock(titleID uint64) {
	titleID = titleID & 0xFFFFFFFFFFFFE000 // Strip back to base titleID
	value, _ := org.lockedTitleIDs.LoadOrStore(titleID, &sync.Mutex{})
	mtx := value.(*sync.Mutex)
	mtx.Lock()
}

func (org *organisationLocks) Unlock(titleID uint64) {

	titleID = titleID & 0xFFFFFFFFFFFFE000 // Strip back to base titleID
	value, ok := org.lockedTitleIDs.Load(titleID)
	if !ok {
		log.Fatal().Msg("Unlock received for missing titleID")
	}
	if value == nil {
		log.Fatal().Msg("Unlock received a nil")
	}
	mtx := value.(*sync.Mutex)
	mtx.Unlock()
}
