package app

import (
	"fmt"
	"log"
	"github.com/jinzhu/gorm"

	"ivorareteambot/types"
	"github.com/sirupsen/logrus"
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

// RemoveTaskByID removes task by it's int primary key identifier
func (a Application) RemoveTaskByID(taskID int) (int64, error) {
	statement := a.db.Delete(types.Task{}, "id = ?", taskID)
	if statement.Error != nil {
		return statement.RowsAffected, statement.Error
	}
	return statement.RowsAffected, nil
}

// RemoveTaskChildHoursByID removes hours which was voted/offered/bidded by voters by task primary key int identifier
func (a Application) RemoveTaskChildHoursByID(taskID int) (int64, error) {
	statement := a.db.Delete(types.TaskHoursBidAndMember{}, "task_id = ?", taskID)
	if statement.Error != nil {
		return statement.RowsAffected, statement.Error
	}
	return statement.RowsAffected, nil
}
func (a Application) RemoveTaskByIdAndChildHours(taskID int) (int64, error) {
	rowsAffected, err := a.RemoveTaskByID(taskID)
	if err != nil {
		return 0, err
	}
	if rowsAffected == 1 {
		childRowsAffected, err := a.RemoveTaskChildHoursByID(taskID)
		if err != nil {
			return childRowsAffected, err
		}
		return childRowsAffected, nil
	}
	return 0, nil
}

//GetAllTasks Used to gather all tasks with theirs <<voting complete>> status
func (a Application) GetAllTasks() ([]types.Task, error) {
	var tasks []types.Task
	if err := a.db.Find(&tasks).Error; err != nil {
		return tasks, err
	}
	return tasks, nil
}

//GetTaskHoursBids Gets hours which was recorded when voting
func (a Application) GetTaskHoursBids(taskID int) ([]types.TaskHoursBidAndMember, error) {
	var membersAndBids []types.TaskHoursBidAndMember
	if err := a.db.Where(&types.TaskHoursBidAndMember{TaskID: taskID}).Find(&membersAndBids).Error; err != nil {
		return membersAndBids, err
	}
	return membersAndBids, nil
}

//GetTask Gets current votable task
func (a Application) GetTask(title string) (types.Task, error) {
	var task = types.Task{Title: title}
	if err := a.db.FirstOrCreate(&task, "title = ?", title).Error; err != nil {
		return task, err
	}
	return task, nil
}

//SetTask Sets the task for voting
func (a Application) SetTask(id int) error {
	return a.db.Save(&types.CurrentTask{ID: 1, TaskID: id}).Error
}

//TODO Token depended last task getting
func (a Application) GetUserBidByTaskIDAndUserIdentity(taskId int, UserIdentity string) (types.TaskHoursBidAndMember, error) {
	logrus.Println("GetCurrentTask:", taskId, UserIdentity, "Запрос прошлого активного задания по которому шло голосование до перезапуска программы...")

	var taskHoursBidAndMember = types.TaskHoursBidAndMember{
		TaskID:         taskId,
		MemberIdentity: UserIdentity,
	}
	if stmt := a.db.First(&taskHoursBidAndMember); stmt.Error != nil && !stmt.RecordNotFound() {
		logrus.Println(fmt.Sprintf("taskHoursBidAndMember: %+v", &taskHoursBidAndMember))
		return taskHoursBidAndMember, stmt.Error
	}

	return taskHoursBidAndMember, nil
}

func (a Application) SetHours(hours int64, taskHoursBidAndMember types.TaskHoursBidAndMember) (int64, error) {
	saveResult := a.db.Save(&taskHoursBidAndMember)
	if err := saveResult.Error; err != nil {
		return saveResult.RowsAffected, err
	}

	return saveResult.RowsAffected, nil
}

// GetCurrentTask Gets current task from Database
// TODO Token depended last task getting
func (a Application) GetCurrentTask() (types.Task, error) {
	logrus.Println("GetCurrentTask:", "Запрос прошлого активного задания по которому шло голосование до перезапуска программы...")
	currentTask := types.CurrentTask{ID: 1}
	if err := a.db.Last(&currentTask).Error; err != nil {
		return types.Task{ID: currentTask.TaskID}, err
	}

	task := types.Task{ID: currentTask.TaskID}
	log.Printf("Task structured object: %+v", task)
	if task.ID > 0 {
		if err := a.db.First(&task).Error; err != nil {
			return task, err
		}
		log.Printf("Found task: %+v", task)
	}
	return task, nil
}
