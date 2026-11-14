package corpus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
)

// IngestCorpus reads all documents specified in the corpus configuration,
// segments them into structured chunks, extracts entities, and prepares search tokens.
func IngestCorpus(cfg *config.CorpusConfig) ([]*Chunk, error) {
	var allChunks []*Chunk
	seenChunkIDs := make(map[string]bool)

	for _, pattern := range cfg.Paths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid file pattern %q: %w", pattern, err)
		}

		if len(matches) == 0 {
			// Check if pattern is a directory
			fi, statErr := os.Stat(pattern)
			if statErr == nil && fi.IsDir() {
				err = filepath.Walk(pattern, func(p string, info os.FileInfo, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					if !info.IsDir() && isSupportedExtension(p) {
						chunks, err := processFile(p, cfg)
						if err != nil {
							return err
						}
						for _, ch := range chunks {
							if !seenChunkIDs[ch.ID] {
								seenChunkIDs[ch.ID] = true
								allChunks = append(allChunks, ch)
							}
						}
					}
					return nil
				})
				if err != nil {
					return nil, fmt.Errorf("error traversing directory %q: %w", pattern, err)
				}
				continue
			}
			return nil, fmt.Errorf("no corpus files found matching path %q", pattern)
		}

		for _, match := range matches {
			fi, err := os.Stat(match)
			if err != nil {
				continue
			}
			if fi.IsDir() {
				err = filepath.Walk(match, func(p string, info os.FileInfo, walkErr error) error {
					if walkErr != nil || info.IsDir() || !isSupportedExtension(p) {
						return walkErr
					}
					chunks, err := processFile(p, cfg)
					if err != nil {
						return err
					}
					for _, ch := range chunks {
						if !seenChunkIDs[ch.ID] {
							seenChunkIDs[ch.ID] = true
							allChunks = append(allChunks, ch)
						}
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
			} else if isSupportedExtension(match) {
				chunks, err := processFile(match, cfg)
				if err != nil {
					return nil, err
				}
				for _, ch := range chunks {
					if !seenChunkIDs[ch.ID] {
						seenChunkIDs[ch.ID] = true
						allChunks = append(allChunks, ch)
					}
				}
			}
		}
	}

	if len(allChunks) == 0 {
		return nil, fmt.Errorf("ingested 0 valid chunks from configured paths %v", cfg.Paths)
	}

	return allChunks, nil
}

func isSupportedExtension(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return ext == ".md" || ext == ".markdown" || ext == ".txt" || ext == ".json" || ext == ".jsonl"
}

func processFile(path string, cfg *config.CorpusConfig) ([]*Chunk, error) {
	ext := strings.ToLower(filepath.Ext(path))
	relPath, err := filepath.Rel(".", path)
	if err != nil {
		relPath = filepath.Clean(path)
	}
	relPath = filepath.ToSlash(relPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	switch ext {
	case ".json":
		return parseJSONFile(relPath, data, cfg)
	case ".jsonl":
		return parseJSONLFile(relPath, data, cfg)
	case ".md", ".markdown":
		return parseMarkdownFile(relPath, string(data), cfg)
	default:
		return parseTextFile(relPath, string(data), cfg)
	}
}

func parseMarkdownFile(docPath string, content string, cfg *config.CorpusConfig) ([]*Chunk, error) {
	var chunks []*Chunk
	lines := strings.Split(content, "\n")

	type section struct {
		heading string
		lines   []string
	}

	var sections []section
	currentHeading := "Overview"
	var currentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := headingPattern.FindStringSubmatch(trimmed); len(m) == 3 {
			if len(currentLines) > 0 {
				sections = append(sections, section{
					heading: currentHeading,
					lines:   currentLines,
				})
				currentLines = nil
			}
			currentHeading = m[2]
		} else {
			currentLines = append(currentLines, line)
		}
	}
	if len(currentLines) > 0 {
		sections = append(sections, section{
			heading: currentHeading,
			lines:   currentLines,
		})
	}

	chunkIdx := 0
	for _, sec := range sections {
		secText := strings.TrimSpace(strings.Join(sec.lines, "\n"))
		if len(secText) < cfg.MinChunkLength {
			continue
		}

		if utf8.RuneCountInString(secText) <= cfg.ChunkSize {
			ch := buildChunk(docPath, chunkIdx, sec.heading, secText)
			chunks = append(chunks, ch)
			chunkIdx++
		} else {
			// Window split with overlap
			windows := splitWithOverlap(secText, cfg.ChunkSize, cfg.ChunkOverlap)
			for _, winText := range windows {
				if len(winText) >= cfg.MinChunkLength {
					ch := buildChunk(docPath, chunkIdx, sec.heading, winText)
					chunks = append(chunks, ch)
					chunkIdx++
				}
			}
		}
	}

	// Fallback if no headings existed
	if len(chunks) == 0 && len(strings.TrimSpace(content)) >= cfg.MinChunkLength {
		windows := splitWithOverlap(strings.TrimSpace(content), cfg.ChunkSize, cfg.ChunkOverlap)
		for idx, win := range windows {
			chunks = append(chunks, buildChunk(docPath, idx, "Document", win))
		}
	}

	return chunks, nil
}

func parseTextFile(docPath string, content string, cfg *config.CorpusConfig) ([]*Chunk, error) {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < cfg.MinChunkLength {
		return nil, nil
	}

	var chunks []*Chunk
	windows := splitWithOverlap(trimmed, cfg.ChunkSize, cfg.ChunkOverlap)
	for idx, win := range windows {
		if len(win) >= cfg.MinChunkLength {
			chunks = append(chunks, buildChunk(docPath, idx, "Text", win))
		}
	}
	return chunks, nil
}

type jsonChunkInput struct {
	ID       string            `json:"id"`
	Text     string            `json:"text"`
	Content  string            `json:"content"`
	Heading  string            `json:"heading"`
	Metadata map[string]string `json:"metadata"`
}

func parseJSONFile(docPath string, data []byte, cfg *config.CorpusConfig) ([]*Chunk, error) {
	var items []jsonChunkInput
	if err := json.Unmarshal(data, &items); err != nil {
		var single jsonChunkInput
		if err2 := json.Unmarshal(data, &single); err2 == nil {
			items = []jsonChunkInput{single}
		} else {
			return nil, fmt.Errorf("invalid json in %q: %w", docPath, err)
		}
	}

	var chunks []*Chunk
	for idx, item := range items {
		body := item.Text
		if body == "" {
			body = item.Content
		}
		if len(strings.TrimSpace(body)) < cfg.MinChunkLength {
			continue
		}

		ch := buildChunk(docPath, idx, item.Heading, body)
		if item.ID != "" {
			ch.ID = item.ID
		}
		if item.Metadata != nil {
			ch.Metadata = item.Metadata
		}
		chunks = append(chunks, ch)
	}

	return chunks, nil
}

func parseJSONLFile(docPath string, data []byte, cfg *config.CorpusConfig) ([]*Chunk, error) {
	var chunks []*Chunk
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	idx := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var item jsonChunkInput
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}

		body := item.Text
		if body == "" {
			body = item.Content
		}
		if len(strings.TrimSpace(body)) < cfg.MinChunkLength {
			continue
		}

		ch := buildChunk(docPath, idx, item.Heading, body)
		if item.ID != "" {
			ch.ID = item.ID
		}
		if item.Metadata != nil {
			ch.Metadata = item.Metadata
		}
		chunks = append(chunks, ch)
		idx++
	}

	return chunks, nil
}

func buildChunk(docPath string, index int, heading string, text string) *Chunk {
	text = strings.TrimSpace(text)
	tokens := Tokenize(text)
	entities := ExtractEntities(text)

	id := ComputeChunkID(docPath, index, text)

	return &Chunk{
		ID:        id,
		DocID:     docPath,
		DocPath:   docPath,
		Heading:   heading,
		Text:      text,
		Tokens:    tokens,
		Entities:  entities,
		Metadata:  make(map[string]string),
		Index:     index,
		CharCount: len(text),
		WordCount: len(tokens),
	}
}

func splitWithOverlap(text string, chunkSize int, overlap int) []string {
	if chunkSize <= overlap {
		chunkSize = overlap + 100
	}

	var segments []string
	runes := []rune(text)
	total := len(runes)

	start := 0
	for start < total {
		end := start + chunkSize
		if end > total {
			end = total
		}

		// Try to break at a natural newline or sentence boundary near the end
		if end < total {
			bestBreak := -1
			for i := end; i > start+chunkSize/2; i-- {
				if runes[i-1] == '\n' || runes[i-1] == '.' {
					bestBreak = i
					break
				}
			}
			if bestBreak != -1 {
				end = bestBreak
			}
		}

		segment := strings.TrimSpace(string(runes[start:end]))
		if len(segment) > 0 {
			segments = append(segments, segment)
		}

		if end >= total {
			break
		}

		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return segments
}
