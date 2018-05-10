package main

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"log"
	"net/http"
	"strconv"
	"ivorareteambot/app"
	"ivorareteambot/controller"
)


func main() {
	db := openDB()
	defer db.Close()

	a := app.New(db)
	c := controller.New(
		a,
		//TODO move it to config.yml
		"someSlackToken",
	)
	c.InitRouters()


	// TODO move it to Application level
	var lastTask LastTask
	db.First(&lastTask, &LastTask{ID: 1})
	if lastTask.TaskID > 0 {
		db.First(&currentTask, &Task{TaskID: lastTask.TaskID})
		fmt.Println("Автовыбор прошлого активного задания по которому шло голосование до перезапуска программы:\n", currentTask)
	}

	serverStart()
}

func openDB() *gorm.DB {

	//TODO move DB credentials to config.yml
	db, err := gorm.Open("mysql", "root:example@tcp(192.168.99.100:3306)/ivorareteambot_db?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
		return nil
	}
	db.LogMode(true)
	return db
}

var db *gorm.DB
var taskTitle string
var currentTask Task

type attachment struct {
	Text string `json:"text"`
}
type message struct {
	Text        string       `json:"text"`
	Attachments []attachment `json:"attachments"`
}

type Task struct {
	TaskID          int `gorm:"primary_key:yes"`
	TaskTitle       string
	TaskBiddingDone int
}
type LastTask struct {
	ID     int
	TaskID int
}

func (LastTask) TableName() string {
	return "last_task"
}

type TaskHoursBidAndMember struct {
	ID             int
	TaskID         int
	MemberIdentity string
	MemberTimeBid  int64
	MemberNick     string
}



func sendMsg_PleaseSpecifyTheTask(w http.ResponseWriter) {
	var msg message
	msg.Text = "Зайдайте Название задачи для которой хотите провести командную оценку времени"
	msg.Attachments = append(msg.Attachments, attachment{Text: "Задать Название задачи можно с помощью команды /setratingsubject"})

	respondJSON(msg, w)
}


//MOVE IT to main
func serverStart() {
	httpPort := 80
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	fmt.Printf("listening on %v\n", httpPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil))
}
func sendMsgOnRwsAffctdOrErr(w http.ResponseWriter, rslt *gorm.DB, scsMsg string, scsMsgData []interface{}, errMsg string, errMsgData []interface{}) bool {
	if rslt.RowsAffected == 1 {
		sendMsg(w, scsMsg, scsMsgData...)
		return true
	}
	if rslt.Error != nil {
		sendMsg(w, errMsg, errMsgData)
	}
	return false
}

func badRouting() {
	http.HandleFunc("/", HandleRouter)
	http.HandleFunc("/tbb_myhoursbidwillbe", HandleRouter)
	http.HandleFunc("/tbb_setbidtask", HandleRouter)
	http.HandleFunc("/tbb_listtaskbids", HandleRouter)
	http.HandleFunc("/tbb_list", HandleRouter)
	http.HandleFunc("/tbb_removetask", HandleRouter)
}


func HandleRouter(w http.ResponseWriter, req *http.Request) {

	switch req.URL.Path {

	case "/":
	case "/tbb_myhoursbidwillbe":
		fmt.Println(req.Body)
		hoursBid, err := strconv.ParseInt(cmdText, 10, 64)
		if err != nil {
			sendMsg(w, "Пожалуйста, укажите целое число! (вы ввели: «%s»)", cmdText)
			return
		}

		if currentTask.TaskTitle == "" {
			sendMsg_PleaseSpecifyTheTask(w)
			return
		}

		//checkDBError(db.FirstOrCreate(&task, &Task{TaskTitle: cmdText}).Error, w)

		UserID := getSlackValueFromPostOrGet("user_id", req)
		UserName := getSlackValueFromPostOrGet("user_name", req)

		var taskHoursBidAndMember TaskHoursBidAndMember
		db.First(&taskHoursBidAndMember, "task_id = ? and member_identity = ?", currentTask.TaskID, UserID)
		fmt.Printf("taskHoursBidAndMember: %v %+v\n", taskHoursBidAndMember.TaskID > 0, taskHoursBidAndMember)
		if taskHoursBidAndMember.TaskID > 0 {
			fmt.Println("We have to make update:")

			oldBid := taskHoursBidAndMember.MemberTimeBid

			taskHoursBidAndMember.MemberNick = UserName
			taskHoursBidAndMember.MemberTimeBid = hoursBid

			updateResult := db.Save(&taskHoursBidAndMember)
			respondMessage(fmt.Sprintf("Ваша оценка для задачи «%s» изменена с %v на %v\nСпасибо!", currentTask.TaskTitle, oldBid, hoursBid))
			sendMsgOnRwsAffctdOrErr(w, updateResult,
				"Ваша оценка для задачи «%s» изменена с %v на %v\nСпасибо!", []interface{}{currentTask.TaskTitle, oldBid, hoursBid},
				"При обновлении оценки по задаче «%s» произошла ошибка:\n", []interface{}{updateResult.Error},
			)
		}
		//fmt.Printf( "New record data\n - %+v",  )
		createResult := db.Create(&TaskHoursBidAndMember{TaskID: currentTask.TaskID, MemberIdentity: UserID, MemberNick: UserName, MemberTimeBid: hoursBid})
		sendMsgOnRwsAffctdOrErr(w, createResult,
			"Ваша оценка для задачи «%s»: %v\nСпасибо!", []interface{}{currentTask.TaskTitle, hoursBid},
			"При добавлении оценки по задаче «%s» произошла ошибка:\n", []interface{}{createResult.Error},
		)
	case "/tbb_setbidtask":
		if cmdText == "" {
			sendMsg(w, "Укажите Название задачи например: «%s Название задачи».", req.URL.Path)
			return
		}
		// Unfound
		var task = Task{TaskTitle: cmdText}
		if err := db.FirstOrCreate(&task, "task_title = ?", cmdText).Error; err != nil {
			sendMsg(w, "При выборе задачи «%s» произошла ошибка:\n%v", cmdText, err)
			return
		}

		fmt.Printf("First or create task result:\n%+v", task)

		if task.TaskBiddingDone != 0 {
			sendMsg(w, "Ставки времени для задачи «%s» уже сделаны! Голосование закрыто.", cmdText)
			return
		}

		db.Save(&LastTask{ID: 1, TaskID: task.TaskID})

		currentTask = task

		/*if (db_isTaskExists(cmdText)) {
			sendMsg(w, "Задача «%s» уже существует!", cmdText)
		}*/

		sendMsg(w, "Задача «%s» выдвинута для совершения ставок оценки времени.", cmdText)
	case "/tbb_listtaskbids":
		if cmdText == "" {
			sendMsg(w, "Укажите Название задачи например: «%s Название задачи».", req.URL.Path)
			return
		}
		// Unfound
		var task Task
		db.Where(&Task{TaskTitle: cmdText}).First(&task)
		if task.TaskID == 0 {
			sendMsg(w, "Задача с точным названием «%s» не найдена.", cmdText)
			return
		}
		var membersAndBids []TaskHoursBidAndMember
		db.Where(&TaskHoursBidAndMember{TaskID: task.TaskID}).Find(&membersAndBids)
		if len(membersAndBids) > 0 {
			var resultMembersAndBidsList string
			for _, memberAndBid := range membersAndBids {
				resultMembersAndBidsList += fmt.Sprintf(
					"%v - %s (%s)",
					memberAndBid.MemberTimeBid,
					memberAndBid.MemberNick,
					memberAndBid.MemberIdentity,
				)
				fmt.Println("memberAndBid:", memberAndBid)
			}
			sendMsg(w, "Для задачи «%s» участниками были сделаны следующие ставки:\n%s", cmdText, resultMembersAndBidsList)
			return
		} else {
			sendMsg(w, "Ставок времени для задачи «%s» не сделано.", cmdText)
			return
		}
		fmt.Println(task)

	case "/tbb_list":
		// Unfound
		var tasks []Task

		if db.Select("task_id, task_title, task_bidding_done").Find(&tasks).RecordNotFound() {
			sendMsg(w, "Список задач пуст.")
			return
		}
		var TaskBiddingDoneString string
		for _, task := range tasks {
			TaskBiddingDoneString = "открыто"
			if task.TaskBiddingDone > 0 {
				TaskBiddingDoneString = "совершено"
			}
			fmt.Fprintf(w, "%v. «%s» (голосование %s)", task.TaskID, task.TaskTitle, TaskBiddingDoneString)
		}
	case "/tbb_removetask":
		if cmdText == "" {
			sendMsg(w, "Укажите Название задачи например: «%s Название задачи».", req.URL.Path)
			return
		}
		// Unfound
		var task Task
		if db.First(&task, &Task{TaskTitle: cmdText}).RecordNotFound() {
			sendMsg(w, "Задача «%s» не найдена.", cmdText)
			return
		}

		if db.Delete(Task{}, "task_id = ?", task.TaskID).RowsAffected == 1 {
			sendMsg(w, "Задача «%s» удалена.", cmdText)
			if db.Delete(TaskHoursBidAndMember{}, "task_id = ?", task.TaskID).RowsAffected > 0 {
				sendMsg(w, "Так же удалены все связанные с ней ставки участников.")
			}
		}

		var deletableTaskID = task.TaskID
		if deletableTaskID == currentTask.TaskID {
			currentTask = Task{}
			sendMsg(w, "Задача «%s» так же снята с голосования.", cmdText)
		}
	}
}


func getSlackToken(r *http.Request) string {
	return getSlackValueFromPostOrGet("token", r)
}
func getSlackCommandStringValue(r *http.Request) string {
	return getSlackValueFromPostOrGet("text", r)
}

func getSlackValueFromPostOrGet(value string, r *http.Request) string {
	if r.Method == "POST" {
		return getSlackPostFieldValue(value, r)
	}
	return getSlackGetQueryParameterValue(value, r)
}

func getSlackPostFieldValue(value string, r *http.Request) string {
	return r.PostFormValue(value)
}
func getSlackGetQueryParameterValue(value string, r *http.Request) string {
	return r.URL.Query().Get(value)
}

func respondJSON(message message, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "app/json")

	msgJSON, _ := json.Marshal(message)
	fmt.Fprint(w, string(msgJSON))
}
func sendMsg(responseWriter http.ResponseWriter, msg string, s ...interface{}) {
	respondJSON(message{Text: fmt.Sprintf(msg, s...)}, responseWriter)
}

func checkError(error error) {
	if error != nil {
		panic(error)
	}
}
func checkDBError(error error, w http.ResponseWriter) {
	if error != nil {
		sendMsg(w, "Возникла ошибка при запросе в базу данных.\nОшибка: %s", error)
		//panic(error)
		return
	}
}
