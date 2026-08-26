// Package domain contains the core entities, value objects and business
// rules of the marine farm environment service. It has no dependencies on
// stores, services or HTTP so the rules stay testable in isolation.
package domain

// ZoneStatus is the lifecycle state of a farm zone (养殖区状态机).
//
//	normal   → warning → danger → aerating → restored
//
// The canonical chain from the prompt is: 正常(normal) → 预警(warning) →
// 危险(danger) → 应急增氧(aerating) → 恢复确认(restored). See zone.go for
// the full legal-transition table.
type ZoneStatus string

const (
	ZoneStatusNormal   ZoneStatus = "normal"
	ZoneStatusWarning  ZoneStatus = "warning"
	ZoneStatusDanger   ZoneStatus = "danger"
	ZoneStatusAerating ZoneStatus = "aerating"
	ZoneStatusRestored ZoneStatus = "restored"
)

// Valid reports whether the status is one of the known zone statuses.
func (s ZoneStatus) Valid() bool {
	switch s {
	case ZoneStatusNormal, ZoneStatusWarning, ZoneStatusDanger, ZoneStatusAerating, ZoneStatusRestored:
		return true
	}
	return false
}

// Label returns the Chinese label used by the frontend.
func (s ZoneStatus) Label() string {
	switch s {
	case ZoneStatusNormal:
		return "正常运行"
	case ZoneStatusWarning:
		return "预警"
	case ZoneStatusDanger:
		return "危险"
	case ZoneStatusAerating:
		return "应急增氧"
	case ZoneStatusRestored:
		return "恢复确认"
	}
	return string(s)
}

// AllZoneStatuses lists every legal zone status.
func AllZoneStatuses() []ZoneStatus {
	return []ZoneStatus{ZoneStatusNormal, ZoneStatusWarning, ZoneStatusDanger, ZoneStatusAerating, ZoneStatusRestored}
}

// WarningType is the category of an abnormal water-quality reading.
type WarningType string

const (
	WarningTypeDOLow       WarningType = "do_low"
	WarningTypeTempShock   WarningType = "temp_shock"
	WarningTypePHAbnormal  WarningType = "ph_abnormal"
	WarningTypeAmmoniaHigh WarningType = "ammonia_high"
)

// Valid reports whether the warning type is known.
func (t WarningType) Valid() bool {
	switch t {
	case WarningTypeDOLow, WarningTypeTempShock, WarningTypePHAbnormal, WarningTypeAmmoniaHigh:
		return true
	}
	return false
}

// Label returns the Chinese label of a warning type.
func (t WarningType) Label() string {
	switch t {
	case WarningTypeDOLow:
		return "溶解氧过低"
	case WarningTypeTempShock:
		return "水温骤变"
	case WarningTypePHAbnormal:
		return "pH 异常"
	case WarningTypeAmmoniaHigh:
		return "氨氮超标"
	}
	return string(t)
}

// AllWarningTypes lists every legal warning type.
func AllWarningTypes() []WarningType {
	return []WarningType{WarningTypeDOLow, WarningTypeTempShock, WarningTypePHAbnormal, WarningTypeAmmoniaHigh}
}

// WarningStatus is the lifecycle state of a warning record.
//
//	pending   - dangerous reading is waiting for cross-validation result
//	           or operator review (待核实)
//	confirmed - the anomaly is verified (已确认)
//	resolved  - the anomaly has been cleared (已解除)
type WarningStatus string

const (
	WarningStatusPending   WarningStatus = "pending"
	WarningStatusConfirmed WarningStatus = "confirmed"
	WarningStatusResolved  WarningStatus = "resolved"
)

// Valid reports whether the warning status is known.
func (s WarningStatus) Valid() bool {
	switch s {
	case WarningStatusPending, WarningStatusConfirmed, WarningStatusResolved:
		return true
	}
	return false
}

// Label returns the Chinese label of a warning status.
func (s WarningStatus) Label() string {
	switch s {
	case WarningStatusPending:
		return "待核实"
	case WarningStatusConfirmed:
		return "已确认"
	case WarningStatusResolved:
		return "已解除"
	}
	return string(s)
}

// WarningLevel is the severity of a warning.
type WarningLevel string

const (
	WarningLevelWarning WarningLevel = "warning"
	WarningLevelDanger  WarningLevel = "danger"
)

// Valid reports whether the warning level is known.
func (l WarningLevel) Valid() bool {
	return l == WarningLevelWarning || l == WarningLevelDanger
}

// Label returns the Chinese label of a warning level.
func (l WarningLevel) Label() string {
	switch l {
	case WarningLevelWarning:
		return "预警"
	case WarningLevelDanger:
		return "危险"
	}
	return string(l)
}

// AeratorStatus is the lifecycle state of an aerator (增氧机状态机):
//
//	stopped → starting → running → stopping → stopped
//
// Any command that does not receive feedback within the configured timeout
// is moved to fault and raises an alert.
type AeratorStatus string

const (
	AeratorStatusStopped  AeratorStatus = "stopped"
	AeratorStatusStarting AeratorStatus = "starting"
	AeratorStatusRunning  AeratorStatus = "running"
	AeratorStatusStopping AeratorStatus = "stopping"
	AeratorStatusFault    AeratorStatus = "fault"
)

// Valid reports whether the aerator status is known.
func (s AeratorStatus) Valid() bool {
	switch s {
	case AeratorStatusStopped, AeratorStatusStarting, AeratorStatusRunning, AeratorStatusStopping, AeratorStatusFault:
		return true
	}
	return false
}

// Label returns the Chinese label of an aerator status.
func (s AeratorStatus) Label() string {
	switch s {
	case AeratorStatusStopped:
		return "已停止"
	case AeratorStatusStarting:
		return "启动中"
	case AeratorStatusRunning:
		return "运行中"
	case AeratorStatusStopping:
		return "停止中"
	case AeratorStatusFault:
		return "故障"
	}
	return string(s)
}

// AllAeratorStatuses lists every legal aerator status.
func AllAeratorStatuses() []AeratorStatus {
	return []AeratorStatus{AeratorStatusStopped, AeratorStatusStarting, AeratorStatusRunning, AeratorStatusStopping, AeratorStatusFault}
}

// FeedbackStatus is the feedback received from an aerator for a command.
type FeedbackStatus string

const (
	FeedbackNone         FeedbackStatus = "none"
	FeedbackAcknowledged FeedbackStatus = "acknowledged"
	FeedbackStarted      FeedbackStatus = "started"
	FeedbackStopped      FeedbackStatus = "stopped"
	FeedbackFault        FeedbackStatus = "fault"
	FeedbackTimeout      FeedbackStatus = "timeout"
)

// Valid reports whether the feedback status is known.
func (f FeedbackStatus) Valid() bool {
	switch f {
	case FeedbackNone, FeedbackAcknowledged, FeedbackStarted, FeedbackStopped, FeedbackFault, FeedbackTimeout:
		return true
	}
	return false
}

// BuoyStatus is the operational state of a monitoring buoy.
type BuoyStatus string

const (
	BuoyStatusActive      BuoyStatus = "active"
	BuoyStatusOffline     BuoyStatus = "offline"
	BuoyStatusMaintenance BuoyStatus = "maintenance"
)

// Valid reports whether the buoy status is known.
func (s BuoyStatus) Valid() bool {
	switch s {
	case BuoyStatusActive, BuoyStatusOffline, BuoyStatusMaintenance:
		return true
	}
	return false
}

// Label returns the Chinese label of a buoy status.
func (s BuoyStatus) Label() string {
	switch s {
	case BuoyStatusActive:
		return "在线"
	case BuoyStatusOffline:
		return "离线"
	case BuoyStatusMaintenance:
		return "维护中"
	}
	return string(s)
}

// AerationAction is the action recorded on an aeration log.
type AerationAction string

const (
	AerationActionStart AerationAction = "start"
	AerationActionStop  AerationAction = "stop"
)

// Valid reports whether the aeration action is known.
func (a AerationAction) Valid() bool {
	return a == AerationActionStart || a == AerationActionStop
}

// Label returns the Chinese label of an aeration action.
func (a AerationAction) Label() string {
	switch a {
	case AerationActionStart:
		return "启动增氧"
	case AerationActionStop:
		return "停止增氧"
	}
	return string(a)
}

// AerationTrigger describes who/what issued the aeration command.
type AerationTrigger string

const (
	TriggerAuto    AerationTrigger = "auto"
	TriggerManual  AerationTrigger = "manual"
	TriggerVerify  AerationTrigger = "verify"
	TriggerRestore AerationTrigger = "restore"
)

// Valid reports whether the trigger is known.
func (t AerationTrigger) Valid() bool {
	switch t {
	case TriggerAuto, TriggerManual, TriggerVerify, TriggerRestore:
		return true
	}
	return false
}
