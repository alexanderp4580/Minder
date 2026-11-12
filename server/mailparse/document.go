package mailparse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/alexanderp4580/Minder/server/domain"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type boundedOutput struct {
	bytes.Buffer
	limit int
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, errors.New("document output limit")
	}
	return b.Buffer.Write(p)
}
func documentCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out := &boundedOutput{limit: 1 << 20}
	cmd.Stdout = out
	if e := cmd.Run(); e != nil {
		return nil, errors.New("document conversion failed")
	}
	return out.Bytes(), nil
}

var pageCount = regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)

func pdfParts(ctx context.Context, part domain.CapturePart, used *int) ([]domain.CapturePart, string) {
	dir, e := os.MkdirTemp("", "minder-document-")
	if e != nil {
		return nil, "PDF conversion is unavailable."
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "source.pdf")
	if e = os.WriteFile(path, part.Data, 0600); e != nil {
		return nil, "PDF could not be read."
	}
	info, e := documentCommand(ctx, "pdfinfo", path)
	if e != nil {
		return nil, "PDF is encrypted, unreadable or document tools are unavailable."
	}
	match := pageCount.FindSubmatch(info)
	if len(match) != 2 {
		return nil, "PDF page count is unavailable."
	}
	count, _ := strconv.Atoi(string(match[1]))
	out := []domain.CapturePart{}
	status := ""
	for page := 1; page <= count; page++ {
		if *used >= 50 {
			status = "The automatic 50-page/image limit is reached."
			break
		}
		*used++
		number := strconv.Itoa(page)
		id := fmt.Sprintf("%s_page_%d", part.ID, page)
		text, e := documentCommand(ctx, "pdftotext", "-f", number, "-l", number, "-enc", "UTF-8", path, "-")
		if e == nil && strings.TrimSpace(string(text)) != "" {
			out = append(out, domain.CapturePart{ID: id, Name: part.Name, MediaType: "text/plain", Text: string(text)})
			continue
		}
		prefix := filepath.Join(dir, "page")
		if _, e = documentCommand(ctx, "pdftoppm", "-f", number, "-l", number, "-singlefile", "-scale-to", "1200", "-jpeg", path, prefix); e != nil {
			status = "Some PDF pages could not be read."
			continue
		}
		data, e := os.ReadFile(prefix + ".jpg")
		if e != nil || len(data) > 2<<20 {
			status = "Some PDF pages exceed the extraction limit."
			continue
		}
		out = append(out, domain.CapturePart{ID: id, Name: part.Name, MediaType: "image/jpeg", Data: data})
	}
	return out, status
}
