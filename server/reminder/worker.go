package reminder

import (
	"context"
	"github.com/alexanderp4580/Minder/server/store"
	"log"
	"strconv"
	"time"
)

type Sender interface {
	Send(context.Context, string, map[string]string) (bool, error)
}
type Worker struct {
	Store  *store.Store
	Sender Sender
}

func (w *Worker) Tick(ctx context.Context, now time.Time) error {
	if e := w.Store.DueReminders(now); e != nil {
		return e
	}
	if w.Sender == nil {
		return nil
	}
	deliveries, e := w.Store.Deliveries(now)
	if e != nil {
		return e
	}
	for _, d := range deliveries {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		current, e := w.Store.DeliveryCurrent(d)
		if e != nil {
			return e
		}
		if !current {
			continue
		}
		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		invalid, sendError := w.Sender.Send(timeout, d.Token, map[string]string{"reminder_id": d.OccurrenceID, "title": d.Title, "scheduled_at": d.ScheduledAt, "late": strconv.FormatBool(d.Late)})
		cancel()
		if e = w.Store.FinishDelivery(d, now, sendError == nil, invalid); e != nil {
			return e
		}
	}
	return nil
}
func (w *Worker) Run(ctx context.Context) {
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		if e := w.Tick(ctx, time.Now().UTC()); e != nil && ctx.Err() == nil {
			log.Print("Reminder worker could not finish its tick")
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
