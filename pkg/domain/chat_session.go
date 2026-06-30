package domain

import "time"

type ChatSession struct {
	Id               int64
	UserId           int
	Title            string
	SelectedRunnerID *int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

type ChatSessionSettings struct {
	SessionID             int64
	SystemPrompt          string
	StopSequences         []string
	TimeoutSeconds        int32
	Temperature           *float32
	TopK                  *int32
	TopP                  *float32
	JSONMode              bool
	JSONSchema            string
	ToolsJSON             string
	Profile               string
	ModelReasoningEnabled bool
	WebSearchEnabled      bool
	WebSearchProvider     string
	MCPEnabled            bool
	MCPServerIDs          []int64
}

type Message struct {
	Id                int64
	SessionId         int64
	Content           string
	Role              MessageRole
	AttachmentName    string
	AttachmentMime    string
	AttachmentFileID  *int64
	AttachmentContent []byte
	ToolCallID        string
	ToolName          string
	ToolCallsJSON     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

type MessageEdit struct {
	Id              int64
	SessionId       int64
	MessageId       int64
	EditorUserId    int
	OldContent      string
	NewContent      string
	SoftDeletedFrom int64
	SoftDeletedTo   int64
	CreatedAt       time.Time
	RevertedAt      *time.Time
}

type AssistantMessageRegeneration struct {
	Id          int64
	SessionId   int64
	MessageId   int64
	RegenUserId int
	OldContent  string
	NewContent  string
	CreatedAt   time.Time
}

func NewChatSession(userId int, title string) *ChatSession {
	return &ChatSession{
		UserId:    userId,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func NewMessage(sessionId int64, content string, role MessageRole) *Message {
	return NewMessageWithAttachment(sessionId, content, role, nil)
}

func NewMessageWithAttachment(sessionId int64, content string, role MessageRole, attachmentFileID *int64) *Message {
	return &Message{
		SessionId:        sessionId,
		Content:          content,
		Role:             role,
		AttachmentFileID: attachmentFileID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}
