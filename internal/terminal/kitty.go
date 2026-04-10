package terminal

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// RenderImageKitty returns a string containing Kitty graphics protocol escape
// sequences that display the image at the given path.
// maxWidth and maxHeight are in terminal cells (characters).
func RenderImageKitty(imagePath string, maxWidth, maxHeight int) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	// Kitty graphics protocol:
	// \033_Gf=100,a=T,c=<cols>,r=<rows>;base64data\033\\
	// f=100 means PNG format
	// a=T means transmit and display
	// c=columns, r=rows for sizing
	// For large payloads, chunk into 4096-byte pieces

	var b strings.Builder

	chunkSize := 4096
	chunks := splitString(encoded, chunkSize)

	for i, chunk := range chunks {
		if i == 0 {
			// First chunk: include all parameters
			if len(chunks) == 1 {
				// Single chunk
				b.WriteString(fmt.Sprintf("\033_Gf=100,a=T,c=%d,r=%d;%s\033\\", maxWidth, maxHeight, chunk))
			} else {
				// Multi-chunk: m=1 means more data follows
				b.WriteString(fmt.Sprintf("\033_Gf=100,a=T,c=%d,r=%d,m=1;%s\033\\", maxWidth, maxHeight, chunk))
			}
		} else if i == len(chunks)-1 {
			// Last chunk: m=0
			b.WriteString(fmt.Sprintf("\033_Gm=0;%s\033\\", chunk))
		} else {
			// Middle chunk: m=1
			b.WriteString(fmt.Sprintf("\033_Gm=1;%s\033\\", chunk))
		}
	}

	return b.String(), nil
}

func splitString(s string, size int) []string {
	var chunks []string
	for len(s) > 0 {
		end := size
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[:end])
		s = s[end:]
	}
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	return chunks
}
