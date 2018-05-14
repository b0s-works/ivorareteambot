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
func New(db *gorm.DB) Application {
	return Application{
		db: db,
	}
}

func (a Application) RemoveTaskById(taskId int) (int64, error) {
	statement := a.db.Delete(types.Task{}, "task_id = ?", taskId)
	if statement.Error != nil {
		return statement.RowsAffected, statement.Error
	}
	return statement.RowsAffected, nil
}
func (a Application) RemoveTaskChildHoursById(taskId int) (int64, error) {
	statement := a.db.Delete(types.TaskHoursBidAndMember{}, "task_id = ?", taskId)
	if statement.Error != nil {
		return statement.RowsAffected, statement.Error
	}
	return statement.RowsAffected, nil
}
func (a Application) RemoveTaskByIdAndChildHours(taskID int) (int64, int64, error) {
	rowsAffected, err := a.RemoveTaskById(taskID)
	if err != nil {
		return rowsAffected, 0, err
	}
	if rowsAffected == 1 {
		childRowsAffected, err := a.RemoveTaskChildHoursById(taskID)
		if err != nil {
			return rowsAffected, childRowsAffected, err
		}
		return rowsAffected, childRowsAffected, nil
	}
	return rowsAffected, 0, nil
}

func (a Application) GetAllTasks() ([]types.Task, error) {
	var tasks []types.Task
	if err := a.db.Table("task").Select("task_id, task_title, task_bidding_done").Find(&tasks).Error; err != nil {
		return tasks, err
	}
	return tasks, nil
}
func (a Application) GetTaskHoursBids(taskId int) ([]types.TaskHoursBidAndMember, error) {
	var membersAndBids []types.TaskHoursBidAndMember
	if err := a.db.Where(&types.TaskHoursBidAndMember{TaskID: taskId}).Find(&membersAndBids).Error; err != nil {
		return membersAndBids, err
	}
	return membersAndBids, nil
}
func (a Application) GetTask(title string) (types.Task, error) {
	var task = types.Task{Title: title}
	if err := a.db.FirstOrCreate(&task, "title = ?", title).Error; err != nil {
		return task, err
	}
	return task, nil
}
func (a Application) SetTask(id int) error {
	return a.db.Save(&types.CurrentTask{ID: 1, TaskID: id}).Error
}

func (a Application) SetHours(value int64) error {
	//checkDBError(db.FirstOrCreate(&task, &Task{Title: cmdText}).Error, w)

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

//TODO Token depended last task getting
func (a Application) GetCurrentTask() (types.Task, error) {
	currentTask := types.CurrentTask{ID: 1}
	if err := a.db.Last(&currentTask).Error; err != nil {
		return types.Task{ID: currentTask.TaskID}, err
	}

	task := types.Task{ID: currentTask.TaskID}
	if task.ID > 0 {
		if err := a.db.First(&currentTask).Error; err != nil {
			return task, err
		}
		log.Println("Автовыбор прошлого активного задания по которому шло голосование до перезапуска программы:\n", task)
	}
	return task, nil
}
