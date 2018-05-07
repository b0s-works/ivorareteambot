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

func main() {
	dbInit()
	defer db.Close()

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
	log.Println("r.URL.Path:", r.URL.Path)

	cmdText := getSlackCommandStringValue(r)
	log.Println("cmdText:", cmdText)

	switch(r.URL.Path) {

	case "/":
	case "/mytaskhoursbidwillbe":
		hoursBid, err := strconv.ParseInt(cmdText, 10, 64)
		if err != nil {
			sendMsg(fmt.Sprintf("Пожалуйста, укажите целое число! (вы ввели: «%s)", cmdText), w)
			return
		}

		if taskTitle == "" {
			sendMsg_PleaseSpecifyTheTask(w)
			return
		}

		sendMsg(fmt.Sprintf("Ваша оценка для задачи «%s»: %v; Спасибо!", taskTitle, hoursBid), w)
	case "/sethoursbidsubject":
		if (cmdText == "") {
			sendMsg(fmt.Sprintf("Укажите Название задачи например: «%s Название задачи».", r.URL.Path), w)
			return
		}
		// Unfound
		var task Task
		db.FirstOrCreate(&task, &Task{TaskTitle: cmdText})
		fmt.Println(task)

		if task.TaskBiddingDone != 0 {
			sendMsg(fmt.Sprintf("Ставки времени для задачи «%s» уже сделаны! Голосование закрыто.", cmdText), w)
			return
		}

		currentTask = task

		/*if (db_isTaskExists(cmdText)) {
			sendMsg(fmt.Sprintf("Задача «%s» уже существует!", cmdText), w)
		}*/

		sendMsg(fmt.Sprintf("Задача «%s» выдвинута для совершения ставок оценки времени.", cmdText), w)
	}
}

func getSlackCommandStringValue(r *http.Request) string {
	if  r.Method == "POST" {
		r.ParseForm()
		fmt.Println(r.PostFormValue("text"), r)
		return r.PostFormValue("text")
	}
	return r.URL.Query().Get("text")
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
	fmt.Fprintf(w, "%+v\n", string(msgJSON))
}
func sendMsg(msg string, responseWriter http.ResponseWriter) {
	respondJSON(message{Text: msg}, responseWriter)
}

func checkError( error error ) {
	if error != nil {
		panic(error)
	}
}