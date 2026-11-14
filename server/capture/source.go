package capture

import (
	"context"
	"errors"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// FindSources lists retained sources that match a word of their subject, sender or text.
func (s *Service) FindSources(_ context.Context, query string) ([]domain.SourceRef, error) {
	return s.repository.FindSources(query)
}

// ReadSource loads a retained source's parts. It checks deletion on every read and changes
// nothing: no extraction, model call, push or item change.
func (s *Service) ReadSource(ctx context.Context, id string) (domain.SourceDocument, error) {
	var ref domain.SourceRef
	var entry domain.Entry
	err := s.repository.Transact(func(tx port.Transaction) (err error) {
		if ref, err = tx.Source(id); err == nil {
			return nil
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		// Owner text that was never reviewed has no record of its own; the chat entry is the source.
		if entry, err = tx.Entry(id); err != nil || entry.Kind != "user" {
			return domain.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return domain.SourceDocument{}, err
	}
	if ref.ID == "" {
		return chatDocument(domain.SourceRef{ID: id, Kind: domain.SourceChat, EntryID: id, CreatedAt: entry.CreatedAt}, entry)
	}
	if ref.Deleted {
		return domain.SourceDocument{}, domain.ErrSourceUnavailable
	}
	if ref.Kind == domain.SourceChat {
		err = s.repository.Transact(func(tx port.Transaction) (err error) { entry, err = tx.Entry(ref.EntryID); return })
		if err != nil {
			return domain.SourceDocument{}, domain.ErrSourceUnavailable
		}
		return chatDocument(ref, entry)
	}
	return s.emailDocument(ctx, ref)
}

// chatDocument presents an owner message as one text part, with no email structure invented.
func chatDocument(ref domain.SourceRef, entry domain.Entry) (domain.SourceDocument, error) {
	ref.Label = domain.ChatLabel(entry.Text)
	part := domain.CapturePart{ID: "text", Name: "Chat message", MediaType: "text/plain", Text: entry.Text}
	return domain.SourceDocument{Source: ref, Document: domain.CaptureDocument{
		CaptureID: ref.CaptureID, Subject: ref.Label, OriginalDate: entry.CreatedAt,
		Parts: []domain.CapturePart{part}, Attachments: []domain.CaptureAttachment{},
		Coverage: []domain.CaptureCoverage{{Part: "text", Status: "parsed"}},
	}}, nil
}

func (s *Service) emailDocument(ctx context.Context, ref domain.SourceRef) (domain.SourceDocument, error) {
	c, err := s.repository.Capture(ref.CaptureID)
	if errors.Is(err, domain.ErrNotFound) || c.Deleted || c.FilesDeleted {
		return domain.SourceDocument{}, domain.ErrSourceUnavailable
	}
	if err != nil {
		return domain.SourceDocument{}, err
	}
	if s.files == nil || s.Parser == nil {
		return domain.SourceDocument{}, domain.ErrSourceUnreadable
	}
	f, err := s.files.Open(c.ID, "original")
	if err != nil {
		return domain.SourceDocument{}, domain.ErrSourceUnreadable
	}
	defer f.Close()
	document, err := s.Parser.Parse(ctx, f)
	if err != nil {
		if ctx.Err() != nil {
			return domain.SourceDocument{}, ctx.Err()
		}
		return domain.SourceDocument{}, domain.ErrSourceUnreadable
	}
	document.CaptureID = c.ID
	return domain.SourceDocument{Source: ref, Document: document}, nil
}
