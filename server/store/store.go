package store

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"sync"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// steps returns the migration files in order. File n takes the database from version n-1 to n.
func steps() []string {
	names, _ := fs.Glob(migrations, "migrations/*.sql")
	sort.Strings(names)
	return names
}

// migrate upgrades the database to the newest version. It refuses a database from a newer release.
func migrate(db *sql.DB) error {
	var current int
	if e := db.QueryRow("PRAGMA user_version").Scan(&current); e != nil {
		return e
	}
	all := steps()
	if current > len(all) {
		return errors.New("The database is newer than this server.")
	}
	for v := current; v < len(all); v++ {
		script, e := migrations.ReadFile(all[v])
		if e != nil {
			return e
		}
		tx, e := db.Begin()
		if e != nil {
			return e
		}
		if _, e = tx.Exec(string(script)); e == nil {
			_, e = tx.Exec(fmt.Sprintf("PRAGMA user_version=%d", v+1))
		}
		if e != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", all[v], e)
		}
		if e = tx.Commit(); e != nil {
			return e
		}
	}
	return nil
}

type Store struct {
	db *sql.DB
	mu sync.Mutex
}
type Tx struct{ tx *sql.Tx }

func ID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func Open(path string) (*Store, error) {
	db, e := sql.Open("sqlite3", path+"?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL")
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	if e = migrate(db); e != nil {
		db.Close()
		return nil, e
	}
	st := &Store{db: db}
	if e = st.seedProfiles(); e != nil {
		db.Close()
		return nil, e
	}
	return st, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Write(fn func(*Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = fn(&Tx{tx}); e != nil {
		return e
	}
	return tx.Commit()
}

// queryer is what *sql.DB and *sql.Tx share.
type queryer interface {
	Query(string, ...any) (*sql.Rows, error)
}

func unmarshal[T any](raw string) (v T, e error) {
	e = json.Unmarshal([]byte(raw), &v)
	return
}

// queryAll runs a query that returns one JSON column and decodes every row into T.
func queryAll[T any](q queryer, query string, args ...any) ([]T, error) {
	rows, e := q.Query(query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		v, e := unmarshal[T](raw)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func decode[T any](row *sql.Row) (T, error) {
	var v T
	var b string
	e := row.Scan(&b)
	if e == sql.ErrNoRows {
		return v, domain.ErrNotFound
	}
	if e != nil {
		return v, e
	}
	e = json.Unmarshal([]byte(b), &v)
	return v, e
}
func body(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return string(b)
}
func (t *Tx) Item(id string) (domain.Item, error) {
	return decode[domain.Item](t.tx.QueryRow("SELECT body FROM item WHERE id=?", id))
}
func (t *Tx) PutItem(i domain.Item) error {
	var old *domain.Item
	previous, err := t.Item(i.ID)
	if err == nil {
		old = &previous
	} else if err != domain.ErrNotFound {
		return err
	}

	_, e := t.tx.Exec("INSERT INTO item(id,body) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", i.ID, body(i))
	if e != nil {
		return e
	}
	return t.syncReminder(old, i)
}
func (t *Tx) DeleteItem(id string) error {
	if e := t.closeReminders(id, "cancelled"); e != nil {
		return e
	}
	if _, e := t.tx.Exec("DELETE FROM reminder_schedule WHERE item_id=?", id); e != nil {
		return e
	}

	_, e := t.tx.Exec("DELETE FROM item WHERE id=?", id)
	return e
}
func (t *Tx) Pending(id string) (domain.Pending, error) {
	return decode[domain.Pending](t.tx.QueryRow("SELECT body FROM pending_change WHERE id=?", id))
}
func (t *Tx) PutPending(p domain.Pending) error {
	_, e := t.tx.Exec("INSERT INTO pending_change(id,body) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", p.ID, body(p))
	return e
}
func (t *Tx) Audit(id string) (domain.Audit, error) {
	return decode[domain.Audit](t.tx.QueryRow("SELECT body FROM audit WHERE id=?", id))
}
func (t *Tx) PutAudit(a domain.Audit) error {
	_, e := t.tx.Exec("INSERT INTO audit(id,actor,body) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", a.ID, a.Actor, body(a))
	return e
}
func (t *Tx) Entry(id string) (domain.Entry, error) {
	return decode[domain.Entry](t.tx.QueryRow("SELECT body FROM chat_entry WHERE id=?", id))
}
func (t *Tx) PutEntry(e domain.Entry) (domain.Entry, error) {
	if e.ID == "" {
		e.ID = ID()
	}
	if e.CreatedAt == "" {
		e.CreatedAt = domain.Now()
	}
	r, err := t.tx.Exec("INSERT INTO event(entry_id,body) VALUES(?,?)", e.ID, "{}")
	if err != nil {
		return e, err
	}
	e.Cursor, err = r.LastInsertId()
	if err != nil {
		return e, err
	}
	if _, err = t.tx.Exec("UPDATE event SET body=? WHERE cursor=?", body(e), e.Cursor); err != nil {
		return e, err
	}
	_, err = t.tx.Exec("INSERT INTO chat_entry(id,body) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", e.ID, body(e))
	return e, err
}

// AttachCost records what the model request behind an entry cost.
func (s *Store) AttachCost(id string, cost float64) (out domain.Entry, err error) {
	err = s.Write(func(t *Tx) error {
		entry, e := t.Entry(id)
		if e != nil {
			return e
		}
		entry.CostEUR = cost
		out, e = t.PutEntry(entry)
		return e
	})
	return
}
func (s *Store) AddEntry(e domain.Entry) (out domain.Entry, err error) {
	err = s.Write(func(t *Tx) error { var e2 error; out, e2 = t.PutEntry(e); return e2 })
	return
}
func (s *Store) Items() ([]domain.Item, error) {
	return queryAll[domain.Item](s.db, "SELECT body FROM item ORDER BY id")
}

// ItemsBetween returns items placed in [from, to), sorted by their placement time.
func (s *Store) ItemsBetween(from, to time.Time) ([]domain.Item, error) {
	all, e := s.Items()
	if e != nil {
		return nil, e
	}
	out := []domain.Item{}
	for _, i := range all {
		at, e := time.Parse(time.RFC3339, domain.PlacedAt(i))
		if e == nil && !at.Before(from) && at.Before(to) {
			out = append(out, i)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return domain.PlacedAt(out[a]) < domain.PlacedAt(out[b]) })
	return out, nil
}
func (s *Store) History(after int64, limit int) ([]domain.Entry, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if after == 0 {
		return queryAll[domain.Entry](s.db, "SELECT body FROM (SELECT seq,body FROM chat_entry ORDER BY seq DESC LIMIT ?) ORDER BY seq", limit)
	}
	return queryAll[domain.Entry](s.db, "SELECT body FROM event WHERE cursor>? ORDER BY cursor LIMIT ?", after, limit)
}
func (s *Store) Events(after int64, limit int) ([]domain.Entry, error) {
	if after == 0 {
		after = -1
	}
	return s.History(after, limit)
}
func (s *Store) AuditCount() (n int, e error) {
	e = s.db.QueryRow("SELECT COUNT(*) FROM audit").Scan(&n)
	return
}
func month() string { return time.Now().UTC().Format("2006-01") }
func (s *Store) MonthCost() (n float64, e error) {
	e = s.db.QueryRow("SELECT COALESCE(SUM(cost_eur),0) FROM usage WHERE month=?", month()).Scan(&n)
	return
}
func (s *Store) RecordUsage(cost float64, v any) error {
	if cost < 0 {
		return fmt.Errorf("invalid cost")
	}
	return s.Write(func(t *Tx) error {
		_, e := t.tx.Exec("INSERT INTO usage(id,month,cost_eur,body) VALUES(?,?,?,?)", ID(), month(), cost, body(v))
		return e
	})
}

// Reserve before a provider call. A crash or unknown provider cost keeps the reservation.
func (s *Store) ReserveUsage(limit, amount float64) (id string, err error) {
	id = ID()
	err = s.Write(func(t *Tx) error {
		var total float64
		if e := t.tx.QueryRow("SELECT COALESCE(SUM(cost_eur),0) FROM usage WHERE month=?", month()).Scan(&total); e != nil {
			return e
		}
		if total+amount > limit {
			return domain.ErrLimit
		}
		_, e := t.tx.Exec("INSERT INTO usage(id,month,cost_eur,body) VALUES(?,?,?,?)", id, month(), amount, body(map[string]any{"reserved": true, "cost_eur": amount}))
		return e
	})
	return
}
func (s *Store) SettleUsage(id string, cost float64, value any) error {
	if cost < 0 {
		return fmt.Errorf("invalid cost")
	}
	return s.Write(func(t *Tx) error {
		_, e := t.tx.Exec("UPDATE usage SET cost_eur=?,body=? WHERE id=?", cost, body(value), id)
		return e
	})
}
