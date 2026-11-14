package serper

import "github.com/alexanderp4580/Minder/server/assistant"

// endpoint is one Serper search: its path, what a call costs and how its reply is read.
type endpoint struct {
	path    string
	credits int
	decode  func([]byte) ([]assistant.Hit, error)
}

var endpoints = map[assistant.LookupKind]endpoint{
	assistant.LookupMaps:   {"/maps", 3, decodeVenues},
	assistant.LookupPlaces: {"/places", 1, decodeVenues},
	assistant.LookupSearch: {"/search", 1, decodeWeb},
}
