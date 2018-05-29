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

// Attachment
type PostChannelMessageAttachment struct {
	Text    string `json:"text"`
	PreText string `json:"pre-text" json:"text"`
}
type PostChannelMessage struct {
	Token       string                         `json:"text"`
	Channel     string                         `json:"text"`
	AsUser      bool                           `json:"as_user" json:"text"`
	Text        string                         `json:"text"`
	Username    string                         `json:"text"`
	Attachments []PostChannelMessageAttachment `json:"attachments"`
}

// AddAttachment
func (pm *PostChannelMessage) AddAttachment(text string, preText string) *PostChannelMessage {
	pm.Attachments = append(pm.Attachments, PostChannelMessageAttachment{Text: text, PreText: preText})
	return pm
}

// NewMessage
func NewPostChannelMessage(text string, channel string, asUser bool, username string, token string) *PostChannelMessage {
	return &PostChannelMessage{
		Channel:     channel,
		Text:        text,
		AsUser:      asUser,
		Username:	 username,
		Token:       token,
		Attachments: make([]PostChannelMessageAttachment, 0),
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
