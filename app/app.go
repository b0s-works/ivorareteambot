package app

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"ivorareteambot/types"
	"log"
	"runtime"
)

//Application Main Application structure
type Application struct {
	db *gorm.DB
}

func GetFunctionName() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return fmt.Sprintf("%s:%d", frame.Line, frame.Function)
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

//TODO Token depended last task getting
func (a Application) GetUserBidByTaskIDAndUserIdentity(taskId int, UserIdentity string) (types.TaskHoursBidAndMember, error) {
	log.Println("GetCurrentTask:", "Запрос прошлого активного задания по которому шло голосование до перезапуска программы...")

	var taskHoursBidAndMember types.TaskHoursBidAndMember
	if err := a.db.First(&taskHoursBidAndMember, "task_id = ? and member_identity = ?", taskId, UserIdentity).Error; err != nil {
		return taskHoursBidAndMember, err
	}

	return taskHoursBidAndMember, nil
}

func (a Application) SetHours(hours int64, taskHoursBidAndMember types.TaskHoursBidAndMember) (int64, error) {
	//checkDBError(db.FirstOrCreate(&task, &Task{Title: cmdText}).Error, w)
	fmt.Printf("%s:", "taskHoursBidAndMember: %v %+v\n", GetFunctionName(), taskHoursBidAndMember.TaskID > 0, taskHoursBidAndMember)


	saveResult := a.db.Save(&taskHoursBidAndMember)
	if err := saveResult.Error; err != nil {
		return saveResult.RowsAffected, err
	}

	return saveResult.RowsAffected, nil
}

//TODO Token depended last task getting
func (a Application) GetCurrentTask() (types.Task, error) {
	log.Println("GetCurrentTask:", "Запрос прошлого активного задания по которому шло голосование до перезапуска программы...")
	currentTask := types.CurrentTask{ID: 1}
	if err := a.db.Last(&currentTask).Error; err != nil {
		return types.Task{ID: currentTask.TaskID}, err
	}

	task := types.Task{ID: currentTask.TaskID}
	if task.ID > 0 {
		if err := a.db.First(&currentTask).Error; err != nil {
			return task, err
		}
	}
	return task, nil
}
