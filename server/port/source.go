package port

import "github.com/alexanderp4580/Minder/server/domain"

// Sources stores the references to retained sources.
type Sources interface {
	Source(string) (domain.SourceRef, error)
	PutSource(domain.SourceRef) error
}
