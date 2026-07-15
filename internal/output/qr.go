package output

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// QR renders s as a compact Unicode QR code suitable for a terminal, using
// half-block characters so it fits in roughly half the vertical space.
func QR(s string) (string, error) {
	q, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("build qr code: %w", err)
	}
	return q.ToSmallString(false), nil
}
