package store

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/alexanderp4580/Minder/server/domain"
)

// maxSources bounds one FindSources answer. More matches ask for a narrower query.
const maxSources = 20

func (t *Tx) Source(id string) (domain.SourceRef, error) {
	return decode[domain.SourceRef](t.tx.QueryRow("SELECT body FROM source WHERE id=?", id))
}

func (t *Tx) PutSource(ref domain.SourceRef) error {
	raw, err := json.Marshal(ref)
	if err != nil {
		return err
	}
	_, err = t.tx.Exec("INSERT INTO source(id,body) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", ref.ID, string(raw))
	return err
}

// likePattern matches text containing query, with LIKE's wildcards taken literally.
func likePattern(query string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(query) + "%"
}

// FindSources lists retained sources whose label or sender contains query, plus owner chat
// messages that contain it, newest first. Deleted sources are left out. It refuses a blank
// query and a query with more than maxSources matches, so no match is dropped silently.
func (s *Store) FindSources(query string) ([]domain.SourceRef, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, &domain.Invalid{Field: "query", Message: "Give a word from the subject, sender or text."}
	}
	pattern := likePattern(query)
	refs, err := queryAll[domain.SourceRef](s.db, `SELECT body FROM source WHERE coalesce(json_extract(body,'$.deleted'),0)=0
		AND (json_extract(body,'$.label') LIKE ?1 ESCAPE '\' OR json_extract(body,'$.sender') LIKE ?1 ESCAPE '\')
		ORDER BY json_extract(body,'$.created_at') DESC LIMIT ?2`, pattern, maxSources+1)
	if err != nil {
		return nil, err
	}
	entries, err := queryAll[domain.Entry](s.db, `SELECT body FROM chat_entry WHERE json_extract(body,'$.kind')='user'
		AND json_extract(body,'$.text') LIKE ?1 ESCAPE '\' ORDER BY json_extract(body,'$.created_at') DESC LIMIT ?2`, pattern, maxSources+1)
	if err != nil {
		return nil, err
	}
	found := map[string]domain.SourceRef{}
	for _, ref := range refs {
		found[ref.ID] = ref
	}
	for _, entry := range entries {
		// A chat message that already has a source record is listed by that record, and not once deleted.
		published, err := decode[domain.SourceRef](s.db.QueryRow("SELECT body FROM source WHERE id=?", entry.ID))
		switch {
		case err == nil && !published.Deleted:
			found[entry.ID] = published
		case errors.Is(err, domain.ErrNotFound):
			found[entry.ID] = domain.SourceRef{ID: entry.ID, Kind: domain.SourceChat, EntryID: entry.ID, Label: domain.ChatLabel(entry.Text), CreatedAt: entry.CreatedAt}
		case err != nil:
			return nil, err
		}
	}
	out := make([]domain.SourceRef, 0, len(found))
	for _, ref := range found {
		out = append(out, ref)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].CreatedAt > out[b].CreatedAt })
	if len(out) > maxSources {
		return nil, &domain.Invalid{Field: "query", Message: "More than 20 sources match. Use a narrower word."}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
