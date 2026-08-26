package domain

import (
	"testing"
	"time"
)

func TestEvaluateDeathAbnormal(t *testing.T) {
	if !EvaluateDeathAbnormal(501, 50000, 0.01) {
		t.Fatal("501 deaths on 50000 stock (>1%) must be abnormal")
	}
	if EvaluateDeathAbnormal(500, 50000, 0.01) {
		t.Fatal("500 deaths on 50000 stock (exactly 1%) must not be abnormal")
	}
	if EvaluateDeathAbnormal(1, 100, 0.01) {
		t.Fatal("1 death on 100 stock (1%) must not be abnormal")
	}
	if EvaluateDeathAbnormal(10, 0, 0.01) {
		t.Fatal("zero stock must never be abnormal")
	}
}

func TestNewFarmLog(t *testing.T) {
	now := time.Now().UTC()
	log := NewFarmLog("fl_1", "z1", "2026-08-25", 500, 800, "病害", "张师傅", 50000, 0.01, now)
	if !log.DeathAbnormal {
		t.Fatal("800 deaths on 50000 stock must be flagged abnormal")
	}
	log2 := NewFarmLog("fl_2", "z1", "2026-08-25", 500, 10, "", "张师傅", 50000, 0.01, now)
	if log2.DeathAbnormal {
		t.Fatal("10 deaths must not be flagged abnormal")
	}
}

func TestValidateLogInput(t *testing.T) {
	if err := ValidateLogInput("2026-08-25", 100, 5, 50000); err != nil {
		t.Fatalf("valid input must pass: %v", err)
	}
	if err := ValidateLogInput("", 100, 5, 50000); err == nil {
		t.Fatal("empty date must fail")
	}
	if err := ValidateLogInput("2026/08/25", 100, 5, 50000); err == nil {
		t.Fatal("wrong date format must fail")
	}
	if err := ValidateLogInput("2026-08-25", -1, 5, 50000); err == nil {
		t.Fatal("negative feed must fail")
	}
	if err := ValidateLogInput("2026-08-25", 100, -1, 50000); err == nil {
		t.Fatal("negative death must fail")
	}
	if err := ValidateLogInput("2026-08-25", 100, 60000, 50000); err == nil {
		t.Fatal("death above stock must fail")
	}
}
