package controller

import (
	"ivorareteambot/app"
	"github.com/gin-gonic/gin"
	"net/http"
	"fmt"
	"log"
	"strconv"
	"encoding/json"
	"ivorareteambot/types"
)

const slackToken = "slackToken"

type Controller struct {
	app app.Application
	gin *gin.Engine
	slackToken string
}

func New( app app.Application, slackToken string ) Controller {
	return Controller{
		app: app,
		gin: gin.Default(),
		slackToken: slackToken,
	}
}

func (c Controller) Serve() {
	http.Handle("/", c.gin)
}

func (c Controller) InitRouters() {
	slack := c.gin.Group("")
	slack.Use(c.slackAuth)
	slack.POST("/", c.requestHandler)
}

func ( c Controller ) requestHandler( g *gin.Context ) {
	cmdText := g.Param("text")
	log.Println("cmdText:", cmdText)

	switch(g.Request.URL.Path) {

	case "/":
	case "/tbb_myhoursbidwillbe":
		curTsk := c.app.GetCurrentTask()
		if curTsk.Title == "" {
			var msg types.Message
			msg.Text = "Зайдайте Название задачи для которой хотите провести командную оценку времени"
			msg.Attachments = append(msg.Attachments, types.Attachment{Text: "Задать Название задачи можно с помощью команды /setratingsubject"})

			respondJSON(msg, w)
			return
		}
		fmt.Println(g.Params)
		hoursBid, err := strconv.ParseInt(cmdText, 10, 64)
		if err != nil {
			sendMsg(g, "Пожалуйста, укажите целое число! (вы ввели: «%s»)", cmdText)
			return
		}
		c.app.Slash_myHoursBidWillBe( hoursBid )
	case "/tbb_setbidtask":
		if (cmdText == "") {
			sendMsg(w, "Укажите Название задачи например: «%s Название задачи».", r.URL.Path)
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
			sendMsg(w, "Укажите Название задачи например: «%s Название задачи».", r.URL.Path)
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
		if (cmdText == "") {
			sendMsg(w, "Укажите Название задачи например: «%s Название задачи».", r.URL.Path)
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

func (c Controller) slackAuth(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		//TODO respond valid to Slack error
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		ctx.Abort()
		return
	}

	if t := form.Value[slackTokenKey]; len(t) == 0 || t[0] != c.slackToken {
		//TODO respond error
		ctx.Abort()
		return
	}

	ctx.Next()
}


type attachment struct {
	Text string `json:"text"`
}
type message struct {
	Text        string       `json:"text"`
	Attachments []attachment `json:"attachments"`
}
func respondJSON(message message, g *gin.Context) {
	msgJSON, _ := json.Marshal(message)
	//string(msgJSON)
	g.JSON(http.StatusOK, string(msgJSON) )
}
func sendMsg(g *gin.Context, msg string, s ...interface{}) {
	respondJSON(message{Text: fmt.Sprintf(msg, s...)}, g)
}