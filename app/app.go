package app

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"ivorareteambot/types"
	"log"
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

func (a Application) setHours( value int64 ) {

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
func (a Application) GetCurrentTask(  ) types.Task {
	// Unfound
	currentTask := types.CurrentTask{ID: 1}
	a.db.Last(&currentTask)
	task := types.Task{ID: currentTask.TaskID}
	if (task.ID > 0) {
		a.db.First(&currentTask)
		log.Println("Автовыбор прошлого активного задания по которому шло голосование до перезапуска программы:\n", task)
	}
	return task
}