package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// ChainPromptHeader is the text prompting the agent to continue.
	ChainPromptHeader = "Based on the above context, please continue."
	// ChainPromptInstruction is the text instructing the agent on output format.
	ChainPromptInstruction = "IMPORTANT: Please start your output with ':::VIBE_OUTPUT:::' so I can extract it reliably."
)

// ChainEntry represents a single conclusion from an agent in the chain.
type ChainEntry struct {
	Agent      string    `json:"agent"`
	Timestamp  time.Time `json:"timestamp"`
	Conclusion string    `json:"conclusion"`
}

// ChainContext represents the shared context for a chain session.
type ChainContext struct {
	SessionID string       `json:"session_id"`
	CreatedAt time.Time    `json:"created_at"`
	Task      string       `json:"task"`
	Chain     []ChainEntry `json:"chain"`
	mu        sync.RWMutex
	path      string `json:"-"`
}

// NewChainContext creates a new chain context.
func NewChainContext(id, task, dir string) (*ChainContext, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	
	path := filepath.Join(dir, id+".json")
	return &ChainContext{
		SessionID: id,
		CreatedAt: time.Now(),
		Task:      task,
		Chain:     make([]ChainEntry, 0),
		path:      path,
	}, nil
}

// LoadChainContext loads a chain context from file.
func LoadChainContext(path string) (*ChainContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ctx ChainContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, err
	}
	ctx.path = path
	return &ctx, nil
}

// Save persists the chain context to file.
func (c *ChainContext) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// AppendConclusion adds a new entry to the chain and saves it.
func (c *ChainContext) AppendConclusion(agent, conclusion string) error {
	c.mu.Lock()
	c.Chain = append(c.Chain, ChainEntry{
		Agent:      agent,
		Timestamp:  time.Now(),
		Conclusion: conclusion,
	})
	c.mu.Unlock()
	
	return c.Save()
}

// GetLatestConclusion returns the most recent conclusion text.
func (c *ChainContext) GetLatestConclusion() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Chain) == 0 {
		return ""
	}
	return c.Chain[len(c.Chain)-1].Conclusion
}

// FormatContext formats the entire chain for injection into prompt.
func (c *ChainContext) FormatContext() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := "【Chain Context】\n"
	result += "Task: " + c.Task + "\n\n"
	
	for _, entry := range c.Chain {
		result += "--- Agent: " + entry.Agent + " ---\n"
		result += entry.Conclusion + "\n\n"
	}
	
	result += ChainPromptHeader + "\n"
	result += ChainPromptInstruction
	return result
}
