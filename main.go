package main

import (
	"encoding/json"
	"net/http"
	"fmt"
	"log"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"ivorareteambot/project/app"
	"ivorareteambot/project/controller"
	"github.com/jinzhu/configor"
)

const dbLoggingEnabled = true

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
type SlackToken struct {
	slackToken string
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

var dbConfig = struct {
	APPName string `default:"ivorareteambot"`

	DB struct {
		Name     string
		User     string `default:"root"`
		Password string `required:"true" env:"example"`

		Host      string `default:"127.0.0.1"`
		Port      uint   `default:"3306"`
		Charset   string `default:"utf8"`
		ParseTime string `default:"true"`
	}
}{}
var slackConfig = struct {
	Token string `default:"someSlackToken"`
}{}

func main() {
	configor.Load(&dbConfig, "config/database.yml")
	fmt.Printf("config: %#v", dbConfig)

	configor.Load(&slackConfig, "config/slack.yml")
	fmt.Printf("config: %#v", slackConfig)

	db, dbErr := openDB("mysql", dbConfig)
	if dbErr != nil {
		panic(dbErr)
	}
	defer db.Close()

	a := app.New(db)
	c := controller.New(
		a,
		slackConfig.Token,
	)

	c.InitRouters()
	badRouting()
	serverStart()
}
func openDB(dialect string, config interface{}) (*gorm.DB, error) {
	var dsn = fmt.Sprintf(
		"%s:%s@tcp(%s:%v)/%s?charset=%s&parseTime=%s&loc=Local",
		config.DB.User,
		config.DB.Password,
		config.DB.Host,
		config.DB.Port,
		config.DB.Name,
		config.DB.Charset,
		config.DB.ParseTime,
	)

	fmt.Printf("dsn: %+v", dsn)

	db, err := gorm.Open( dialect, dsn )
	db.LogMode( dbLoggingEnabled )

	return db, err
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
func requestTokenAndSearchitInDb(w http.ResponseWriter, r *http.Request) SlackToken {
	var slackTokenFromForm = getSlackToken(r)
	log.Println("r.URL.Path:", r.URL.Path, slackTokenFromForm)

	var slackToken SlackToken
	statement := db.Where("slack_token = ?", slackTokenFromForm).First(&slackToken)
	if statement.Error != nil {
		sendMsg(w,
			"Произошла ошибка при запросе токена из базы данных:\n%+v",
			statement.Error,
		)
	} else if statement.RecordNotFound() {
		sendMsg(w, "Токен этого рабочего пространства не найден в базе данных ivorareteambot. Попросите владельца рабочего пространства «%s» добавить секретный токен в базу данных бота.",
			getSlackPostFieldValue("team_domain", r),
		)
	}
	return slackToken
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
	w.Header().Set("Content-Type", "application/json")

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
