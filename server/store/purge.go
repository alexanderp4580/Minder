package store

import (
	"log/slog"
	"sort"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// PurgeItem deletes an item and the data that exists only because of it.
// It never logs titles or text.
//
// Pending and Audit rows have no JSON tags, so the queries below use the Go field
// names ($.Proposal.item_id, $.After.id). TestPurgeItemRemovesItsDataAndKeepsOthers
// fails if a tag changes them.
func (t *Tx) PurgeItem(id, keepPending string) (port.Purged, error) {
	var out port.Purged
	entries := map[string]bool{}
	changes, e := t.purgeChanges(id, keepPending, entries)
	if e != nil {
		return out, e
	}
	if e = t.collectOccurrenceEntries(id, entries); e != nil {
		return out, e
	}
	sources, e := t.SourcesForItem(id)
	if e != nil {
		return out, e
	}
	// DeleteItem closes reminders, which can rewrite their entries, so entries go last.
	if e = t.DeleteItem(id); e != nil {
		return out, e
	}
	if e = t.deleteItemRows(id); e != nil {
		return out, e
	}
	if out.OrphanSources, e = t.unusedSources(sources); e != nil {
		return out, e
	}
	if out.EntryIDs, e = t.deleteEntries(entries); e != nil {
		return out, e
	}
	slog.Info("item purged", "item_id", id, "entries", len(out.EntryIDs), "changes", changes, "orphan_sources", len(out.OrphanSources))
	return out, nil
}

// purgeChanges deletes the pending changes of an item, except keepPending, and notes their entries.
func (t *Tx) purgeChanges(id, keepPending string, entries map[string]bool) (int, error) {
	changes, e := queryAll[domain.Pending](t.tx, "SELECT body FROM pending_change WHERE id<>? AND (json_extract(body,'$.Proposal.item_id')=? OR json_extract(body,'$.After.id')=? OR json_extract(body,'$.Before.id')=?)", keepPending, id, id, id)
	if e != nil {
		return 0, e
	}
	for _, p := range changes {
		noteEntries(entries, p.EntryID, p.ResultID)
		if _, e = t.tx.Exec("DELETE FROM pending_change WHERE id=?", p.ID); e != nil {
			return 0, e
		}
	}
	return len(changes), nil
}

// collectOccurrenceEntries notes the reminder entries of an item.
func (t *Tx) collectOccurrenceEntries(id string, entries map[string]bool) error {
	rows, e := t.tx.Query("SELECT entry_id,result_id FROM reminder_occurrence WHERE item_id=?", id)
	if e != nil {
		return e
	}
	for rows.Next() {
		var entry, result string
		if e = rows.Scan(&entry, &result); e != nil {
			rows.Close()
			return e
		}
		noteEntries(entries, entry, result)
	}
	return rows.Close()
}

// deleteItemRows removes the item's occurrences, audit rows and source links.
func (t *Tx) deleteItemRows(id string) error {
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{"DELETE FROM reminder_occurrence WHERE item_id=?", []any{id}},
		{"DELETE FROM audit WHERE json_extract(body,'$.After.id')=? OR json_extract(body,'$.Before.id')=?", []any{id, id}},
		{"DELETE FROM item_source WHERE item_id=?", []any{id}},
	} {
		if _, e := t.tx.Exec(q.sql, q.args...); e != nil {
			return e
		}
	}
	return nil
}

// unusedSources returns the sources that no remaining item links to.
func (t *Tx) unusedSources(sources []string) ([]string, error) {
	var unused []string
	for _, source := range sources {
		var n int
		if e := t.tx.QueryRow("SELECT COUNT(*) FROM item_source WHERE source_id=?", source).Scan(&n); e != nil {
			return nil, e
		}
		if n == 0 {
			unused = append(unused, source)
		}
	}
	return unused, nil
}

// deleteEntries removes chat entries and their events and returns their IDs in order.
func (t *Tx) deleteEntries(entries map[string]bool) ([]string, error) {
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, q := range []string{"DELETE FROM chat_entry WHERE id=?", "DELETE FROM event WHERE entry_id=?"} {
			if _, e := t.tx.Exec(q, id); e != nil {
				return nil, e
			}
		}
	}
	return ids, nil
}

func noteEntries(entries map[string]bool, ids ...string) {
	for _, id := range ids {
		if id != "" {
			entries[id] = true
		}
	}
}
