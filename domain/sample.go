package domain

import (
	"fmt"
	"time"
)

// WaterSample is one water-quality reading reported by a buoy (水质数据).
// Besides the raw measurements it carries the over-limit verdict and the
// list of violated indicators.
type WaterSample struct {
	ID          string    `json:"id"`
	BuoyID      string    `json:"buoy_id"`
	ZoneID      string    `json:"zone_id"`
	DO          float64   `json:"do"`
	Temperature float64   `json:"temperature"`
	Salinity    float64   `json:"salinity"`
	PH          float64   `json:"ph"`
	Ammonia     float64   `json:"ammonia"`
	Timestamp   time.Time `json:"timestamp"`
	OverLimit   bool      `json:"over_limit"`
	Violations  []string  `json:"violations"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewWaterSample builds a sample with no violation verdict yet.
func NewWaterSample(id, buoyID, zoneID string, do, temperature, salinity, ph, ammonia float64, ts time.Time) *WaterSample {
	return &WaterSample{
		ID:          id,
		BuoyID:      buoyID,
		ZoneID:      zoneID,
		DO:          do,
		Temperature: temperature,
		Salinity:    salinity,
		PH:          ph,
		Ammonia:     ammonia,
		Timestamp:   ts,
		CreatedAt:   time.Now().UTC(),
	}
}

// SampleThresholds is the water-quality envelope used to judge a sample.
type SampleThresholds struct {
	DOWarnThreshold   float64
	DODangerThreshold float64
	TempMin           float64
	TempMax           float64
	SalinityMin       float64
	SalinityMax       float64
	PHMin             float64
	PHMax             float64
	AmmoniaMax        float64
}

// Violation describes one over-limit indicator.
type Violation struct {
	Indicator string
	Type      WarningType
	Level     WarningLevel
	Value     float64
	Limit     float64
}

// EvaluateLimits judges a sample against the zone's thresholds and returns
// every violated indicator plus the over-limit verdict. Dissolved oxygen is
// judged at both the warning (< DOWarnThreshold) and the danger level
// (< DODangerThreshold); the returned DO violation reflects the most severe
// one.
func (s *WaterSample) EvaluateLimits(th SampleThresholds) ([]Violation, bool) {
	var violations []Violation
	if s.DO < th.DOWarnThreshold {
		level := WarningLevelWarning
		limit := th.DOWarnThreshold
		if s.DO < th.DODangerThreshold {
			level = WarningLevelDanger
			limit = th.DODangerThreshold
		}
		violations = append(violations, Violation{
			Indicator: "do",
			Type:      WarningTypeDOLow,
			Level:     level,
			Value:     s.DO,
			Limit:     limit,
		})
	}
	if s.Temperature < th.TempMin {
		violations = append(violations, Violation{
			Indicator: "temperature", Type: WarningTypeTempShock, Level: WarningLevelWarning,
			Value: s.Temperature, Limit: th.TempMin,
		})
	}
	if s.Temperature > th.TempMax {
		violations = append(violations, Violation{
			Indicator: "temperature", Type: WarningTypeTempShock, Level: WarningLevelWarning,
			Value: s.Temperature, Limit: th.TempMax,
		})
	}
	if s.PH < th.PHMin {
		violations = append(violations, Violation{
			Indicator: "ph", Type: WarningTypePHAbnormal, Level: WarningLevelWarning,
			Value: s.PH, Limit: th.PHMin,
		})
	}
	if s.PH > th.PHMax {
		violations = append(violations, Violation{
			Indicator: "ph", Type: WarningTypePHAbnormal, Level: WarningLevelWarning,
			Value: s.PH, Limit: th.PHMax,
		})
	}
	if s.Ammonia > th.AmmoniaMax {
		violations = append(violations, Violation{
			Indicator: "ammonia", Type: WarningTypeAmmoniaHigh, Level: WarningLevelWarning,
			Value: s.Ammonia, Limit: th.AmmoniaMax,
		})
	}
	over := len(violations) > 0
	if over {
		s.OverLimit = true
		s.Violations = make([]string, 0, len(violations))
		for _, v := range violations {
			s.Violations = append(s.Violations, string(v.Type))
		}
	}
	return violations, over
}

// ViolationSummary renders a short human description of a violation list.
func ViolationSummary(vs []Violation) string {
	if len(vs) == 0 {
		return "正常"
	}
	out := ""
	for i, v := range vs {
		if i > 0 {
			out += "; "
		}
		out += fmt.Sprintf("%s %v (限值 %v)", v.Type.Label(), round2(v.Value), round2(v.Limit))
	}
	return out
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// DOStatus returns the dissolved-oxygen verdict for the sample:
// danger, warning or normal.
func (s *WaterSample) DOStatus(th SampleThresholds) WarningLevel {
	switch {
	case s.DO < th.DODangerThreshold:
		return WarningLevelDanger
	case s.DO < th.DOWarnThreshold:
		return WarningLevelWarning
	}
	return ""
}

// SampleRange is a time-bounded query used for cross validation and trend
// rendering.
type SampleRange struct {
	ZoneID string
	BuoyID string
	From   time.Time
	To     time.Time
}
