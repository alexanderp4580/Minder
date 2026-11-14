package assistant

import "context"

// LookupKind names one way of searching the web for the owner.
type LookupKind string

const (
	LookupMaps   LookupKind = "maps"
	LookupPlaces LookupKind = "places"
	LookupSearch LookupKind = "search"
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

// LookupTool is a tool the model may call after the owner approves a lookup.
// The description tells the model what the search costs and returns.
type LookupTool struct {
	Name        string
	Kind        LookupKind
	Description string
}

// LookupTools is the catalogue of lookup tools, in the order the model sees them.
var LookupTools = []LookupTool{
	{"lookup_maps", LookupMaps, "Google Maps search, 3 credits. Returns up to three places with title, full address (street, postal code, town, country), phone, website, category and rating. Use it for a venue's address or phone number."},
	{"lookup_places", LookupPlaces, "Google local result, 1 credit. Returns the single best match with title, street address (postal code and town are often missing), website, category and rating. Use it to confirm that a venue exists."},
	{"lookup_search", LookupSearch, "Google web search, 1 credit. Returns up to three pages with title, link and snippet, and no structured address or phone. Use it for booking pages, opening hours or anything that is not one place."},
}

// LookupToolKind returns the kind a lookup tool name stands for.
func LookupToolKind(name string) (LookupKind, bool) {
	for _, tool := range LookupTools {
		if tool.Name == name {
			return tool.Kind, true
		}
	}
	return "", false
}
