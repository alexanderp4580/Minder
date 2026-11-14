package domain

// ItemColors are the six approved label colours; red is reserved for actions and urgency.
var ItemColors = []string{"blue", "teal", "green", "amber", "lilac", "slate"}

func ValidColor(color string) bool {
	for _, c := range ItemColors {
		if c == color {
			return true
		}
	}
	return false
}
func validatePresentation(i *Item) error {
	if i.Color == "" {
		i.Color = "slate"
	}
	if !ValidColor(i.Color) {
		return &Invalid{Field: "color", Message: "Choose Blue, Teal, Green, Amber, Lilac or Slate."}
	}
	if i.LocationKind != "" && i.LocationKind != "address" && i.LocationKind != "label" {
		return &Invalid{Field: "location_kind", Message: "Choose address or label for the location."}
	}
	if i.DateEnd != "" && i.DateEnd != "inclusive" && i.DateEnd != "checkout" {
		return &Invalid{Field: "date_end", Message: "Choose inclusive or checkout for a date-only end."}
	}
	return nil
}
