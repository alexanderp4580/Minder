package store

import (
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"time"
)

func captureOne[T any](q queryer, table, id string) (out T, err error) {
	rows, e := queryAll[T](q, "SELECT body FROM "+table+" WHERE id=?", id)
	if e != nil {
		return out, e
	}
	if len(rows) == 0 {
		return out, domain.ErrNotFound
	}
	return rows[0], nil
}
func (t *Tx) Capture(id string) (domain.Capture, error) {
	return captureOne[domain.Capture](t.tx, "capture", id)
}
func (s *Store) Capture(id string) (domain.Capture, error) {
	return captureOne[domain.Capture](s.db, "capture", id)
}
func (t *Tx) PutCapture(c domain.Capture) error {
	b, e := json.Marshal(c)
	if e != nil {
		return e
	}
	_, e = t.tx.Exec("INSERT INTO capture(id,body) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", c.ID, string(b))
	return e
}
func (t *Tx) Receipt(id string) (domain.CaptureReceipt, error) {
	return captureOne[domain.CaptureReceipt](t.tx, "capture_receipt", id)
}
func (t *Tx) PutReceipt(r domain.CaptureReceipt) error {
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	_, e = t.tx.Exec("INSERT INTO capture_receipt(id,body) VALUES(?,?)", r.ID, string(b))
	return e
}
func (t *Tx) LinkSource(item, source string) error {
	_, e := t.tx.Exec("INSERT OR IGNORE INTO item_source(item_id,source_id) VALUES(?,?)", item, source)
	return e
}
func (t *Tx) SourcesForItem(id string) ([]string, error) {
	rows, e := t.tx.Query("SELECT source_id FROM item_source WHERE item_id=? ORDER BY source_id", id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if e := rows.Scan(&id); e != nil {
			return nil, e
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func (s *Store) ClaimCapture(now time.Time) (out domain.Capture, err error) {
	err = s.Transact(func(tx port.Transaction) error {
		// Store's transaction serializes claims; use its concrete transaction for this storage query only.
		rows, e := queryAll[domain.Capture](tx.(*Tx).tx, "SELECT body FROM capture WHERE json_extract(body,'$.status') IN ('received','processing') ORDER BY rowid")
		if e != nil {
			return e
		}
		for _, c := range rows {
			if c.Due(now) {
				c.Status = "processing"
				c.LeaseUntil = now.Add(5 * time.Minute).Unix()
				out = c
				return tx.PutCapture(c)
			}
		}
		return domain.ErrNotFound
	})
	return
}

func (t *Tx) RedactCapture(id string) error {
	for _, table := range []string{"chat_entry", "event"} {
		rows, e := queryAll[domain.Entry](t.tx, "SELECT body FROM "+table+" WHERE json_extract(body,'$.capture.id')=?", id)
		if e != nil {
			return e
		}
		for _, entry := range rows {
			entry.Capture = &domain.Capture{ID: id, Status: "reviewed", Deleted: true}
			entry.Text = "Email source deleted. Confirmed items remain."
			raw, e := json.Marshal(entry)
			if e != nil {
				return e
			}
			key, column := entry.ID, "id"
			if table == "event" {
				key, column = entry.ID, "entry_id"
			}
			if _, e = t.tx.Exec("UPDATE "+table+" SET body=? WHERE "+column+"=?", string(raw), key); e != nil {
				return e
			}
		}
	}
	pending, e := queryAll[domain.Pending](t.tx, "SELECT body FROM pending_change WHERE json_extract(body,'$.Proposal.capture_id')=?", id)
	if e != nil {
		return e
	}
	for _, p := range pending {
		if p.State == "confirmed" || p.Proposal.Action == "delete_capture" {
			continue
		}
		p.State = "failed"
		p.Proposal = domain.Proposal{CaptureID: id}
		p.Before = nil
		p.After = domain.Item{}
		if e = t.PutPending(p); e != nil {
			return e
		}
		entry, e := t.Entry(p.EntryID)
		if e != nil {
			return e
		}
		entry.Text = "Email source deleted."
		entry.Change.State = "failed"
		entry.Change.Fields = nil
		if _, e = t.PutEntry(entry); e != nil {
			return e
		}
	}
	_, e = t.tx.Exec("DELETE FROM capture_push WHERE source_id=?", id)
	return e
}
func (t *Tx) QueueCaptureNotifications(id string, revision int) error {
	_, e := t.tx.Exec("INSERT OR IGNORE INTO capture_push(source_id,revision,device_id) SELECT ?,?,id FROM push_device", id, revision)
	return e
}
func (s *Store) CaptureDeliveries(now time.Time) ([]domain.CaptureDelivery, error) {
	rows, e := s.db.Query("SELECT n.source_id,n.revision,n.device_id,d.token FROM capture_push n JOIN push_device d ON d.id=n.device_id JOIN capture c ON c.id=n.source_id WHERE n.sent=0 AND n.next_at<=? AND coalesce(json_extract(c.body,'$.deleted'),0)=0 AND json_extract(c.body,'$.status') IN ('needs-answer','ready-for-review') AND n.revision=json_extract(c.body,'$.revision')", now.Unix())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.CaptureDelivery{}
	for rows.Next() {
		var d domain.CaptureDelivery
		if e = rows.Scan(&d.CaptureID, &d.Revision, &d.DeviceID, &d.Token); e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s *Store) FinishCaptureDelivery(d domain.CaptureDelivery, invalid bool, next time.Time) error {
	if invalid {
		_, e := s.db.Exec("DELETE FROM push_device WHERE id=? AND token=?", d.DeviceID, d.Token)
		return e
	}
	sent := 1
	if !next.IsZero() {
		sent = 0
	}
	_, e := s.db.Exec("UPDATE capture_push SET sent=?,next_at=? WHERE source_id=? AND revision=? AND device_id=?", sent, next.Unix(), d.CaptureID, d.Revision, d.DeviceID)
	return e
}
func (s *Store) DeletedCaptures() ([]domain.Capture, error) {
	return queryAll[domain.Capture](s.db, "SELECT body FROM capture WHERE json_extract(body,'$.deleted')=1 AND coalesce(json_extract(body,'$.files_deleted'),0)=0")
}

func (s *Store) CaptureIDs() (map[string]bool, error) {
	rows, e := s.db.Query("SELECT id FROM capture")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	known := map[string]bool{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		known[id] = true
	}
	return known, rows.Err()
}

func (s *Store) PendingEmailCandidates(exclude string) ([]domain.CaptureCandidate, error) {
	sources, e := queryAll[domain.Capture](s.db, "SELECT body FROM capture WHERE id<>? AND coalesce(json_extract(body,'$.deleted'),0)=0 ORDER BY rowid DESC LIMIT 200", exclude)
	if e != nil {
		return nil, e
	}
	out := []domain.CaptureCandidate{}
	for _, source := range sources {
		for _, candidate := range source.Candidates {
			if candidate.State == "pending" || candidate.State == "needs-answer" {
				out = append(out, candidate)
				if len(out) >= 50 {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

func (s *Store) itemSourceIDs(item string) ([]string, error) {
	rows, e := s.db.Query("SELECT link.source_id FROM item_source link JOIN capture c ON c.id=link.source_id WHERE link.item_id=? AND coalesce(json_extract(c.body,'$.deleted'),0)=0 ORDER BY link.source_id", item)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
