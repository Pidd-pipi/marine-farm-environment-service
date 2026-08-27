// Package config centralises every tunable knob of the marine farm
// environment service: dissolved-oxygen thresholds, cross-validation
// window, restore conditions, aerator feedback timeout, water-quality
// ranges, HTTP listen address, logging level and environment overrides.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultPort is used when the PORT environment variable is not set.
const DefaultPort = "8080"

// Config aggregates every tunable constant of the service. Fields are
// exported so tests and callers can construct custom configurations.
type Config struct {
	// Port is the TCP port the HTTP server listens on (PORT env overrides).
	Port string

	// DataFile is the JSON persistence file. Empty disables persistence.
	DataFile string

	// LogLevel is the structured-logging level: debug, info, warn or error.
	LogLevel string

	// DOWarnThreshold is the dissolved-oxygen level (mg/L) below which a
	// zone enters the warning state (spec: < 4 mg/L).
	DOWarnThreshold float64

	// DODangerThreshold is the dissolved-oxygen level (mg/L) below which a
	// zone enters the danger state (spec: < 3 mg/L).
	DODangerThreshold float64

	// DORestoreThreshold is the dissolved-oxygen level (mg/L) above which,
	// sustained for RestoreSustained, a zone may be confirmed as restored
	// (spec: >= 5 mg/L).
	DORestoreThreshold float64

	// RestoreSustained is how long dissolved oxygen must stay above
	// DORestoreThreshold before the operator may confirm restore
	// (spec: 30 minutes).
	RestoreSustained time.Duration

	// CrossCheckWindow is the time window used for neighbouring-buoy
	// cross validation of a dangerous dissolved-oxygen reading
	// (spec: 15 minutes).
	CrossCheckWindow time.Duration

	// SamplePeriod is the expected buoy reporting period (spec: 5 minutes).
	SamplePeriod time.Duration

	// SamplePeriodTolerance is how much earlier than SamplePeriod a new
	// report is still accepted (prevents clock-drift rejects).
	SamplePeriodTolerance time.Duration

	// AeratorFeedbackTimeout is how long a start/stop command may stay
	// without feedback before it is treated as a fault.
	AeratorFeedbackTimeout time.Duration

	// RestoreCheckInterval is how often the background restore checker
	// runs (spec: every 5 minutes).
	RestoreCheckInterval time.Duration

	// TempMin/TempMax is the acceptable water-temperature range (°C).
	TempMin float64
	TempMax float64

	// SalinityMin/SalinityMax is the acceptable salinity range (‰).
	SalinityMin float64
	SalinityMax float64

	// PHMin/PHMax is the acceptable pH range.
	PHMin float64
	PHMax float64

	// AmmoniaMax is the maximum acceptable ammonia concentration (mg/L).
	AmmoniaMax float64

	// DeathAbnormalRatio is the share of stock above which a single day's
	// death count is flagged as abnormal (spec: 1%).
	DeathAbnormalRatio float64

	// MaxSamplesPerBuoy caps the retained sample history per buoy.
	MaxSamplesPerBuoy int
}

// Default returns a Config populated with the domain defaults from the
// prompt: 4/3/5 mg/L DO thresholds, 15-minute cross-check window, 30-minute
// sustained restore condition, 5-minute reporting period and 1% death ratio.
func Default() *Config {
	return &Config{
		Port:                   DefaultPort,
		DataFile:               "data/marine_data.json",
		LogLevel:               "info",
		DOWarnThreshold:        4.0,
		DODangerThreshold:      3.0,
		DORestoreThreshold:     5.0,
		RestoreSustained:       30 * time.Minute,
		CrossCheckWindow:       15 * time.Minute,
		SamplePeriod:           5 * time.Minute,
		SamplePeriodTolerance:  60 * time.Second,
		AeratorFeedbackTimeout: 2 * time.Minute,
		RestoreCheckInterval:   5 * time.Minute,
		TempMin:                10,
		TempMax:                32,
		SalinityMin:            25,
		SalinityMax:            35,
		PHMin:                  7.0,
		PHMax:                  8.8,
		AmmoniaMax:             0.2,
		DeathAbnormalRatio:     0.01,
		MaxSamplesPerBuoy:      2000,
	}
}

// FromEnv builds a Config from environment variables, falling back to
// Default for anything unset. Supported variables are documented in the
// README environment-variable table.
func FromEnv() *Config {
	cfg := Default()
	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}
	if v, ok := os.LookupEnv("DATA_FILE"); ok {
		cfg.DataFile = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(strings.TrimSpace(v))
	}
	if f, ok := envFloat("DO_WARN_THRESHOLD"); ok {
		cfg.DOWarnThreshold = f
	}
	if f, ok := envFloat("DO_DANGER_THRESHOLD"); ok {
		cfg.DODangerThreshold = f
	}
	if f, ok := envFloat("DO_RESTORE_THRESHOLD"); ok {
		cfg.DORestoreThreshold = f
	}
	if d, ok := envDuration("RESTORE_SUSTAINED"); ok {
		cfg.RestoreSustained = d
	}
	if d, ok := envDuration("CROSS_CHECK_WINDOW"); ok {
		cfg.CrossCheckWindow = d
	}
	if d, ok := envDuration("SAMPLE_PERIOD"); ok {
		cfg.SamplePeriod = d
	}
	if d, ok := envDuration("SAMPLE_PERIOD_TOLERANCE"); ok {
		cfg.SamplePeriodTolerance = d
	}
	if d, ok := envDuration("AERATOR_TIMEOUT"); ok {
		cfg.AeratorFeedbackTimeout = d
	}
	if d, ok := envDuration("RESTORE_CHECK_INTERVAL"); ok {
		cfg.RestoreCheckInterval = d
	}
	if f, ok := envFloat("TEMP_MIN"); ok {
		cfg.TempMin = f
	}
	if f, ok := envFloat("TEMP_MAX"); ok {
		cfg.TempMax = f
	}
	if f, ok := envFloat("SALINITY_MIN"); ok {
		cfg.SalinityMin = f
	}
	if f, ok := envFloat("SALINITY_MAX"); ok {
		cfg.SalinityMax = f
	}
	if f, ok := envFloat("PH_MIN"); ok {
		cfg.PHMin = f
	}
	if f, ok := envFloat("PH_MAX"); ok {
		cfg.PHMax = f
	}
	if f, ok := envFloat("AMMONIA_MAX"); ok {
		cfg.AmmoniaMax = f
	}
	if f, ok := envFloat("DEATH_ABNORMAL_RATIO"); ok {
		cfg.DeathAbnormalRatio = f
	}
	if n, ok := envInt("MAX_SAMPLES_PER_BUOY"); ok {
		cfg.MaxSamplesPerBuoy = n
	}
	return cfg
}

// envFloat reads a float64 environment variable.
func envFloat(key string) (float64, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// envInt reads an int environment variable.
func envInt(key string) (int, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

// envDuration reads a time.Duration environment variable.
func envDuration(key string) (time.Duration, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, false
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, false
	}
	return d, true
}

// Validate checks that the configuration is internally consistent and
// returns a descriptive error otherwise.
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("config: port must not be empty")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("config: LOG_LEVEL must be one of debug/info/warn/error, got %q", c.LogLevel)
	}
	if c.DOWarnThreshold <= c.DODangerThreshold {
		return fmt.Errorf("config: DOWarnThreshold (%v) must exceed DODangerThreshold (%v)", c.DOWarnThreshold, c.DODangerThreshold)
	}
	if c.DORestoreThreshold <= c.DOWarnThreshold {
		return fmt.Errorf("config: DORestoreThreshold (%v) must exceed DOWarnThreshold (%v)", c.DORestoreThreshold, c.DOWarnThreshold)
	}
	if c.RestoreSustained <= 0 {
		return fmt.Errorf("config: RestoreSustained must be positive")
	}
	if c.CrossCheckWindow <= 0 {
		return fmt.Errorf("config: CrossCheckWindow must be positive")
	}
	if c.SamplePeriod <= 0 {
		return fmt.Errorf("config: SamplePeriod must be positive")
	}
	if c.SamplePeriodTolerance < 0 {
		return fmt.Errorf("config: SamplePeriodTolerance must not be negative")
	}
	if c.AeratorFeedbackTimeout <= 0 {
		return fmt.Errorf("config: AeratorFeedbackTimeout must be positive")
	}
	if c.RestoreCheckInterval <= 0 {
		return fmt.Errorf("config: RestoreCheckInterval must be positive")
	}
	if c.TempMin >= c.TempMax {
		return fmt.Errorf("config: TempMin must be below TempMax")
	}
	if c.SalinityMin >= c.SalinityMax {
		return fmt.Errorf("config: SalinityMin must be below SalinityMax")
	}
	if c.PHMin >= c.PHMax {
		return fmt.Errorf("config: PHMin must be below PHMax")
	}
	if c.AmmoniaMax <= 0 {
		return fmt.Errorf("config: AmmoniaMax must be positive")
	}
	if c.DeathAbnormalRatio <= 0 || c.DeathAbnormalRatio >= 1 {
		return fmt.Errorf("config: DeathAbnormalRatio must be in (0,1)")
	}
	if c.MaxSamplesPerBuoy < 10 {
		return fmt.Errorf("config: MaxSamplesPerBuoy must be >= 10")
	}
	return nil
}

// WaterRange returns the acceptable water-quality envelope as a struct,
// used by domain limit evaluation.
func (c *Config) WaterRange() WaterRange {
	return WaterRange{
		TempMin:     c.TempMin,
		TempMax:     c.TempMax,
		SalinityMin: c.SalinityMin,
		SalinityMax: c.SalinityMax,
		PHMin:       c.PHMin,
		PHMax:       c.PHMax,
		AmmoniaMax:  c.AmmoniaMax,
	}
}

// WaterRange is the acceptable water-quality envelope.
type WaterRange struct {
	TempMin     float64
	TempMax     float64
	SalinityMin float64
	SalinityMax float64
	PHMin       float64
	PHMax       float64
	AmmoniaMax  float64
}
