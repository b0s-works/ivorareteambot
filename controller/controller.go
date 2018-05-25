package controller

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"ivorareteambot/app"
	"ivorareteambot/types"

	"github.com/sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

// Controller main controller structure
type Controller struct {
	app        app.Application
	//TODO May be we have to use another way of c.Gin accessing for listenAndServe in main.go? Probably that can be wrong when you make c.Gin variable globally accessible?
	Gin        *gin.Engine
	slackToken string
}

// New controller constructor
func New(app app.Application, slackToken string) Controller {
	return Controller{
		app:        app,
		Gin:        gin.Default(),
		slackToken: slackToken,
	}
}

// Serve - start HTTP requests listener
func (c Controller) Serve() {
	http.Handle("/", c.Gin)
}

func myLog() {

}

// InitRouters initialize all handlers and routes
func (c Controller) InitRouters() {
	c.Gin.Use(func(ctx *gin.Context) {
		logrus.Error("request given")
	})
	slack := c.Gin.Group("/slack")
	slack.Use(c.slackAuth)
	{
		slack.POST("/", func(ctx *gin.Context) {
			log.Println(ctx.Request.URL.Path, ctx)
		})

		slack.POST("/sethours", c.setHours)

		slack.POST("/settask", c.setTask)

		slack.POST("/listtaskhours", c.listTaskHours)

		slack.POST("/listalltasks", c.listAllTasks)

		slack.POST("/removetask", c.removeTask)
	}
}

func newMessage(mainMessage string) *types.Message {
	return types.NewMessage(mainMessage)
}

func checkError(err error, prefixMsg string, ctx *gin.Context) {
	if err != nil {
		var msg = newMessage(fmt.Sprintf("%s:\n%+v", prefixMsg, err))
		ctx.JSON(http.StatusInternalServerError, msg)
	}
	return
}

func (c Controller) slackAuth(ctx *gin.Context) {
	token := ctx.Request.FormValue("token")
	if len(token) == 0 || token != c.slackToken {
		log.Println(fmt.Sprintf("SlackToken that came from Slack are empty or wrong. Token that came is:\n\t\t\t«%v»", token))
		ctx.Abort()
		return
	}

	ctx.Next()
}

func (c Controller) setHours(ctx *gin.Context) {
	taskTitle := ctx.Params.ByName("text")
	log.Println("taskTitle:", taskTitle)

	curTsk, err := c.app.GetCurrentTask()
	checkError(err, "При запросе текущей задачи произошла ошибка", ctx)

	if curTsk.Title == "" {
		response := newMessage("Зайдайте «Название задачи» для которой хотите провести командную оценку времени").
			AddAttachment("Задать «Название задачи» можно с помощью команды «/tbb_settask»")
		ctx.JSON(http.StatusOK, response)
		return
	}
	fmt.Println(ctx)

	hoursBid, err := strconv.ParseInt(taskTitle, 10, 64)
	if err != nil {
		ctx.JSON(
			http.StatusOK,
			newMessage(fmt.Sprintf("Пожалуйста, укажите целое число! (вы ввели: «%s»)", taskTitle)),
		)
		return
	}

	UserIdentity, _ := ctx.GetPostForm("user_id")
	UserName, _ := ctx.GetPostForm("user_name")
	if UserIdentity == "" || UserName == "" {
		ctx.JSON(
			http.StatusOK,
			newMessage(fmt.Sprintf("Поле хранящее идентификатор пользователя или имя пользователя пусто.")),
		)
		return
	}

	taskHoursBidAndMember, err := c.app.GetUserBidByTaskIDAndUserIdentity(curTsk.ID, UserIdentity)
	checkError(err, "При запросе из базы данных, оценки времени пользователя, произошла ошибка", ctx)

	oldBid := taskHoursBidAndMember.MemberTimeBid

	taskHoursBidAndMember.MemberTimeBid = hoursBid
	taskHoursBidAndMember.MemberIdentity = UserIdentity
	taskHoursBidAndMember.MemberNick = UserName

	rowsAffected, err := c.app.SetHours(hoursBid, taskHoursBidAndMember)
	checkError(err, "При сохранении, в базу данных, часовой оценки пользователя, произошла ошибка", ctx)

	if rowsAffected > 1 || rowsAffected == 0 {
		ctx.JSON(
			http.StatusOK,
			newMessage(
				fmt.Sprintf(
					"При сохранении, в базу данных, часовой оценки пользователя, произошла неизвестная неполадка. Количество изменённых записей равно %+v.",
					rowsAffected,
				),
			),
		)
		return
	}
	if oldBid > 0 {
		ctx.JSON(
			http.StatusOK,
			newMessage(fmt.Sprintf("Ваша оценка для задачи «%s» изменена с %v на %v\nСпасибо!", curTsk.Title, oldBid, hoursBid)),
		)
		return
	}
	ctx.JSON(
		http.StatusOK,
		newMessage(fmt.Sprintf("Ваша оценка для задачи «%s» равная %v, записана.\nСпасибо!", curTsk.Title, hoursBid)),
	)
}

func (c Controller) setTask(gCtx *gin.Context) {
	taskTitle := gCtx.Params.ByName("text")
	log.Println("taskTitle:", taskTitle)

	if taskTitle == "" {
		gCtx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)))
		return
	}
	task, err := c.app.GetTask(taskTitle)
	checkError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%+v", taskTitle, err), gCtx)

	log.Printf("First or create task result:\n%+v", task)

	if task.BiddingDone != 0 {
		gCtx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Ставки времени для задачи «%s» уже сделаны! Голосование закрыто.", taskTitle)))
		return
	}

	err = c.app.SetTask(task.ID)
	checkError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%+v", taskTitle, err), gCtx)

	gCtx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Задача «%s» выдвинута для совершения ставок оценки времени.", taskTitle)))
	return

}

func (c Controller) listTaskHours(gCtx *gin.Context) {
	taskTitle := gCtx.Params.ByName("text")
	log.Println("cmdText:", taskTitle)

	if taskTitle == "" {
		gCtx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)))
		return
	}
	var task, err = c.app.GetTask(taskTitle)
	checkError(err, "Произошла ошибка при обращении к базе данных", gCtx)

	if task.ID == 0 {
		gCtx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Задача с точным названием «%s» не найдена.", taskTitle)))
		return
	}
	membersAndBids, err := c.app.GetTaskHoursBids(task.ID)
	checkError(err, "Произошла ошибка при обращении к базе данных", gCtx)

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
		gCtx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Для задачи «%s» участниками были сделаны следующие ставки:\n%s", taskTitle, resultMembersAndBidsList)))
		return
	}

	fmt.Println(task)

	gCtx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Ставок времени для задачи «%s» не сделано.", taskTitle)))
	return
}

func (c Controller) listAllTasks(ctx *gin.Context) {
	// Unfound
	tasks, err := c.app.GetAllTasks()
	checkError(err, "При запросе всех задач в базу данных, произошла ошибка", ctx)

	if len(tasks) > 0 {
		ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Список задач пуст.")))
		return
	}
	var TaskBiddingDoneString string
	for _, task := range tasks {
		TaskBiddingDoneString = "открыто"
		if task.BiddingDone > 0 {
			TaskBiddingDoneString = "совершено"
		}
		ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("%v. «%s» (голосование %s)", task.ID, task.Title, TaskBiddingDoneString)))
	}
}

func (c Controller) removeTask(ctx *gin.Context) {
	taskTitle := ctx.Params.ByName("text")
	log.Println("taskTitle:", taskTitle)

	/*curTsk, err := c.app.GetCurrentTask(); checkError(err, "При запросе текущей задачи произошла ошибка", ctx)*/

	if taskTitle == "" {
		ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", ctx.Request.URL.Path)))
		return
	}
	// Unfound
	task, err := c.app.GetTask(taskTitle)
	checkError(err, fmt.Sprintf("При поиске задачи «%s» в базе данных произошла ошибка:\n%v", taskTitle, err), ctx)

	if task.ID == 0 {
		ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Задача «%s» не найдена.", taskTitle)))
		return
	}

	childRowsAffected, err := c.app.RemoveTaskByIdAndChildHours(task.ID)
	checkError(err, fmt.Sprintf("При удалении задачи «%s» из базы данных произошла ошибка:\n%v", taskTitle, err), ctx)

	response := newMessage(fmt.Sprintf("Задача «%s» удалена.", taskTitle))

	if childRowsAffected > 0 {
		response.AddAttachment("Так же удалены все связанные с ней ставки участников.")
	}
	currentTask, err := c.app.GetCurrentTask()
	if err != nil {
		c.respondMessage(ctx, "При запросе текущей задачи голосованияиз базы данных, произошла ошибка")
	}

	if currentTask.ID == task.ID {
		if err := c.app.SetTask(0); err != nil {
			c.respondMessage(ctx, "При выполнении запроса в базу данных для обнуления текущей задачи, произошла ошибка")
			return
		}
		response.AddAttachment(fmt.Sprintf("Задача «%s» так же снята с голосования.", taskTitle))
	}

	ctx.JSON(http.StatusOK, response)
}
