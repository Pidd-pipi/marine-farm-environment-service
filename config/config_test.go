package config

import (
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DOWarnThreshold != 4.0 || cfg.DODangerThreshold != 3.0 || cfg.DORestoreThreshold != 5.0 {
		t.Fatalf("unexpected DO thresholds: %v %v %v", cfg.DOWarnThreshold, cfg.DODangerThreshold, cfg.DORestoreThreshold)
	}
	if cfg.RestoreSustained != 30*time.Minute {
		t.Fatalf("unexpected restore sustained: %v", cfg.RestoreSustained)
	}
	if cfg.CrossCheckWindow != 15*time.Minute {
		t.Fatalf("unexpected cross-check window: %v", cfg.CrossCheckWindow)
	}
	if cfg.DeathAbnormalRatio != 0.01 {
		t.Fatalf("unexpected death ratio: %v", cfg.DeathAbnormalRatio)
	}
}

func TestValidate(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config must validate: %v", err)
	}

	bad := *cfg
	bad.DOWarnThreshold = 2.5
	bad.DODangerThreshold = 3.0
	if err := bad.Validate(); err == nil {
		t.Fatal("expected error when warn threshold <= danger threshold")
	}

	bad = *cfg
	bad.RestoreSustained = 0
	if err := bad.Validate(); err == nil {
		t.Fatal("expected error when RestoreSustained is zero")
	}

	bad = *cfg
	bad.PHMin = 9
	if err := bad.Validate(); err == nil {
		t.Fatal("expected error when PHMin >= PHMax")
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("PORT", "19099")
	t.Setenv("DATA_FILE", "/tmp/x.json")
	t.Setenv("DO_WARN_THRESHOLD", "4.5")
	t.Setenv("RESTORE_SUSTAINED", "2s")
	t.Setenv("CROSS_CHECK_WINDOW", "3m")

	cfg := FromEnv()
	if cfg.Port != "19099" {
		t.Fatalf("env PORT not applied: %s", cfg.Port)
	}
	if cfg.DataFile != "/tmp/x.json" {
		t.Fatalf("env DATA_FILE not applied: %s", cfg.DataFile)
	}
	if cfg.DOWarnThreshold != 4.5 {
		t.Fatalf("env DO_WARN_THRESHOLD not applied: %v", cfg.DOWarnThreshold)
	}
	if cfg.RestoreSustained != 2*time.Second {
		t.Fatalf("env RESTORE_SUSTAINED not applied: %v", cfg.RestoreSustained)
	}
	if cfg.CrossCheckWindow != 3*time.Minute {
		t.Fatalf("env CROSS_CHECK_WINDOW not applied: %v", cfg.CrossCheckWindow)
	}
}

func TestWaterRange(t *testing.T) {
	cfg := Default()
	r := cfg.WaterRange()
	if r.TempMin != 10 || r.TempMax != 32 {
		t.Fatalf("unexpected temp range: %v..%v", r.TempMin, r.TempMax)
	}
	if r.AmmoniaMax != 0.2 {
		t.Fatalf("unexpected ammonia max: %v", r.AmmoniaMax)
	}
}
