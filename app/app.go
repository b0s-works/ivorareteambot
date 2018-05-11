package app

import (
	"github.com/jinzhu/gorm"
	"fmt"
	"strconv"
)

//Application Main Application structure
type Application struct {
	db *gorm.DB

}

//New Application constructor
func New ( db *gorm.DB ) Application {
	return Application{
		db: db,
	}
}

type LastTask struct {
	ID     int
	TaskID int
}
func (LastTask) TableName() string {
	return "last_task"
}

func (a Application) Slash_myHoursBidWillBe( value int64 ) {
	curTsk := a.getLastTask()
	
	if curTsk.Title == "" {
		var msg message
		msg.Text = "Зайдайте Название задачи для которой хотите провести командную оценку времени"
		msg.Attachments = append(msg.Attachments, attachment{Text: "Задать Название задачи можно с помощью команды /setratingsubject"})

		respondJSON(msg, w)
	return
	}

	//checkDBError(db.FirstOrCreate(&task, &Task{Title: cmdText}).Error, w)

	UserID := getSlackValueFromPostOrGet("user_id", r)
	UserName := getSlackValueFromPostOrGet("user_name", r)

	var taskHoursBidAndMember TaskHoursBidAndMember
	db.First(&taskHoursBidAndMember, "task_id = ? and member_identity = ?", curTsk.ID, UserID)
	fmt.Printf("taskHoursBidAndMember: %v %+v\n", taskHoursBidAndMember.TaskID > 0, taskHoursBidAndMember)
	if taskHoursBidAndMember.TaskID > 0 {
	fmt.Println("We have to make update:")

	oldBid := taskHoursBidAndMember.MemberTimeBid

	taskHoursBidAndMember.MemberNick = UserName
	taskHoursBidAndMember.MemberTimeBid = hoursBid

	updateResult := db.Save(&taskHoursBidAndMember)
	sendMsgOnRwsAffctdOrErr(w, updateResult,
	"Ваша оценка для задачи «%s» изменена с %v на %v\nСпасибо!", []interface{}{curTsk.Title, oldBid, hoursBid},
	"При обновлении оценки по задаче «%s» произошла ошибка:\n", []interface{}{updateResult.Error},
	)
	}
	//fmt.Printf( "New record data\n - %+v",  )
	createResult := db.Create(&TaskHoursBidAndMember{TaskID: curTsk.ID, MemberIdentity: UserID, MemberNick: UserName, MemberTimeBid: hoursBid})
	sendMsgOnRwsAffctdOrErr(w, createResult,
	"Ваша оценка для задачи «%s»: %v\nСпасибо!", []interface{}{curTsk.Title, hoursBid},
	"При добавлении оценки по задаче «%s» произошла ошибка:\n", []interface{}{createResult.Error},
)
}


type Task struct {
	ID          int `gorm:"primary_key:yes"`
	Title       string
	BiddingDone int
}
//TODO Token depended last task getting
func (a Application) getLastTask(  ) Task {
	// Unfound
	var lastTask LastTask
	a.db.First(&lastTask, &LastTask{ID: 1})
	var currentTask Task
	if (lastTask.TaskID > 0) {
		a.db.First(&currentTask, &Task{ID: lastTask.TaskID})
		fmt.Println("Автовыбор прошлого активного задания по которому шло голосование до перезапуска программы:\n", currentTask)
	}
	return currentTask
}