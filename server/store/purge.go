package store

import (
	"log/slog"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// PurgeItem deletes an item and the data that exists only because of it.
// It never logs titles or text.
func (t *Tx) PurgeItem(id, keepPending string) (port.Purged, error) {
	var out port.Purged
	entries := map[string]bool{}
	add := func(ids ...string) {
		for _, e := range ids {
			if e != "" {
				entries[e] = true
			}
		}
	}
	changes, e := queryAll[domain.Pending](t.tx, "SELECT body FROM pending_change WHERE id<>? AND (json_extract(body,'$.Proposal.item_id')=? OR json_extract(body,'$.After.id')=? OR json_extract(body,'$.Before.id')=?)", keepPending, id, id, id)
	if e != nil {
		return out, e
	}
	for _, p := range changes {
		add(p.EntryID, p.ResultID)
		if _, e = t.tx.Exec("DELETE FROM pending_change WHERE id=?", p.ID); e != nil {
			return out, e
		}
	}
	rows, e := t.tx.Query("SELECT entry_id,result_id FROM reminder_occurrence WHERE item_id=?", id)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var entry, result string
		if e = rows.Scan(&entry, &result); e != nil {
			rows.Close()
			return out, e
		}
		add(entry, result)
	}
	if e = rows.Close(); e != nil {
		return out, e
	}
	sources, e := t.SourcesForItem(id)
	if e != nil {
		return out, e
	}
	// DeleteItem closes reminders, which may rewrite their entries; remove entries last.
	if e = t.DeleteItem(id); e != nil {
		return out, e
	}
	if _, e = t.tx.Exec("DELETE FROM reminder_occurrence WHERE item_id=?", id); e != nil {
		return out, e
	}
	if _, e = t.tx.Exec("DELETE FROM audit WHERE json_extract(body,'$.After.id')=? OR json_extract(body,'$.Before.id')=?", id, id); e != nil {
		return out, e
	}
	if _, e = t.tx.Exec("DELETE FROM item_source WHERE item_id=?", id); e != nil {
		return out, e
	}
	for _, source := range sources {
		var n int
		if e = t.tx.QueryRow("SELECT COUNT(*) FROM item_source WHERE source_id=?", source).Scan(&n); e != nil {
			return out, e
		}
		if n == 0 {
			out.OrphanSources = append(out.OrphanSources, source)
		}
	}
	for entry := range entries {
		for _, q := range []string{"DELETE FROM chat_entry WHERE id=?", "DELETE FROM event WHERE entry_id=?"} {
			if _, e = t.tx.Exec(q, entry); e != nil {
				return out, e
			}
		}
		out.EntryIDs = append(out.EntryIDs, entry)
	}
	slog.Info("item purged", "item_id", id, "entries", len(out.EntryIDs), "changes", len(changes), "orphan_sources", len(out.OrphanSources))
	return out, nil
}
