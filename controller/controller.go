package controller

import (
	"net/http"
	"fmt"
	"log"
	"strconv"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"ivorareteambot/app"
	"ivorareteambot/types"
)

const slackToken = "slackToken"

type Controller struct {
	app        app.Application
	gin        *gin.Engine
	slackToken string
}

func New(app app.Application, slackToken string) Controller {
	return Controller{
		app:        app,
		gin:        gin.Default(),
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

func (c Controller) checkTaskSelected(gCtx *gin.Context) types.Task {
	return curTsk
}

func (c Controller) checkDBError( err error, prefixMsg string, gCtx *gin.Context ) {
	if err != nil {
		var msg = types.Message{Text: fmt.Sprintf("%s:\n%v", prefixMsg, err)}
		gCtx.JSON(http.StatusInternalServerError, msg)
		panic(msg)
		return
	}
}

func (c Controller) requestHandler(gCtx *gin.Context) {
	cmdText := gCtx.Params.ByName("text")
	log.Println("cmdText:", cmdText)

	curTsk, err := c.app.GetCurrentTask()
	c.checkDBError( err, "При запросе текущей задачи произошла ошибка", gCtx )

	switch(gCtx.Request.URL.Path) {
	case "/":

	case "/tbb_sethours":
		if curTsk.Title == "" {
			msg := types.Message{Text: "Зайдайте «Название задачи» для которой хотите провести командную оценку времени"}
			msg.Attachments = append(msg.Attachments, types.Attachment{Text: "Задать «Название задачи» можно с помощью команды «/tbb_settask»"})

			gCtx.JSON(http.StatusOK, msg)
			return
		}
		fmt.Println(gCtx.Params)

		hoursBid, err := strconv.ParseInt(cmdText, 10, 64)
		if err != nil {
			gCtx.JSON(
				http.StatusOK,
				types.Message{Text: fmt.Sprintf("Пожалуйста, укажите целое число! (вы ввели: «%s»)", cmdText)},
			)
			return
		}
		c.app.SetHours(hoursBid)
		return

	case "/tbb_setbidtask":
		if cmdText == "" {
			gCtx.JSON(
				http.StatusOK,
				types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)},
			)
			return
		}
		task, err := c.app.GetTask(cmdText)
		c.checkDBError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%v", cmdText), gCtx)

		log.Printf("First or create task result:\n%+v", task)

		if task.BiddingDone != 0 {
			gCtx.JSON(
				http.StatusOK,
				types.Message{Text: fmt.Sprintf("Ставки времени для задачи «%s» уже сделаны! Голосование закрыто.", cmdText)},
			)
			return
		}

		err = c.app.SetTask(task.ID)
		c.checkDBError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%v", cmdText), gCtx)

		gCtx.JSON(
			http.StatusOK,
			types.Message{Text: fmt.Sprintf("Задача «%s» выдвинута для совершения ставок оценки времени.", cmdText)},
		)
		return

	case "/tbb_listtaskbids":
		if cmdText == "" {
			gCtx.JSON(
				http.StatusOK,
				types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)},
			)
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
			gCtx.JSON(
				http.StatusOK,
				types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)},
			)
			return
		}
		// Unfound
		task, err := c.app.GetTask(cmdText)
		c.checkDBError(err, fmt.Sprintf("При поиске задачи «%s» в базе данных произошла ошибка:\n%v", cmdText), gCtx)

		if task.ID == 0 {
			gCtx.JSON(
				http.StatusOK,
				types.Message{Text: fmt.Sprintf("Задача «%s» не найдена.", cmdText)},
			)
			return
		}

		rowsAffected, err := c.app.RemoveTaskByIdAndChildHours( task.ID )
		c.checkDBError(err, fmt.Sprintf("При удалении задачи «%s» из базы данных произошла ошибка:\n%v", cmdText), gCtx)

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
func respondJSON(message message, g *gin.Context) {
	msgJSON, _ := json.Marshal(message)
	//string(msgJSON)
	g.JSON(http.StatusOK, string(msgJSON))
}
func sendMsg(g *gin.Context, msg string, s ...interface{}) {
	respondJSON(message{Text: fmt.Sprintf(msg, s...)}, g)
}
