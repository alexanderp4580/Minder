package port

import "github.com/alexanderp4580/Minder/server/domain"

type Captures interface {
	RedactCapture(string) error
	QueueCaptureNotifications(string, int) error
	Capture(string) (domain.Capture, error)
	PutCapture(domain.Capture) error
	Receipt(string) (domain.CaptureReceipt, error)
	PutReceipt(domain.CaptureReceipt) error
	LinkSource(string, string) error
	SourcesForItem(string) ([]string, error)
}
