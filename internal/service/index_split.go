package service

import (
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
)

const (
	serverMaxChunkBytes = 8192
	serverChunkOverlap  = 3
	serverMaxSegments   = 32
)

// expandOversizedChunks разделяет устаревшие большие клиентские блоки данных на стороне сервера
func expandOversizedChunks(chunks []domain.IndexChunk) []domain.IndexChunk {
	if len(chunks) == 0 {
		return chunks
	}

	out := make([]domain.IndexChunk, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk.Content) <= serverMaxChunkBytes {
			out = append(out, chunk)
			continue
		}

		segments := splitChunkLines(chunk.Content, serverMaxChunkBytes, serverChunkOverlap)
		if len(segments) == 0 {
			out = append(out, chunk)
			continue
		}

		for i, seg := range segments {
			nc := chunk
			if i == 0 {
				nc.ID = chunk.ID
			} else {
				nc.ID = fmt.Sprintf("%s-%d", strings.TrimSpace(chunk.ID), i)
			}

			nc.Content = seg

			if i > 0 {
				nc.Imports = nil
			}

			out = append(out, nc)
		}
	}
	return out
}

func splitChunkLines(content string, maxBytes, overlap int) []string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		return nil
	}

	var segments []string
	start := 0
	for start < len(lines) && len(segments) < serverMaxSegments {
		end := start + 1
		for end <= len(lines) {
			segment := strings.Join(lines[start:end], "\n")
			if len(segment) > maxBytes {
				break
			}
			end++
		}

		takeEnd := end - 1
		if takeEnd <= start {
			line := lines[start]
			if len(line) > maxBytes {
				line = line[:maxBytes] + "..."
			}

			segments = append(segments, line)
			start++
			continue
		}

		segments = append(segments, strings.Join(lines[start:takeEnd], "\n"))
		if takeEnd >= len(lines) {
			break
		}

		start = takeEnd - overlap
		if start < 0 {
			start = 0
		}
	}

	return segments
}
