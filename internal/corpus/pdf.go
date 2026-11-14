package corpus

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
)

var (
	pdfStreamPattern = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	pdfTextTjPattern = regexp.MustCompile(`\((.*?)\)\s*Tj`)
	pdfTextTJPattern = regexp.MustCompile(`\[(.*?)\]\s*TJ`)
	pdfStringLiteral = regexp.MustCompile(`\((.*?)\)`)
)

// parsePDFFile extracts text streams from a PDF file using standard library pure Go parsers.
func parsePDFFile(docPath string, data []byte, cfg *config.CorpusConfig) ([]*Chunk, error) {
	var extractedText strings.Builder

	// Find all streams
	matches := pdfStreamPattern.FindAllSubmatch(data, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		rawStream := m[1]

		// Try decompressing with Flate/Zlib
		var textBytes []byte
		zr, err := zlib.NewReader(bytes.NewReader(rawStream))
		if err == nil {
			decompressed, readErr := io.ReadAll(zr)
			zr.Close()
			if readErr == nil && len(decompressed) > 0 {
				textBytes = decompressed
			}
		}

		if len(textBytes) == 0 {
			textBytes = rawStream
		}

		// Extract text from text operators (Tj and TJ)
		for _, tjMatch := range pdfTextTjPattern.FindAllSubmatch(textBytes, -1) {
			if len(tjMatch) >= 2 {
				extractedText.WriteString(cleanPDFString(string(tjMatch[1])))
				extractedText.WriteString(" ")
			}
		}

		for _, tjMatch := range pdfTextTJPattern.FindAllSubmatch(textBytes, -1) {
			if len(tjMatch) >= 2 {
				for _, lit := range pdfStringLiteral.FindAllSubmatch(tjMatch[1], -1) {
					if len(lit) >= 2 {
						extractedText.WriteString(cleanPDFString(string(lit[1])))
					}
				}
				extractedText.WriteString(" ")
			}
		}
	}

	content := strings.TrimSpace(extractedText.String())
	if len(content) < cfg.MinChunkLength {
		// Fallback: extract any ASCII string sequences > 4 chars
		var asciiBuilder strings.Builder
		var currentWord strings.Builder
		for _, b := range data {
			if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
				currentWord.WriteByte(b)
			} else {
				if currentWord.Len() >= 4 {
					asciiBuilder.WriteString(currentWord.String())
					asciiBuilder.WriteString(" ")
				}
				currentWord.Reset()
			}
		}
		if currentWord.Len() >= 4 {
			asciiBuilder.WriteString(currentWord.String())
		}
		content = strings.TrimSpace(asciiBuilder.String())
	}

	if len(content) < cfg.MinChunkLength {
		return nil, fmt.Errorf("could not extract sufficient text from PDF %q", docPath)
	}

	// Chunk the extracted text using sliding windows
	windows := splitWithOverlap(content, cfg.ChunkSize, cfg.ChunkOverlap)
	var chunks []*Chunk
	for idx, win := range windows {
		if len(win) >= cfg.MinChunkLength {
			heading := fmt.Sprintf("Page %d Section", (idx/2)+1)
			chunks = append(chunks, buildChunk(docPath, idx, heading, win))
		}
	}

	return chunks, nil
}

func cleanPDFString(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\(`, "(")
	s = strings.ReplaceAll(s, `\)`, ")")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
