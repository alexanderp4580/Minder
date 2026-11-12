// Package mailparse converts bounded MIME parts into text and image source evidence.
package mailparse

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strconv"
	"strings"

	"github.com/alexanderp4580/Minder/server/domain"
	"golang.org/x/net/html/charset"
)

type Parser struct{}
type traversal struct {
	document                  domain.CaptureDocument
	parts, attachments, pages int
}

func (p *Parser) Parse(ctx context.Context, raw io.Reader) (domain.CaptureDocument, error) {
	data, e := io.ReadAll(io.LimitReader(raw, domain.MaxEmailBytes+1))
	if e != nil {
		return domain.CaptureDocument{}, e
	}
	if len(data) > domain.MaxEmailBytes {
		return domain.CaptureDocument{}, errors.New("Email exceeds 20 MiB")
	}
	message, e := mail.ReadMessage(bytes.NewReader(data))
	if e != nil {
		return domain.CaptureDocument{}, errors.New("Email headers are unreadable")
	}
	subject := message.Header.Get("Subject")
	if decoded, e := new(mime.WordDecoder).DecodeHeader(subject); e == nil {
		subject = decoded
	}
	walk := &traversal{document: domain.CaptureDocument{Subject: subject, Parts: []domain.CapturePart{}, Attachments: []domain.CaptureAttachment{}, Coverage: []domain.CaptureCoverage{}}}

	e = walk.part(ctx, message.Header, message.Body, "1", 0)
	return walk.document, e
}
func (w *traversal) part(ctx context.Context, header mail.Header, body io.Reader, id string, depth int) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	w.parts++
	if w.parts > 50 || depth > 12 {
		return errors.New("Email has too many MIME parts")
	}
	media, params, e := mime.ParseMediaType(header.Get("Content-Type"))
	if header.Get("Content-Type") == "" {
		media = "text/plain"
		params = map[string]string{}
	} else if e != nil {
		return errors.New("Email has an invalid MIME type")
	}
	var reader io.Reader = body
	switch strings.ToLower(header.Get("Content-Transfer-Encoding")) {
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, reader)
	case "quoted-printable":
		reader = quotedprintable.NewReader(reader)
	}
	if strings.HasPrefix(media, "multipart/") {
		if params["boundary"] == "" {
			return errors.New("Email MIME boundary is missing")
		}
		mr := multipart.NewReader(reader, params["boundary"])
		for n := 1; ; n++ {
			part, e := mr.NextRawPart()
			if errors.Is(e, io.EOF) {
				break
			}
			if e != nil {
				return errors.New("Email MIME parts are unreadable")
			}
			e = w.part(ctx, mail.Header(part.Header), part, id+"."+strconv.Itoa(n), depth+1)
			part.Close()
			if e != nil {
				return e
			}
		}
		return nil
	}
	if strings.HasPrefix(media, "text/") && params["charset"] != "" {
		decoded, err := charset.NewReaderLabel(params["charset"], reader)
		if err != nil {
			return errors.New("An email text charset is unsupported")
		}
		reader = decoded
	}
	data, e := io.ReadAll(io.LimitReader(reader, (10<<20)+1))
	if e != nil {
		return errors.New("An email part could not be decoded")
	}
	if len(data) > 10<<20 {
		return errors.New("An email part exceeds 10 MiB")
	}
	disposition, dp, _ := mime.ParseMediaType(header.Get("Content-Disposition"))
	name := dp["filename"]
	if name == "" {
		name = params["name"]
	}
	attachment := disposition == "attachment" || name != "" || (!strings.HasPrefix(media, "text/") && media != "message/rfc822") || media == "text/calendar"
	if attachment {
		w.attachments++
		if w.attachments > 25 {
			return errors.New("Email has more than 25 attachments")
		}
		if name == "" {
			name = "attachment-" + id
		}
		if decoded, e := new(mime.WordDecoder).DecodeHeader(name); e == nil {
			name = decoded
		}
	}
	part := domain.CapturePart{ID: strings.ReplaceAll(id, ".", "_"), Name: name, MediaType: media, Data: data, Attachment: attachment}
	coverage := domain.CaptureCoverage{Part: part.ID, Name: name, Status: "parsed"}
	switch media {
	case "text/plain":
		part.Text = string(data)
	case "text/html":
		part.Text = htmlText(data)
	case "text/calendar", "application/ics":
		text, e := calendarText(data)
		if e != nil {
			coverage.Status = "unreadable"
			coverage.Detail = "Calendar attachment is unreadable."
		} else {
			part.Text = text
		}
	case "image/jpeg", "image/png":
		w.pages++
		if w.pages > 50 {
			coverage.Status = "deferred"
			coverage.Detail = "The automatic page/image limit is reached."
		}
	case "application/pdf":
		pages, status := pdfParts(ctx, part, &w.pages)
		w.document.Parts = append(w.document.Parts, pages...)
		if status != "" {
			coverage.Status = "unreadable"
			coverage.Detail = status
		}
	case "message/rfc822":
		nested, e := mail.ReadMessage(bytes.NewReader(data))
		if e != nil {
			coverage.Status = "unreadable"
			coverage.Detail = "Attached email is unreadable."
		} else {
			if date, e := nested.Header.Date(); e == nil {
				w.document.OriginalDate = date.UTC().Format("2006-01-02T15:04:05Z")
			}
			if e = w.part(ctx, nested.Header, nested.Body, id+".1", depth+1); e != nil {
				return e
			}
		}
	default:
		coverage.Status = "unsupported"
		coverage.Detail = "This file is retained but cannot be parsed."
	}
	if len(part.Text) > 200000 {
		part.Text = ""
		coverage.Status = "deferred"
		coverage.Detail = "This text exceeds the automatic extraction limit."
	}
	if attachment {
		digest := sha256.Sum256(data)
		w.document.Attachments = append(w.document.Attachments, domain.CaptureAttachment{ID: part.ID, Name: name, MediaType: media, Size: int64(len(data)), SHA256: hex.EncodeToString(digest[:])})
	}
	if coverage.Status != "parsed" {
		w.document.Questions = append(w.document.Questions, fmt.Sprintf("%s Provide readable content or acknowledge this extraction warning.", coverage.Detail))
	}
	w.document.Parts = append(w.document.Parts, part)
	w.document.Coverage = append(w.document.Coverage, coverage)
	return nil
}
