package reminder

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

type Sender interface {
	Send(context.Context, string, map[string]string) (bool, error)
}
type DeliveryRepository interface {
	Deliveries(time.Time) ([]domain.Delivery, error)
	DeliveryCurrent(domain.Delivery) (bool, error)
	FinishDelivery(domain.Delivery, time.Time, bool, bool) error
}

// StatsProvider is optional. When the repository has it, the worker logs queue state in its heartbeat.
type StatsProvider interface {
	ReminderStats(time.Time) (domain.ReminderStats, error)
}
type Scheduler interface{ ScheduleDue(time.Time) error }

type Worker struct {
	Repository DeliveryRepository
	Scheduler  Scheduler
	Sender     Sender
	// Heartbeat is how often a tick also logs queue state. Zero turns it off; Run defaults to five minutes.
	Heartbeat time.Duration
	lastBeat  time.Time
}

func (w *Worker) Tick(ctx context.Context, now time.Time) error {
	started := time.Now()
	found, sent, failed, skipped, removed := 0, 0, 0, 0, 0
	defer func() {
		slog.Debug("reminder tick", "deliveries_found", found, "sent", sent, "failed", failed, "skipped", skipped, "devices_removed", removed, "push_sender", w.Sender != nil, "duration_ms", time.Since(started).Milliseconds())
	}()
	w.heartbeat(now)
	if e := w.Scheduler.ScheduleDue(now); e != nil {
		return e
	}
	if w.Sender == nil {
		return nil
	}
	deliveries, e := w.Repository.Deliveries(now)
	if e != nil {
		return e
	}
	found = len(deliveries)
	for _, d := range deliveries {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		current, e := w.Repository.DeliveryCurrent(d)
		if e != nil {
			return e
		}
		if !current {
			skipped++
			slog.Debug("push delivery skipped", "reason", "occurrence closed or token changed", "occurrence_id", d.OccurrenceID, "device_id", d.DeviceID)
			continue
		}
		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		attempt := time.Now()
		invalid, sendError := w.Sender.Send(timeout, d.Token, map[string]string{"reminder_id": d.OccurrenceID, "title": d.Title, "scheduled_at": d.ScheduledAt, "late": strconv.FormatBool(d.Late)})
		cancel()
		retryAt := now.Add(time.Minute)
		switch {
		case invalid:
			removed++
			slog.Warn("push device removed", "reason", "token rejected by Firebase", "device_id", d.DeviceID, "token_fp", domain.Fingerprint(d.Token), "occurrence_id", d.OccurrenceID, "error", sendError)
		case sendError != nil:
			failed++
			slog.Warn("push delivery failed", "occurrence_id", d.OccurrenceID, "device_id", d.DeviceID, "token_fp", domain.Fingerprint(d.Token), "error", sendError, "retry_at", retryAt.Format(time.RFC3339), "attempt_ms", time.Since(attempt).Milliseconds(), "late", d.Late)
		default:
			sent++
			slog.Info("push delivered", "occurrence_id", d.OccurrenceID, "device_id", d.DeviceID, "token_fp", domain.Fingerprint(d.Token), "scheduled_at", d.ScheduledAt, "late", d.Late, "attempt_ms", time.Since(attempt).Milliseconds())
		}
		if e = w.Repository.FinishDelivery(d, retryAt, sendError == nil, invalid); e != nil {
			return e
		}
	}
	return nil
}

// heartbeat logs queue state now and then, so a quiet worker still leaves evidence of what it is holding.
func (w *Worker) heartbeat(now time.Time) {
	if w.Heartbeat <= 0 || now.Sub(w.lastBeat) < w.Heartbeat {
		return
	}
	w.lastBeat = now
	attrs := []any{"push_sender", w.Sender != nil}
	if provider, ok := w.Repository.(StatsProvider); ok {
		stats, e := provider.ReminderStats(now)
		if e != nil {
			slog.Error("reminder worker heartbeat failed", "error", e)
			return
		}
		attrs = append(attrs, "items", stats.Items, "devices", stats.Devices, "schedules_active", stats.SchedulesActive, "occurrences_active", stats.OccurrencesActive, "deliveries_pending", stats.DeliveriesPending, "deliveries_due", stats.DeliveriesDue, "oldest_due_seconds", stats.OldestDueSeconds, "next_schedule_at", stats.NextScheduleAt)
	}
	slog.Info("reminder worker heartbeat", attrs...)
}
func (w *Worker) runOnce(ctx context.Context) {
	if e := w.Tick(ctx, time.Now().UTC()); e != nil {
		if ctx.Err() != nil {
			slog.Debug("reminder worker tick interrupted", "error", e)
			return
		}
		slog.Error("reminder worker tick failed", "error", e)
	}
}
func (w *Worker) Run(ctx context.Context) {
	if w.Heartbeat == 0 {
		w.Heartbeat = 5 * time.Minute
	}
	slog.Info("reminder worker started", "interval_s", 15, "heartbeat_s", int(w.Heartbeat.Seconds()), "push_sender", w.Sender != nil)
	defer slog.Info("reminder worker stopped")
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		w.runOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
