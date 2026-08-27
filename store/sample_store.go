package store

import (
	"sort"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// SampleStore is the repository of water-quality samples.
type SampleStore struct{ s *Store }

// Create persists a new sample and trims the per-buoy history to the
// configured cap.
func (b *SampleStore) Create(sample *domain.WaterSample, maxPerBuoy int) error {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	b.s.state.Samples = append(b.s.state.Samples, *sample)
	if maxPerBuoy > 0 {
		b.trimLocked(sample.BuoyID, maxPerBuoy)
	}
	return b.s.saveLocked()
}

// trimLocked removes the oldest samples of a buoy beyond the cap.
func (b *SampleStore) trimLocked(buoyID string, maxPerBuoy int) {
	var kept []domain.WaterSample
	count := 0
	for i := len(b.s.state.Samples) - 1; i >= 0; i-- {
		s := b.s.state.Samples[i]
		if s.BuoyID == buoyID {
			if count >= maxPerBuoy {
				continue
			}
			count++
		}
		kept = append([]domain.WaterSample{s}, kept...)
	}
	b.s.state.Samples = kept
}

// Get returns a copy of the sample with the given id.
func (b *SampleStore) Get(id string) (domain.WaterSample, error) {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	for i := range b.s.state.Samples {
		if b.s.state.Samples[i].ID == id {
			return cloneWaterSample(b.s.state.Samples[i]), nil
		}
	}
	return domain.WaterSample{}, domain.NotFound("water sample", id)
}

// ListByBuoy returns the most recent `limit` samples of a buoy, newest
// first.
func (b *SampleStore) ListByBuoy(buoyID string, limit int) []domain.WaterSample {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	out := make([]domain.WaterSample, 0)
	for i := range b.s.state.Samples {
		if b.s.state.Samples[i].BuoyID == buoyID {
			out = append(out, cloneWaterSample(b.s.state.Samples[i]))
		}
	}
	return sortSamplesDesc(out, limit)
}

// ListByZone returns the most recent `limit` samples of a zone, newest
// first.
func (b *SampleStore) ListByZone(zoneID string, limit int) []domain.WaterSample {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	out := make([]domain.WaterSample, 0)
	for i := range b.s.state.Samples {
		if b.s.state.Samples[i].ZoneID == zoneID {
			out = append(out, cloneWaterSample(b.s.state.Samples[i]))
		}
	}
	return sortSamplesDesc(out, limit)
}

// ListByBuoySince returns the samples of a buoy in [from, to], oldest
// first; used by the restore checker and cross validation.
func (b *SampleStore) ListByBuoySince(buoyID string, from, to time.Time) []domain.WaterSample {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	out := make([]domain.WaterSample, 0)
	for i := range b.s.state.Samples {
		s := b.s.state.Samples[i]
		if s.BuoyID == buoyID && !s.Timestamp.Before(from) && !s.Timestamp.After(to) {
			out = append(out, cloneWaterSample(s))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}

// ListByZoneSince returns the samples of a zone in [from, to], oldest
// first; used by the restore checker.
func (b *SampleStore) ListByZoneSince(zoneID string, from, to time.Time) []domain.WaterSample {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	out := make([]domain.WaterSample, 0)
	for i := range b.s.state.Samples {
		s := b.s.state.Samples[i]
		if s.ZoneID == zoneID && !s.Timestamp.Before(from) && !s.Timestamp.After(to) {
			out = append(out, cloneWaterSample(s))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}

// LatestByBuoy returns the most recent sample of a buoy, if any.
func (b *SampleStore) LatestByBuoy(buoyID string) (domain.WaterSample, bool) {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	var latest *domain.WaterSample
	for i := range b.s.state.Samples {
		s := &b.s.state.Samples[i]
		if s.BuoyID != buoyID {
			continue
		}
		if latest == nil || s.Timestamp.After(latest.Timestamp) {
			latest = s
		}
	}
	if latest == nil {
		return domain.WaterSample{}, false
	}
	return cloneWaterSample(*latest), true
}

// LatestByZone returns the most recent sample of a zone, if any.
func (b *SampleStore) LatestByZone(zoneID string) (domain.WaterSample, bool) {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	var latest *domain.WaterSample
	for i := range b.s.state.Samples {
		s := &b.s.state.Samples[i]
		if s.ZoneID != zoneID {
			continue
		}
		if latest == nil || s.Timestamp.After(latest.Timestamp) {
			latest = s
		}
	}
	if latest == nil {
		return domain.WaterSample{}, false
	}
	return cloneWaterSample(*latest), true
}

// Count returns the total number of samples.
func (b *SampleStore) Count() int {
	b.s.mu.RLock()
	defer b.s.mu.RUnlock()
	return len(b.s.state.Samples)
}

func sortSamplesDesc(in []domain.WaterSample, limit int) []domain.WaterSample {
	out := make([]domain.WaterSample, 0, len(in))
	out = append(out, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
