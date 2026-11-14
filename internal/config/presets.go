package config

import (
	"fmt"
	"strings"
)

// Preset represents a pre-configured integration profile for popular RAG frameworks.
type Preset struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	RequestTemplate string          `json:"request_template"`
	ResponseMapping ResponseMapping `json:"response_mapping"`
}

// AvailablePresets returns standard integration templates for common RAG stacks.
var AvailablePresets = map[string]Preset{
	"generic": {
		Name:            "generic",
		Description:     "Standard generic RAG HTTP schema",
		RequestTemplate: `{"query": "{query}", "top_k": 5}`,
		ResponseMapping: ResponseMapping{
			AnswerPath:    "answer",
			ChunksPath:    "chunks",
			ChunkIDPath:   "id",
			ChunkTextPath: "text",
		},
	},
	"langchain": {
		Name:            "langchain",
		Description:     "LangChain / LangServe chain endpoint schema",
		RequestTemplate: `{"input": {"question": "{query}"}}`,
		ResponseMapping: ResponseMapping{
			AnswerPath:    "output.answer",
			ChunksPath:    "output.context",
			ChunkIDPath:   "metadata.id",
			ChunkTextPath: "page_content",
		},
	},
	"llamaindex": {
		Name:            "llamaindex",
		Description:     "LlamaIndex query engine HTTP schema",
		RequestTemplate: `{"query": "{query}"}`,
		ResponseMapping: ResponseMapping{
			AnswerPath:    "response",
			ChunksPath:    "source_nodes",
			ChunkIDPath:   "node.id_",
			ChunkTextPath: "node.text",
		},
	},
	"openai": {
		Name:            "openai",
		Description:     "OpenAI Chat Completions / Assistants schema",
		RequestTemplate: `{"messages": [{"role": "user", "content": "{query}"}]}`,
		ResponseMapping: ResponseMapping{
			AnswerPath:    "choices.0.message.content",
			ChunksPath:    "choices.0.message.context.chunks",
			ChunkIDPath:   "id",
			ChunkTextPath: "text",
		},
	},
	"dspy": {
		Name:            "dspy",
		Description:     "DSPy compiled RAG module HTTP schema",
		RequestTemplate: `{"question": "{query}"}`,
		ResponseMapping: ResponseMapping{
			AnswerPath:    "answer",
			ChunksPath:    "context",
			ChunkIDPath:   "id",
			ChunkTextPath: "text",
		},
	},
	"flowise": {
		Name:            "flowise",
		Description:     "Flowise AI prediction endpoint schema",
		RequestTemplate: `{"question": "{query}"}`,
		ResponseMapping: ResponseMapping{
			AnswerPath:    "text",
			ChunksPath:    "sourceDocuments",
			ChunkIDPath:   "metadata.id",
			ChunkTextPath: "pageContent",
		},
	},
}

// ApplyPreset applies a named integration preset to the TargetConfig.
func (c *TargetConfig) ApplyPreset(presetName string) error {
	name := strings.ToLower(strings.TrimSpace(presetName))
	if name == "" {
		return nil
	}

	preset, exists := AvailablePresets[name]
	if !exists {
		var valid []string
		for k := range AvailablePresets {
			valid = append(valid, k)
		}
		return fmt.Errorf("unknown preset %q, available presets: %s", presetName, strings.Join(valid, ", "))
	}

	c.RequestTemplate = preset.RequestTemplate
	c.ResponseMapping = preset.ResponseMapping
	return nil
}
