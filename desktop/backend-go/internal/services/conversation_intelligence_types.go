package services

import "time"

// ConversationAnalysis represents a detailed analysis of a conversation
type ConversationAnalysis struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	UserID         string                 `json:"user_id"`
	Title          string                 `json:"title"`
	Summary        string                 `json:"summary"`
	KeyPoints      []string               `json:"key_points"`
	Topics         []ConversationTopic    `json:"topics"`
	Sentiment      SentimentAnalysis      `json:"sentiment"`
	Entities       []ConversationEntity   `json:"entities"`
	ActionItems    []ActionItem           `json:"action_items"`
	Questions      []Question             `json:"questions"`
	Decisions      []ConversationDecision `json:"decisions"`
	CodeMentions   []CodeMention          `json:"code_mentions"`
	MessageCount   int                    `json:"message_count"`
	TokenCount     int                    `json:"token_count"`
	Duration       string                 `json:"duration"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ConversationTopic represents a topic discussed in the conversation
type ConversationTopic struct {
	Name         string   `json:"name"`
	Confidence   float64  `json:"confidence"`
	Keywords     []string `json:"keywords"`
	FirstMention int      `json:"first_mention"` // Message index
	Frequency    int      `json:"frequency"`
}

// SentimentAnalysis represents sentiment analysis results
type SentimentAnalysis struct {
	Overall     string               `json:"overall"` // positive, negative, neutral, mixed
	Score       float64              `json:"score"`   // -1 to 1
	Progression []SentimentPoint     `json:"progression"`
	Highlights  []SentimentHighlight `json:"highlights"`
}

// SentimentPoint represents sentiment at a point in the conversation
type SentimentPoint struct {
	MessageIndex int     `json:"message_index"`
	Sentiment    string  `json:"sentiment"`
	Score        float64 `json:"score"`
}

// SentimentHighlight represents a significant sentiment moment
type SentimentHighlight struct {
	MessageIndex int    `json:"message_index"`
	Text         string `json:"text"`
	Sentiment    string `json:"sentiment"`
	Reason       string `json:"reason"`
}

// ConversationEntity represents an entity mentioned in the conversation
type ConversationEntity struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"` // person, organization, technology, file, concept
	Mentions int      `json:"mentions"`
	Context  []string `json:"context"`
	Related  []string `json:"related"`
}

// ActionItem represents a task or action mentioned
type ActionItem struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Priority     string   `json:"priority"` // high, medium, low
	Status       string   `json:"status"`   // pending, completed, cancelled
	AssignedTo   string   `json:"assigned_to,omitempty"`
	DueDate      string   `json:"due_date,omitempty"`
	MessageIndex int      `json:"message_index"`
	Tags         []string `json:"tags"`
}

// Question represents a question in the conversation
type Question struct {
	Text         string `json:"text"`
	AskedBy      string `json:"asked_by"` // user, assistant
	MessageIndex int    `json:"message_index"`
	Answered     bool   `json:"answered"`
	Answer       string `json:"answer,omitempty"`
}

// ConversationDecision represents a decision made in the conversation
type ConversationDecision struct {
	Description  string   `json:"description"`
	Context      string   `json:"context"`
	Alternatives []string `json:"alternatives,omitempty"`
	Rationale    string   `json:"rationale,omitempty"`
	MessageIndex int      `json:"message_index"`
}

// CodeMention represents code discussed in the conversation
type CodeMention struct {
	FilePath     string `json:"file_path,omitempty"`
	Language     string `json:"language,omitempty"`
	Snippet      string `json:"snippet"`
	Context      string `json:"context"`
	MessageIndex int    `json:"message_index"`
}

// Message represents a conversation message for analysis
type Message struct {
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// wordCount is used for keyword frequency analysis
type wordCount struct {
	word  string
	count int
}
