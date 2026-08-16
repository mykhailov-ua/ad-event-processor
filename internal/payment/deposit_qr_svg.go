package payment

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func cryptoDepositQRSVG(data string) string {
	if strings.TrimSpace(data) == "" {
		return ""
	}
	qr, err := qrcode.New(data, qrcode.Low)
	if err != nil {
		return ""
	}
	bitmap := qr.Bitmap()
	size := len(bitmap)
	if size == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 `)
	b.WriteString(fmt.Sprintf("%d %d", size, size))
	b.WriteString(`" shape-rendering="crispEdges"><rect width="100%" height="100%" fill="#fff"/>`)
	for y, row := range bitmap {
		for x, dark := range row {
			if dark {
				b.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1" fill="#000"/>`, x, y))
			}
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}
