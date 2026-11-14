package capture

import (
	"errors"
	"log/slog"
	"sort"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/review"
)

// restoreFailureReports exposes failures recorded before chat reports existed. It neither
// retries extraction nor queues another notification for an already failed revision.
func (s *Service) restoreFailureReports() error {
	repository, ok := s.repository.(interface {
		CaptureIDs() (map[string]bool, error)
	})
	if !ok {
		return nil
	}
	known, err := repository.CaptureIDs()
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(known))
	for id := range known {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	restored := 0
	err = s.repository.Transact(func(tx port.Transaction) error {
		for _, id := range ids {
			c, err := tx.Capture(id)
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if c.Deleted || c.Status != "failed" {
				continue
			}
			added, err := review.PostFailure(tx, c)
			if err != nil {
				return err
			}
			if added {
				restored++
			}
		}
		return nil
	})
	if err == nil && restored > 0 {
		slog.Info("email failure reports restored", "reports", restored)
	}
	return err
}
