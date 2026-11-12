package reminder

import (
	"context"
	"log"
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
type Scheduler interface{ ScheduleDue(time.Time) error }

type Worker struct {
	Repository DeliveryRepository
	Scheduler  Scheduler
	Sender     Sender
}

func (w *Worker) Tick(ctx context.Context, now time.Time) error {
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
	for _, d := range deliveries {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		current, e := w.Repository.DeliveryCurrent(d)
		if e != nil {
			return e
		}
		if !current {
			continue
		}
		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		invalid, sendError := w.Sender.Send(timeout, d.Token, map[string]string{"reminder_id": d.OccurrenceID, "title": d.Title, "scheduled_at": d.ScheduledAt, "late": strconv.FormatBool(d.Late)})
		cancel()
		if e = w.Repository.FinishDelivery(d, now.Add(time.Minute), sendError == nil, invalid); e != nil {
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
