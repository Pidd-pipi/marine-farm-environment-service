package domain

import (
	"fmt"
	"time"
)

// WarningRecord is an abnormal water-quality event (预警记录). Dangerous
// dissolved-oxygen readings are cross-validated against neighbouring buoys
// before they may trigger aeration.
type WarningRecord struct {
	ID          string        `json:"id"`
	ZoneID      string        `json:"zone_id"`
	BuoyID      string        `json:"buoy_id"`
	Type        WarningType   `json:"type"`
	Level       WarningLevel  `json:"level"`
	Status      WarningStatus `json:"status"`
	DO          float64       `json:"do"`
	Temperature float64       `json:"temperature"`
	Salinity    float64       `json:"salinity"`
	PH          float64       `json:"ph"`
	Ammonia     float64       `json:"ammonia"`
	Detail      string        `json:"detail"`

	// CrossChecked records whether a neighbouring-buoy cross validation
	// was performed for this warning.
	CrossChecked bool `json:"cross_checked"`

	// CrossCheckOK records whether neighbouring buoys reported normal data
	// inside the window (true → the reading is suspicious and stays
	// pending; false → no contradicting data, the warning is confirmed).
	CrossCheckOK bool `json:"cross_check_ok"`

	ReportedAt time.Time  `json:"reported_at"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// NewWarningRecord builds a warning record from a reported sample. The
// default status is confirmed; call Pending for cross-checked dangers.
func NewWarningRecord(id, zoneID, buoyID string, wtype WarningType, level WarningLevel, s *WaterSample, detail string, now time.Time) *WarningRecord {
	return &WarningRecord{
		ID:          id,
		ZoneID:      zoneID,
		BuoyID:      buoyID,
		Type:        wtype,
		Level:       level,
		Status:      WarningStatusConfirmed,
		DO:          s.DO,
		Temperature: s.Temperature,
		Salinity:    s.Salinity,
		PH:          s.PH,
		Ammonia:     s.Ammonia,
		Detail:      detail,
		ReportedAt:  s.Timestamp,
		CreatedAt:   now,
	}
}

// Pending marks the warning as awaiting verification.
func (w *WarningRecord) Pending() {
	w.Status = WarningStatusPending
}

// Verify confirms a pending warning.
func (w *WarningRecord) Verify(now time.Time) error {
	if w.Status != WarningStatusPending {
		return Conflict("warning %s is %s, only pending warnings can be verified", w.ID, w.Status)
	}
	w.Status = WarningStatusConfirmed
	ts := now
	w.VerifiedAt = &ts
	return nil
}

// Resolve marks the warning as resolved.
func (w *WarningRecord) Resolve(now time.Time) error {
	if w.Status == WarningStatusResolved {
		return nil
	}
	w.Status = WarningStatusResolved
	ts := now
	w.ResolvedAt = &ts
	return nil
}

// IsOpen reports whether the warning still needs attention.
func (w *WarningRecord) IsOpen() bool {
	return w.Status == WarningStatusPending || w.Status == WarningStatusConfirmed
}

// CrossCheckResult is the outcome of a neighbouring-buoy cross validation.
type CrossCheckResult struct {
	// Contradicted is true when another buoy in the same zone reported
	// normal data inside the cross-check window, making the reading
	// suspicious and keeping the warning pending.
	Contradicted bool `json:"contradicted"`

	// Checked records whether a neighbouring buoy with fresh normal data
	// exists.
	Checked bool `json:"checked"`

	// EvidenceSampleID is the neighbouring normal sample used as evidence.
	EvidenceSampleID string `json:"evidence_sample_id,omitempty"`

	// Reason is a human-readable explanation.
	Reason string `json:"reason"`
}

// NeighbourSamples is the input contract for cross validation: all samples
// of the other buoys in the zone inside the window.
type NeighbourSamples struct {
	ZoneID  string
	BuoyID  string // reporting buoy (excluded from evidence)
	From    time.Time
	To      time.Time
	Samples []WaterSample
}

// EvaluateCrossValidation performs the neighbouring-buoy cross validation
// against the provided neighbour samples. Samples from the reporting buoy
// itself are ignored. normalDO is the dissolved-oxygen level considered
// "normal" for a neighbour (the zone warning threshold by default).
//
// Rule: if any other buoy of the same zone reported a normal dissolved
// oxygen within the window, the dangerous reading is marked 待核实
// (pending) and must not directly trigger aeration.
func EvaluateCrossValidation(ns NeighbourSamples, normalDO float64) CrossCheckResult {
	res := CrossCheckResult{}
	for i := 0; i < len(ns.Samples); i++ {
		for j := i + 1; j < len(ns.Samples); j++ {
			if ns.Samples[j].Timestamp.Before(ns.Samples[i].Timestamp) {
				ns.Samples[i], ns.Samples[j] = ns.Samples[j], ns.Samples[i]
			}
		}
	}
	var evidence *WaterSample
	for i := range ns.Samples {
		s := &ns.Samples[i]
		if s.BuoyID == ns.BuoyID {
			continue
		}
		if s.Timestamp.Before(ns.From) || s.Timestamp.After(ns.To) {
			continue
		}
		if s.DO >= normalDO {
			if evidence == nil || s.Timestamp.After(evidence.Timestamp) {
				evidence = s
			}
		}
	}
	if evidence == nil {
		res.Checked = false
		res.Contradicted = false
		res.Reason = "无相邻浮标正常数据，危险读数成立"
		return res
	}
	res.Checked = true
	res.Contradicted = true
	res.EvidenceSampleID = evidence.ID
	res.Reason = fmt.Sprintf(
		"相邻浮标 %s 于 %s 上报溶解氧 %v mg/L（正常），本读数待人工核实",
		evidence.BuoyID, evidence.Timestamp.Format("15:04"), round2(evidence.DO),
	)
	return res
}

// String returns a compact description used in audit entries.
func (w *WarningRecord) String() string {
	return fmt.Sprintf("warning %s (%s/%s)", w.ID, w.Type, w.Status)
}
