package store

import (
	"github.com/alexanderp4580/Minder/server/domain"
)

var builtinProfiles = []domain.Profile{
	{ID: "daily-task", Name: "Daily task", RemindBeforeMinutes: 0, NagMinutes: 15},
	{ID: "chore", Name: "Chore", RemindBeforeMinutes: 0, NagMinutes: 0},
	{ID: "birthday", Name: "Birthday", RemindBeforeMinutes: 1440, NagMinutes: 0},
	{ID: "day-event", Name: "Day event", RemindBeforeMinutes: 60, NagMinutes: 0},
	{ID: "trip", Name: "Trip", RemindBeforeMinutes: 2880, NagMinutes: 0},
}

// seedProfiles adds missing built-in profiles. It never overwrites an edited one.
func (s *Store) seedProfiles() error {
	for _, p := range builtinProfiles {
		p.Builtin, p.Version = true, 1
		if _, e := s.db.Exec("INSERT OR IGNORE INTO profile(id,body) VALUES(?,?)", p.ID, body(p)); e != nil {
			return e
		}
	}
	return nil
}
func (t *Tx) Profile(id string) (domain.Profile, error) {
	return decode[domain.Profile](t.tx.QueryRow("SELECT body FROM profile WHERE id=?", id))
}

// profileOf returns nil for an empty or unknown profile id.
func (t *Tx) profileOf(id string) (*domain.Profile, error) {
	if id == "" {
		return nil, nil
	}
	p, e := t.Profile(id)
	if e == domain.ErrNotFound {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &p, nil
}

// PutProfile saves a profile and reschedules the reminders of items that use it.
func (t *Tx) PutProfile(p domain.Profile) error {
	previous, e := t.profileOf(p.ID)
	if e != nil {
		return e
	}
	if _, e = t.tx.Exec("INSERT INTO profile(id,body) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", p.ID, body(p)); e != nil {
		return e
	}
	items, e := queryAll[domain.Item](t.tx, "SELECT body FROM item WHERE json_extract(body,'$.profile')=?", p.ID)
	if e != nil {
		return e
	}
	for _, i := range items {
		if e = t.applyReminder(i.ID, specOf(i, previous), specOf(i, &p)); e != nil {
			return e
		}
	}
	return nil
}

// DeleteProfile removes a profile that no item uses.
func (t *Tx) DeleteProfile(id string) error {
	var n int
	if e := t.tx.QueryRow("SELECT count(*) FROM item WHERE json_extract(body,'$.profile')=?", id).Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return domain.ErrConflict
	}
	_, e := t.tx.Exec("DELETE FROM profile WHERE id=?", id)
	return e
}
func (s *Store) Profiles() ([]domain.Profile, error) {
	return queryAll[domain.Profile](s.db, "SELECT body FROM profile ORDER BY json_extract(body,'$.builtin') DESC, json_extract(body,'$.name')")
}

func listNames(q queryer) ([]string, error) {
	rows, e := q.Query("SELECT DISTINCT json_extract(body,'$.list') AS n FROM item WHERE n IS NOT NULL AND n<>'' ORDER BY n")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if e = rows.Scan(&n); e != nil {
			return nil, e
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
func (s *Store) Lists() ([]string, error)  { return listNames(s.db) }
func (t *Tx) ListNames() ([]string, error) { return listNames(t.tx) }
