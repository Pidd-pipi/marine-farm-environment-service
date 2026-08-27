package store

import (
	"sort"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// ZoneStore is the repository of farm zones.
type ZoneStore struct{ s *Store }

// Create persists a new farm zone and returns the stored copy.
func (z *ZoneStore) Create(zone *domain.FarmZone) error {
	z.s.mu.Lock()
	defer z.s.mu.Unlock()
	z.s.state.Zones = append(z.s.state.Zones, *zone)
	return z.s.saveLocked()
}

// Get returns a copy of the zone with the given id.
func (z *ZoneStore) Get(id string) (domain.FarmZone, error) {
	z.s.mu.RLock()
	defer z.s.mu.RUnlock()
	for i := range z.s.state.Zones {
		if z.s.state.Zones[i].ID == id {
			return z.s.state.Zones[i], nil
		}
	}
	return domain.FarmZone{}, domain.NotFound("farm zone", id)
}

// List returns all zones sorted by creation time.
func (z *ZoneStore) List() []domain.FarmZone {
	z.s.mu.RLock()
	defer z.s.mu.RUnlock()
	out := make([]domain.FarmZone, 0, len(z.s.state.Zones))
	for i := range z.s.state.Zones {
		out = append(out, z.s.state.Zones[i])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// Update persists a modified zone. The zone must already exist.
func (z *ZoneStore) Update(zone *domain.FarmZone) error {
	z.s.mu.Lock()
	defer z.s.mu.Unlock()
	for i := range z.s.state.Zones {
		if z.s.state.Zones[i].ID == zone.ID {
			z.s.state.Zones[i] = *zone
			return z.s.saveLocked()
		}
	}
	return domain.NotFound("farm zone", zone.ID)
}

// Count returns the number of zones.
func (z *ZoneStore) Count() int {
	z.s.mu.RLock()
	defer z.s.mu.RUnlock()
	return len(z.s.state.Zones)
}

// StatusSince returns the recorded status-since timestamp of a zone.
func (z *ZoneStore) StatusSince(id string) (time.Time, error) {
	zone, err := z.Get(id)
	if err != nil {
		return time.Time{}, err
	}
	return zone.StatusSince, nil
}
