package serper

import (
	"encoding/json"
	"errors"

	"github.com/alexanderp4580/Minder/server/assistant"
)

const maxHits = 3

// firstHits keeps at most maxHits and never returns nil.
func firstHits(hits []assistant.Hit) []assistant.Hit {
	if len(hits) > maxHits {
		hits = hits[:maxHits]
	}
	return append([]assistant.Hit{}, hits...)
}

// decodeVenues reads a maps or places reply. Maps names the category type; places names it category.
func decodeVenues(body []byte) ([]assistant.Hit, error) {
	var reply struct {
		Places []struct {
			Title    string  `json:"title"`
			Address  string  `json:"address"`
			Phone    string  `json:"phoneNumber"`
			Type     string  `json:"type"`
			Category string  `json:"category"`
			Website  string  `json:"website"`
			Rating   float64 `json:"rating"`
		} `json:"places"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		return nil, errors.New("the lookup reply was unreadable")
	}
	hits := make([]assistant.Hit, 0, len(reply.Places))
	for _, p := range reply.Places {
		category := p.Type
		if category == "" {
			category = p.Category
		}
		hits = append(hits, assistant.Hit{Title: p.Title, Address: p.Address, Phone: p.Phone, Category: category, Website: p.Website, Rating: p.Rating})
	}
	return firstHits(hits), nil
}

// decodeWeb reads a web search reply.
func decodeWeb(body []byte) ([]assistant.Hit, error) {
	var reply struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		return nil, errors.New("the lookup reply was unreadable")
	}
	hits := make([]assistant.Hit, 0, len(reply.Organic))
	for _, o := range reply.Organic {
		hits = append(hits, assistant.Hit{Title: o.Title, Website: o.Link, Snippet: o.Snippet})
	}
	return firstHits(hits), nil
}
