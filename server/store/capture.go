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
