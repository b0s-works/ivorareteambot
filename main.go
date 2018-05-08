package main

import (
	"encoding/json"
	"net/http"
	"fmt"
	"log"
	"strconv"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

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
	TaskID          int
	TaskTitle       string
	TaskBiddingDone int
}
type LastTask struct {
	ID     int
	TaskID int
}
type SlackToken struct {
	slackToken string
}

func (LastTask) TableName() string {
	return "last_task"
}

type TaskHoursBidAndMember struct {
	TaskID         int
	MemberIdentity string
	MemberTimeBid  int64
	MemberNick     string
}

func main() {
	dbInit()
	defer db.Close()
	// Unfound
	var lastTask LastTask
	db.First(&lastTask, &LastTask{ID: 1})
	if (lastTask.TaskID > 0) {
		db.First(&currentTask, &Task{TaskID: lastTask.TaskID})
		fmt.Println("Автовыбор прошлого активного задания по которому шло голосование до перезапуска программы:\n", currentTask)
	}

	badRouting()
	serverStart()
}

func sendMsg_PleaseSpecifyTheTask(w http.ResponseWriter) {
	var msg message
	msg.Text = "Зайдайте Название задачи для которой хотите провести командную оценку времени"
	msg.Attachments = append(msg.Attachments, attachment{Text: "Задать Название задачи можно с помощью команды /setratingsubject"})

	respondJSON(msg, w)
}

func serverStart() {
	httpPort := 80
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	fmt.Printf("listening on %v\n", httpPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil))
}

func badRouting() {
	http.HandleFunc("/", HandleRouter)
	http.HandleFunc("/mytaskhoursbidwillbe", HandleRouter)
	http.HandleFunc("/sethoursbidsubject", HandleRouter)
}
func HandleRouter(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var slackTokenFromForm string = getSlackToken(r);
	log.Println("r.URL.Path:", r.URL.Path, slackTokenFromForm)

	var slackToken SlackToken
	if db.Where("slack_token = ?", slackTokenFromForm).First(&slackToken).RecordNotFound() {
		sendMsg(
			fmt.Sprintf(
				"Токен этого рабочего пространства не найден в базе данных ivorareteambot. Попросите владельца рабочего пространства <<%s>> добавить секретный токен в базу данных бота.",
				getSlackPostFieldValue("team_domain", r), ), w)
		return
	}

	fmt.Println("slackToken: ", slackToken);
	cmdText := getSlackCommandStringValue(r)
	log.Println("cmdText:", cmdText)

	switch(r.URL.Path) {

	case "/":
	case "/mytaskhoursbidwillbe":
		fmt.Println(r.Body)
		hoursBid, err := strconv.ParseInt(cmdText, 10, 64)
		if err != nil {
			sendMsg(fmt.Sprintf("Пожалуйста, укажите целое число! (вы ввели: «%s)", cmdText), w)
			return
		}

		if currentTask.TaskTitle == "" {
			sendMsg_PleaseSpecifyTheTask(w)
			return
		}

		//checkDBError(db.FirstOrCreate(&task, &Task{TaskTitle: cmdText}).Error, w)

		UserID := getSlackPostFieldValue("user_id", r)
		UserName := getSlackPostFieldValue("user_name", r)

		var newMemberBidData = TaskHoursBidAndMember{
			TaskID:        currentTask.TaskID,
			MemberNick:    UserName,
			MemberTimeBid: hoursBid,
		}

		checkDBError(db.Where(&TaskHoursBidAndMember{MemberIdentity: UserID}).
			Attrs(&newMemberBidData).
			FirstOrCreate(&newMemberBidData).Error, w)

		sendMsg(fmt.Sprintf("Ваша оценка для задачи «%s»: %v; Спасибо!", currentTask.TaskTitle, hoursBid), w)
	case "/sethoursbidsubject":
		if (cmdText == "") {
			sendMsg(fmt.Sprintf("Укажите Название задачи например: «%s Название задачи».", r.URL.Path), w)
			return
		}
		// Unfound
		var task Task
		checkDBError(db.FirstOrCreate(&task, &Task{TaskTitle: cmdText}).Error, w)
		fmt.Println(task)

		if task.TaskBiddingDone != 0 {
			sendMsg(fmt.Sprintf("Ставки времени для задачи «%s» уже сделаны! Голосование закрыто.", cmdText), w)
			return
		}

		currentTask = task

		/*if (db_isTaskExists(cmdText)) {
			sendMsg(fmt.Sprintf("Задача «%s» уже существует!", cmdText), w)
		}*/
		db.Where(&LastTask{ID: 1}).
			Attrs(&LastTask{TaskID: task.TaskID}).
			FirstOrCreate(&LastTask{ID: 1, TaskID: task.TaskID})

		sendMsg(fmt.Sprintf("Задача «%s» выдвинута для совершения ставок оценки времени.", cmdText), w)
	case "/taskbidsbot_listbids":
		if cmdText == "" {
			sendMsg(fmt.Sprintf("Укажите Название задачи например: «%s Название задачи».", r.URL.Path), w)
			return
		}
		// Unfound
		var task Task
		db.Where(&Task{TaskTitle: cmdText}).First(&task)
		if task.TaskID == 0 {
			sendMsg(fmt.Sprintf("Задача с точным названием «%s» не найдена.", cmdText), w)
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
				fmt.Println("memberAndBid:", memberAndBid);
			}
			sendMsg(fmt.Sprintf("Для задачи «%s» участниками были сделаны следующие ставки:\n%s", cmdText, resultMembersAndBidsList), w)
			return
		} else {
			sendMsg(fmt.Sprintf("Ставок времени для задачи «%s» не сделано.", cmdText), w)
			return
		}
		fmt.Println(task)
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

func dbInit() {
	var err error
	db, err = gorm.Open("mysql", "root:example@tcp(192.168.99.100:3306)/ivorareteambot_db?charset=utf8&parseTime=True&loc=Local")
	db.LogMode(true)

	checkError(err)
}

func respondJSON(message message, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")

	msgJSON, _ := json.Marshal(message)
	fmt.Fprint(w, string(msgJSON))
}
func sendMsg(msg string, responseWriter http.ResponseWriter) {
	respondJSON(message{Text: msg}, responseWriter)
}

func checkError(error error) {
	if error != nil {
		panic(error)
	}
}
func checkDBError(error error, w http.ResponseWriter) {
	if error != nil {
		sendMsg(fmt.Sprintf("Возникла ошибка при запросе в базу данных.\nОшибка: %s", error), w)
		//panic(error)
		return
	}
}
