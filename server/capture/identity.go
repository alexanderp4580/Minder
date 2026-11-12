package capture

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/alexanderp4580/Minder/server/domain"
)

func stableCandidateID(c domain.CaptureCandidate, index int) string {
	// Evidence identifies the source item across a date correction; distinct segments retain separate indexes.
	text := fmt.Sprint(index)
	for _, e := range c.Evidence {
		text += "\n" + e.Part + "\n" + e.Excerpt
	}
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:12])
}
