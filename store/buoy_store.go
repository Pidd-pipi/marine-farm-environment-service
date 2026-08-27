package store

import (
	"sort"

	"example.com/marine-farm-environment-service/domain"
)

// BuoyStore is the repository of monitoring buoys.
type BuoyStore struct{ s *Store }

// Create persists a new buoy.
func (b *BuoyStore) Create(buoy *domain.Buoy) error {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	b.s.state.Buoys = append(b.s.state.Buoys, *buoy)
	return b.s.saveLocked()
}

// Get returns a copy of the buoy with the given id.
func (b *BuoyStore) Get(id string) (domain.Buoy, error) {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	for i := range b.s.state.Buoys {
		if b.s.state.Buoys[i].ID == id {
			return cloneBuoy(b.s.state.Buoys[i]), nil
		}
	}
	return domain.Buoy{}, domain.NotFound("buoy", id)
}

// List returns all buoys sorted by creation time.
func (b *BuoyStore) List() []domain.Buoy {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	out := make([]domain.Buoy, 0, len(b.s.state.Buoys))
	for _, buoy := range b.s.state.Buoys {
		out = append(out, cloneBuoy(buoy))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// ListByZone returns the buoys belonging to a zone.
func (b *BuoyStore) ListByZone(zoneID string) []domain.Buoy {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	var out []domain.Buoy
	for i := range b.s.state.Buoys {
		if b.s.state.Buoys[i].ZoneID == zoneID {
			out = append(out, cloneBuoy(b.s.state.Buoys[i]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// Update persists a modified buoy.
func (b *BuoyStore) Update(buoy *domain.Buoy) error {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	for i := range b.s.state.Buoys {
		if b.s.state.Buoys[i].ID == buoy.ID {
			b.s.state.Buoys[i] = *buoy
			return b.s.saveLocked()
		}
	}
	return domain.NotFound("buoy", buoy.ID)
}

// CountByZone returns the number of buoys of a zone.
func (b *BuoyStore) CountByZone(zoneID string) int {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	n := 0
	for i := range b.s.state.Buoys {
		if b.s.state.Buoys[i].ZoneID == zoneID {
			n++
		}
	}
	return n
}

// Count returns the total number of buoys.
func (b *BuoyStore) Count() int {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	return len(b.s.state.Buoys)
}
