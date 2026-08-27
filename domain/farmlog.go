package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// FarmLog is the daily farming record (养殖日志) of a zone: feeding amount,
// death count and disease notes. A single-day death count above 1% of the
// zone stock is automatically flagged as abnormal.
type FarmLog struct {
	ID            string    `json:"id"`
	ZoneID        string    `json:"zone_id"`
	Date          string    `json:"date"` // YYYY-MM-DD
	FeedAmount    float64   `json:"feed_amount"`
	DeathCount    int       `json:"death_count"`
	DiseaseNote   string    `json:"disease_note"`
	DeathAbnormal bool      `json:"death_abnormal"`
	Operator      string    `json:"operator"`
	CreatedAt     time.Time `json:"created_at"`
}

// NewFarmLog builds a farm log and evaluates the death-abnormal flag.
func NewFarmLog(id, zoneID, date string, feedAmount float64, deathCount int, diseaseNote, operator string, stock int, abnormalRatio float64, now time.Time) *FarmLog {
	return &FarmLog{
		ID:            id,
		ZoneID:        zoneID,
		Date:          date,
		FeedAmount:    feedAmount,
		DeathCount:    deathCount,
		DiseaseNote:   diseaseNote,
		DeathAbnormal: EvaluateDeathAbnormal(deathCount, stock, abnormalRatio),
		Operator:      operator,
		CreatedAt:     now,
	}
}

// EvaluateDeathAbnormal reports whether a single-day death count exceeds
// the abnormal share of the zone stock (spec: > 1%).
func EvaluateDeathAbnormal(deathCount, stock int, ratio float64) bool {
	if stock <= 0 || ratio <= 0 {
		return false
	}
	limit := float64(stock) * ratio
	return float64(deathCount) >= limit
}

// ValidateLogInput validates the business fields of a farm log request.
func ValidateLogInput(date string, feedAmount float64, deathCount int, stock int) error {
	if strings.TrimSpace(date) == "" {
		return InvalidInput("date is required")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return InvalidInput("date must be YYYY-MM-DD: %v", err)
	}
	if math.IsNaN(feedAmount) || math.IsInf(feedAmount, 0) || feedAmount < 0 {
		return InvalidInput("feed_amount must be a finite non-negative number")
	}
	if deathCount < 0 {
		return InvalidInput("death_count must be >= 0")
	}
	if deathCount > stock {
		return InvalidInput("death_count %d exceeds zone stock %d", deathCount, stock)
	}
	return nil
}

// String returns a compact description used in audit entries.
func (f *FarmLog) String() string {
	return fmt.Sprintf("farm log %s (%s, %s)", f.ID, f.ZoneID, f.Date)
}
