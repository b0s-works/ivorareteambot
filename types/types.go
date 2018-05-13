package types

type Attachment struct {
	Text string `json:"text"`
}
type Message struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}
type Task struct {
	ID          int `gorm:"primary_key:yes"`
	Title       string
	BiddingDone int
}
type CurrentTask struct {
	ID     int
	TaskID int
}
type SlackToken struct {
	slackToken string
}

func (CurrentTask) TableName() string {
	return "current_task"
}

type TaskHoursBidAndMember struct {
	ID             int
	TaskID         int
	MemberIdentity string
	MemberTimeBid  int64
	MemberNick     string
}