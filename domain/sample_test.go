package domain

import (
	"testing"
	"time"
)

func testThresholds() SampleThresholds {
	return SampleThresholds{
		DOWarnThreshold:   4.0,
		DODangerThreshold: 3.0,
		TempMin:           10,
		TempMax:           32,
		SalinityMin:       25,
		SalinityMax:       35,
		PHMin:             7.0,
		PHMax:             8.8,
		AmmoniaMax:        0.2,
	}
}

func TestEvaluateLimitsNormal(t *testing.T) {
	s := NewWaterSample("s1", "b1", "z1", 6.2, 24.5, 31.0, 8.1, 0.06, time.Now())
	vs, over := s.EvaluateLimits(testThresholds())
	if over || len(vs) != 0 {
		t.Fatalf("normal sample must not be over limit: %v %v", vs, over)
	}
}

func TestEvaluateLimitsDangerDO(t *testing.T) {
	s := NewWaterSample("s1", "b1", "z1", 2.4, 24.5, 31.0, 8.1, 0.06, time.Now())
	vs, over := s.EvaluateLimits(testThresholds())
	if !over || len(vs) != 1 {
		t.Fatalf("expected one violation, got %v", vs)
	}
	if vs[0].Type != WarningTypeDOLow || vs[0].Level != WarningLevelDanger {
		t.Fatalf("expected do_low danger violation, got %+v", vs[0])
	}
	if s.DOStatus(testThresholds()) != WarningLevelDanger {
		t.Fatal("DOStatus should be danger")
	}
}

func TestEvaluateLimitsWarningDO(t *testing.T) {
	s := NewWaterSample("s1", "b1", "z1", 3.5, 24.5, 31.0, 8.1, 0.06, time.Now())
	vs, _ := s.EvaluateLimits(testThresholds())
	if len(vs) != 1 || vs[0].Level != WarningLevelWarning {
		t.Fatalf("expected do_low warning, got %+v", vs)
	}
}

func TestEvaluateLimitsMultiple(t *testing.T) {
	s := NewWaterSample("s1", "b1", "z1", 2.8, 35.0, 31.0, 9.2, 0.5, time.Now())
	vs, over := s.EvaluateLimits(testThresholds())
	if !over {
		t.Fatal("expected over limit")
	}
	types := map[WarningType]bool{}
	for _, v := range vs {
		types[v.Type] = true
	}
	if !types[WarningTypeDOLow] || !types[WarningTypeTempShock] || !types[WarningTypePHAbnormal] || !types[WarningTypeAmmoniaHigh] {
		t.Fatalf("expected all four violations, got %+v", vs)
	}
}

func TestViolationSummary(t *testing.T) {
	s := NewWaterSample("s1", "b1", "z1", 2.4, 24.5, 31.0, 8.1, 0.06, time.Now())
	vs, _ := s.EvaluateLimits(testThresholds())
	if ViolationSummary(vs) == "" {
		t.Fatal("summary must not be empty")
	}
}
