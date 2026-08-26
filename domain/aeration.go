package domain

import (
	"fmt"
	"time"
)

// AerationLog records one aerator command and its feedback lifecycle
// (增氧联动记录). Aeration is the emergency response to a confirmed
// dissolved-oxygen danger: the system issues a start command, waits for
// device feedback, and treats missing feedback as a fault.
type AerationLog struct {
	ID          string          `json:"id"`
	ZoneID      string          `json:"zone_id"`
	AeratorID   string          `json:"aerator_id"`
	Action      AerationAction  `json:"action"`
	Trigger     AerationTrigger `json:"trigger"`
	Status      AeratorStatus   `json:"status"`
	Feedback    FeedbackStatus  `json:"feedback"`
	CommandTime time.Time       `json:"command_time"`
	FeedbackAt  *time.Time      `json:"feedback_at,omitempty"`
	Note        string          `json:"note"`
	CreatedAt   time.Time       `json:"created_at"`
}

// NewAerationLog builds a new aerator command. A start command enters the
// starting state; a stop command enters the stopping state.
func NewAerationLog(id, zoneID, aeratorID string, action AerationAction, trigger AerationTrigger, note string, now time.Time) (*AerationLog, error) {
	if !action.Valid() {
		return nil, InvalidInput("invalid aeration action %q", action)
	}
	if !trigger.Valid() {
		return nil, InvalidInput("invalid aeration trigger %q", trigger)
	}
	status := AeratorStatusStarting
	if action == AerationActionStop {
		status = AeratorStatusStopping
	}
	return &AerationLog{
		ID:          id,
		ZoneID:      zoneID,
		AeratorID:   aeratorID,
		Action:      action,
		Trigger:     trigger,
		Status:      status,
		Feedback:    FeedbackNone,
		CommandTime: now,
		CreatedAt:   now,
		Note:        note,
	}, nil
}

// ApplyFeedback processes device feedback and advances the aerator state
// machine. Returns the new status after applying the feedback.
//
//	start command:
//	  acknowledged → starting
//	  started      → running (after an ack)
//	  fault        → fault
//	stop command:
//	  acknowledged → stopping
//	  stopped      → stopped (after an ack)
//	  fault        → fault
//
// An acknowledged command may receive its terminal feedback (started /
// stopped / fault); a terminal feedback may never be replaced.
func (a *AerationLog) ApplyFeedback(fb FeedbackStatus, now time.Time) (AeratorStatus, error) {
	if !fb.Valid() {
		return a.Status, InvalidInput("invalid feedback status %q", fb)
	}
	if fb == FeedbackNone {
		return a.Status, InvalidInput("feedback must not be none")
	}
	if a.Feedback != FeedbackNone {
		if a.Feedback != FeedbackAcknowledged {
			return a.Status, Conflict("aeration %s already received terminal feedback %s", a.ID, a.Feedback)
		}
		if fb == FeedbackAcknowledged {
			return a.Status, Conflict("aeration %s is already acknowledged", a.ID)
		}
	}
	a.Feedback = fb
	ts := now
	a.FeedbackAt = &ts

	switch a.Action {
	case AerationActionStart:
		switch fb {
		case FeedbackAcknowledged:
			a.Status = AeratorStatusStarting
		case FeedbackStarted:
			a.Status = AeratorStatusRunning
		case FeedbackFault, FeedbackTimeout:
			a.Status = AeratorStatusFault
		default:
			return a.Status, InvalidInput("feedback %s is not valid for a start command", fb)
		}
	case AerationActionStop:
		switch fb {
		case FeedbackAcknowledged:
			a.Status = AeratorStatusStopping
		case FeedbackStopped:
			a.Status = AeratorStatusStopped
		case FeedbackFault, FeedbackTimeout:
			a.Status = AeratorStatusFault
		default:
			return a.Status, InvalidInput("feedback %s is not valid for a stop command", fb)
		}
	}
	return a.Status, nil
}

// HasTerminalFeedback reports whether the command received a terminal
// device response (started / stopped / fault / timeout).
func (a *AerationLog) HasTerminalFeedback() bool {
	switch a.Feedback {
	case FeedbackStarted, FeedbackStopped, FeedbackFault, FeedbackTimeout:
		return true
	}
	return false
}

// TimedOut reports whether the command has not received terminal feedback
// within the given timeout (a bare acknowledgment does not count).
func (a *AerationLog) TimedOut(timeout time.Duration, now time.Time) bool {
	if a.HasTerminalFeedback() {
		return false
	}
	return now.Sub(a.CommandTime) > timeout
}

// MarkTimeout moves the command to the fault state with a timeout feedback.
func (a *AerationLog) MarkTimeout(now time.Time) error {
	if a.HasTerminalFeedback() {
		return Conflict("aeration %s already has terminal feedback %s", a.ID, a.Feedback)
	}
	a.Feedback = FeedbackTimeout
	ts := now
	a.FeedbackAt = &ts
	a.Status = AeratorStatusFault
	return nil
}

// IsActive reports whether the aerator is energised (starting or running).
func (a *AerationLog) IsActive() bool {
	return a.Status == AeratorStatusStarting || a.Status == AeratorStatusRunning
}

// aeratorTransitionTable encodes the legal aerator status transitions.
// Self-transitions are always allowed.
//
//	stopped  → starting
//	starting → running, fault
//	running  → stopping, fault
//	stopping → stopped, fault
//	fault    → stopped (manual reset after repair)
var aeratorTransitionTable = map[AeratorStatus]map[AeratorStatus]bool{
	AeratorStatusStopped:  {AeratorStatusStarting: true},
	AeratorStatusStarting: {AeratorStatusRunning: true, AeratorStatusFault: true},
	AeratorStatusRunning:  {AeratorStatusStopping: true, AeratorStatusFault: true},
	AeratorStatusStopping: {AeratorStatusStopped: true, AeratorStatusFault: true},
	AeratorStatusFault:    {AeratorStatusStopped: true},
}

// CanAeratorTransition reports whether moving from `from` to `to` is legal
// in the aerator state machine.
func CanAeratorTransition(from, to AeratorStatus) bool {
	if from == to {
		return true
	}
	return aeratorTransitionTable[from][to]
}

// AllowedAeratorTransitionsFrom returns the legal next states of `from`.
func AllowedAeratorTransitionsFrom(from AeratorStatus) []AeratorStatus {
	out := make([]AeratorStatus, 0, len(aeratorTransitionTable[from]))
	for s := range aeratorTransitionTable[from] {
		out = append(out, s)
	}
	return out
}

// String returns a compact description used in audit entries.
func (a *AerationLog) String() string {
	return fmt.Sprintf("aeration %s (%s/%s)", a.ID, a.Action, a.Status)
}
