// Package service implements the application use-cases. Services orchestrate
// the domain rules on top of the store repositories; HTTP handlers stay thin
// and delegate every business decision here.
package service

import (
	"context"
	"log/slog"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/store"
)

// Services aggregates every application service. Handlers receive this
// bundle so they never touch the store directly.
type Services struct {
	Cfg      *config.Config
	Store    *store.Store
	Zones    *ZoneService
	Buoys    *BuoyService
	Ingest   *IngestService
	Warnings *WarningService
	Aeration *AerationService
	Restore  *RestoreChecker
	FarmLog  *FarmLogService
	Audit    *AuditService
	Overview *OverviewService
}

// New wires the service graph in dependency order.
func New(cfg *config.Config, st *store.Store) *Services {
	audit := NewAuditService(st)
	zones := NewZoneService(st, audit)
	buoys := NewBuoyService(st, audit)
	aeration := NewAerationService(cfg, st, audit)
	ingest := NewIngestService(cfg, st, aeration, audit)
	warnings := NewWarningService(st, aeration, audit)
	restore := NewRestoreChecker(cfg, st, aeration, audit)
	farmlog := NewFarmLogService(cfg, st, audit)
	overview := NewOverviewService(cfg, st)
	return &Services{
		Cfg:      cfg,
		Store:    st,
		Zones:    zones,
		Buoys:    buoys,
		Ingest:   ingest,
		Warnings: warnings,
		Aeration: aeration,
		Restore:  restore,
		FarmLog:  farmlog,
		Audit:    audit,
		Overview: overview,
	}
}

// StartSweepers launches the background restore checker. It stops when the
// context is cancelled.
func (s *Services) StartSweepers(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		s.Restore.Run(context.Background())
		close(done)
	}()
	slog.Info("restore checker started", "interval", s.Cfg.RestoreCheckInterval)
	return done
}
