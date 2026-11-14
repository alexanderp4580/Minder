package assistant

import (
	"context"

	"github.com/alexanderp4580/Minder/server/domain"
)

// LookupKind and its values come from the domain, which owns the offer format.
type LookupKind = domain.LookupKind

const (
	LookupMaps   = domain.LookupMaps
	LookupPlaces = domain.LookupPlaces
	LookupSearch = domain.LookupSearch
)

// Hit is one trimmed search result. Each kind fills the fields it has.
type Hit struct {
	Title    string  `json:"title,omitempty"`
	Address  string  `json:"address,omitempty"`
	Phone    string  `json:"phone,omitempty"`
	Category string  `json:"category,omitempty"`
	Website  string  `json:"website,omitempty"`
	Snippet  string  `json:"snippet,omitempty"`
	Rating   float64 `json:"rating,omitempty"`
}

// Finder runs one search of the given kind.
type Finder interface {
	Lookup(ctx context.Context, kind LookupKind, query string) ([]Hit, error)
}

// KindInfo tells the model what one kind of search costs and returns.
type KindInfo struct {
	Kind        LookupKind
	Description string
}

// LookupKinds lists the kinds the model may offer, in the order it sees them.
var LookupKinds = []KindInfo{
	{LookupMaps, "Google Maps search, 3 credits. Returns up to three places with title, full address (street, postal code, town, country), phone, website, category and rating. Use it for a venue's address or phone number."},
	{LookupPlaces, "Google local result, 1 credit. Returns the single best match with title, street address (postal code and town are often missing), website, category and rating. Use it to confirm that a venue exists."},
	{LookupSearch, "Google web search, 1 credit. Returns up to three pages with title, link and snippet, and no structured address or phone. Use it for booking pages, opening hours or anything that is not one place."},
}
