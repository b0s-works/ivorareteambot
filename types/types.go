package types

// Attachment
type Attachment struct {
	Text string `json:"text"`
}

// Message
type Message struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

// AddAttachment
func (m *Message) AddAttachment(text string) *Message {
	m.Attachments = append(m.Attachments, Attachment{text})
	return m
}

// NewMessage
func NewMessage(text string) *Message {
	return &Message{
		Text:        text,
		Attachments: make([]Attachment, 0),
	}
}

// Task
type Task struct {
	ID          int `gorm:"primary_key:yes"`
	Title       string
	BiddingDone int
}

// CurrentTask
type CurrentTask struct {
	ID     int
	TaskID int
}

// SlackToken
type SlackToken struct {
	slackToken string
}

// TableName
func (CurrentTask) TableName() string {
	return "current_task"
}

// TaskHoursBidAndMember
type TaskHoursBidAndMember struct {
	ID             int
	TaskID         int
	MemberIdentity string
	MemberTimeBid  int64
	MemberNick     string
}
