package chatpbmap

import (
	"github.com/magomedcoder/gen/api/pb/chatpb"
	"github.com/magomedcoder/gen/pkg/chatstream"
)

func StreamChunkKindToPB(kind chatstream.StreamChunkKind) chatpb.StreamChunkKind {
	switch kind {
	case chatstream.StreamChunkKindToolStatus:
		return chatpb.StreamChunkKind_STREAM_CHUNK_KIND_TOOL_STATUS
	case chatstream.StreamChunkKindNotice:
		return chatpb.StreamChunkKind_STREAM_CHUNK_KIND_NOTICE
	case chatstream.StreamChunkKindReasoning:
		return chatpb.StreamChunkKind_STREAM_CHUNK_KIND_REASONING
	case chatstream.StreamChunkKindRAGMeta:
		return chatpb.StreamChunkKind_STREAM_CHUNK_KIND_RAG_META
	default:
		return chatpb.StreamChunkKind_STREAM_CHUNK_KIND_TEXT
	}
}

func StreamChunkRole(kind chatstream.StreamChunkKind) string {
	if kind == chatstream.StreamChunkKindNotice || kind == chatstream.StreamChunkKindRAGMeta {
		return "system"
	}
	return "assistant"
}

func RAGSourcesPayloadToPB(p *chatstream.RAGSourcesPayload) *chatpb.RagSourcesPayload {
	if p == nil {
		return nil
	}

	out := &chatpb.RagSourcesPayload{
		Mode:                p.Mode,
		FileId:              p.FileID,
		TopK:                p.TopK,
		NeighborWindow:      p.NeighborWindow,
		DeepRagMapCalls:     p.DeepRAGMapCalls,
		DroppedByBudget:     p.DroppedByBudget,
		FullDocumentExcerpt: p.FullDocumentExcerpt,
	}

	for _, c := range p.Chunks {
		out.Chunks = append(out.Chunks, &chatpb.RagChunkPreview{
			ChunkIndex:   c.ChunkIndex,
			Score:        c.Score,
			IsNeighbor:   c.IsNeighbor,
			HeadingPath:  c.HeadingPath,
			PdfPageStart: c.PdfPageStart,
			PdfPageEnd:   c.PdfPageEnd,
			Excerpt:      c.Excerpt,
		})
	}

	return out
}
