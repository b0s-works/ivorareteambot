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
	"bytes"
	"io/ioutil"
)

// Controller main controller structure
type Controller struct {
	app app.Application
	//TODO May be we have to use another way of c.Gin accessing for listenAndServe in main.go? Probably that can be wrong when you make c.Gin variable globally accessible?
	Gin           *gin.Engine
	slackInToken  string
	slackOutToken string
}

// New controller constructor
func New(app app.Application, slackInToken string, slackOutToken string) Controller {
	return Controller{
		app:           app,
		Gin:           gin.Default(),
		slackInToken:  slackInToken,
		slackOutToken: slackOutToken,
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
		ctx.Request.ParseForm()
		for key, value := range ctx.Request.PostForm {
			logrus.Println(key, value[0])
		}
		logrus.Error("request given", ctx.Request.Body)
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
		logrus.Error(msg)
		ctx.JSON(http.StatusInternalServerError, msg)
	}
	return
}

func (c Controller) slackAuth(ctx *gin.Context) {
	token := ctx.Request.FormValue("token")
	if len(token) == 0 || token != c.slackInToken {
		ctx.JSON(
			http.StatusOK,
			newMessage("Security Slack token that came from Slack are empty or wrong."),
		)
		logrus.Println(fmt.Sprintf("SlackToken that came from Slack are empty or wrong. Token that came is:\n\t\t\t«%v»", token))
		ctx.Abort()
		return
	}

	ctx.Next()
}

func (c Controller) setHours(ctx *gin.Context) {
	taskTitle := ctx.Request.FormValue("text")
	log.Println("taskTitle:", taskTitle)

	curTsk, err := c.app.GetCurrentTask()
	checkError(err, "Error occurred while current task request.", ctx)

	if curTsk.Title == "" {
		response := newMessage("Please, specify the «Task title», that will be set to hours voting.").
			AddAttachment("«Task title» can be specified by «/tbb_settask» command.")
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
	checkError(err, "Error occurred when hours requested from db", ctx)

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

func (c Controller) setTask(ctx *gin.Context) {
	taskTitle := ctx.Request.FormValue("text")
	log.Println("taskTitle:", taskTitle)

	if taskTitle == "" {
		ctx.JSON(http.StatusOK, newMessage("Please, specify «Task title».").
			AddAttachment(fmt.Sprintf("«Task title» can be specified by «%[1]s» command. For example:\n%[1]s Task title.", "/tbb_settask")))
		return
	}
	task, err := c.app.GetTask(taskTitle)
	checkError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%+v", taskTitle, err), ctx)

	log.Printf("First or create task result:\n%+v", task)

	if task.BiddingDone != 0 {
		ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Ставки времени для задачи «%s» уже сделаны! Голосование закрыто.", taskTitle)))
		return
	}

	err = c.app.SetTask(task.ID)
	checkError(err, fmt.Sprintf("Error occurred on «%s» task set:\n%+v", taskTitle, err), ctx)
	var url = "https://slack.com/api/chat.postMessage"
	var msg = fmt.Sprintf("Задача «%s» выдвинута для совершения ставок оценки времени.", taskTitle)

	//logrus.Println( string(jsonReqData) )

	var jsonReqData = []byte(fmt.Sprintf(`{
		"channel": "%s",
		"text":  "%s"
	}`,
		ctx.Request.FormValue("channel_id"),
		msg,
	))
	logrus.Println( string(jsonReqData) )

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonReqData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.slackOutToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Задача «%s» выдвинута для совершения ставок оценки времени.", taskTitle)))
	return

}

func (c Controller) listTaskHours(ctx *gin.Context) {
	taskTitle := ctx.Request.FormValue("text")
	log.Println("cmdText:", taskTitle)

	if taskTitle == "" {
		ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", ctx.Request.URL.Path)))
		return
	}
	var task, err = c.app.GetTask(taskTitle)
	checkError(err, "Произошла ошибка при обращении к базе данных", ctx)

	if task.ID == 0 {
		ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Задача с точным названием «%s» не найдена.", taskTitle)))
		return
	}
	membersAndBids, err := c.app.GetTaskHoursBids(task.ID)
	checkError(err, "Произошла ошибка при обращении к базе данных", ctx)

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
		ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Для задачи «%s» участниками были сделаны следующие ставки:\n%s", taskTitle, resultMembersAndBidsList)))
		return
	}

	fmt.Println(task)

	ctx.JSON(http.StatusOK, newMessage(fmt.Sprintf("Ставок времени для задачи «%s» не сделано.", taskTitle)))
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
	taskTitle := ctx.Request.FormValue("text")
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
