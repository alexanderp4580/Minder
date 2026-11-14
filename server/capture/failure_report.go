package capture

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// postFailureReport appends one visible report per failed revision. The email card keeps
// the source and Retry action; this message puts the failure at the current reading position.
func postFailureReport(tx port.Transaction, c domain.Capture) (bool, error) {
	id := fmt.Sprintf("capture-failure-%s-%d", c.ID, c.Revision)
	if _, err := tx.Entry(id); err == nil {
		return false, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return false, err
	}
	text := c.Error
	if text == "" {
		text = domain.ReasonOther.Sentence()
	}
	_, err := tx.PutEntry(domain.Entry{ID: id, Kind: "minder", CaptureID: c.ID, Text: text})
	return err == nil, err
}

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
			added, err := postFailureReport(tx, c)
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
