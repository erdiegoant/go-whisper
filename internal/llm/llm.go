package llm

// Processor cleans up a transcript using a language model.
// Both the Claude and Ollama clients implement this interface.
type Processor interface {
	Process(systemPrompt, text string) (string, error)
}
