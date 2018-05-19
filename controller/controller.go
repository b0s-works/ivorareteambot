package controller

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"ivorareteambot/app"
	"ivorareteambot/types"

	"github.com/gin-gonic/gin"
)

// Controller main controller structure
type Controller struct {
	app        app.Application
	gin        *gin.Engine
	slackToken string
}

// New controller constructor
func New(app app.Application, slackToken string) Controller {
	return Controller{
		app:        app,
		gin:        gin.Default(),
		slackToken: slackToken,
	}
}

// Serve - start listener
func (c Controller) Serve() {
	http.Handle("/", c.gin)
}

// InitRouters initialize all handlers and routes
func (c Controller) InitRouters() {

	slack := c.gin.Group("/slack")
	slack.Use(c.slackAuth)
	{
		slack.POST("/", func(ctx *gin.Context) {
			log.Println(ctx.Request.URL.Path, ctx)
		})

		slack.POST("/sethours", c.setHours)

		slack.POST("/setbidtask", c.setBidTask)

		slack.POST("/listtaskbids", func(gCtx *gin.Context) {
			cmdText := gCtx.Params.ByName("text")
			log.Println("cmdText:", cmdText)

			if cmdText == "" {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)})
				return
			}
			var task, err = c.app.GetTask(cmdText)
			c.checkDBError(err, "Произошла ошибка при обращении к базе данных", gCtx)

			if task.ID == 0 {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Задача с точным названием «%s» не найдена.", cmdText)})
				return
			}
			membersAndBids, err := c.app.GetTaskHoursBids(task.ID)
			c.checkDBError(err, "Произошла ошибка при обращении к базе данных", gCtx)

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
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Для задачи «%s» участниками были сделаны следующие ставки:\n%s", cmdText, resultMembersAndBidsList)})
				return
			} else {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Ставок времени для задачи «%s» не сделано.", cmdText)})
				return
			}
			fmt.Println(task)
		})

		slack.POST("/tbb_list", func(ctx *gin.Context) {
			// Unfound
			tasks, err := c.app.GetAllTasks()
			c.checkDBError(err, "При запросе всех задач в базу данных, произошла ошибка", ctx)

			if len(tasks) > 0 {
				ctx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Список задач пуст.")})
				return
			}
			var TaskBiddingDoneString string
			for _, task := range tasks {
				TaskBiddingDoneString = "открыто"
				if task.BiddingDone > 0 {
					TaskBiddingDoneString = "совершено"
				}
				ctx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("%v. «%s» (голосование %s)", task.ID, task.Title, TaskBiddingDoneString)})
			}
		})
		slack.POST("/tbb_removetask", func(ctx *gin.Context) {
			taskTitle := ctx.Params.ByName("text")
			log.Println("taskTitle:", taskTitle)

			/*curTsk, err := c.app.GetCurrentTask(); c.checkDBError(err, "При запросе текущей задачи произошла ошибка", ctx)*/

			if taskTitle == "" {
				ctx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", ctx.Request.URL.Path)})
				return
			}
			// Unfound
			task, err := c.app.GetTask(taskTitle)
			c.checkDBError(err, fmt.Sprintf("При поиске задачи «%s» в базе данных произошла ошибка:\n%v", taskTitle), ctx)

			if task.ID == 0 {
				ctx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Задача «%s» не найдена.", taskTitle)})
				return
			}

			childRowsAffected, err := c.app.RemoveTaskByIdAndChildHours(task.ID)
			c.checkDBError(err, fmt.Sprintf("При удалении задачи «%s» из базы данных произошла ошибка:\n%v", taskTitle), ctx)

			response := types.NewMessage(fmt.Sprintf("Задача «%s» удалена.", taskTitle))

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
		})
	}
}

func (c Controller) checkDBError(err error, prefixMsg string, ctx *gin.Context) {
	if err != nil {
		var msg = types.Message{Text: fmt.Sprintf("%s:\n%v", prefixMsg, err)}
		ctx.JSON(http.StatusInternalServerError, msg)
		panic(msg)
		return
	}
}

func (c Controller) makeAMessage(mainMsg string, supportingMessages []string) types.Message {
	var msg types.Message

	msg.Text = mainMsg

	if len(supportingMessages) > 0 {
		for _, sprtngMsg := range supportingMessages {
			msg.Attachments = append(msg.Attachments, types.Attachment{Text: sprtngMsg})
		}
	}
	return msg
}

func (c Controller) slackAuth(ctx *gin.Context) {
	form := ctx.Request.MultipartForm
	if t := form.Value["token"]; len(t) == 0 || t[0] != c.slackToken {
		//TODO respond error
		ctx.Abort()
		return
	}

	ctx.Next()
}

func (c Controller) setHours(ctx *gin.Context) {
	cmdText := ctx.Params.ByName("text")
	log.Println("cmdText:", cmdText)

	curTsk, err := c.app.GetCurrentTask()
	if err != nil {

	}
	c.checkDBError(err, "При запросе текущей задачи произошла ошибка", ctx)

	if curTsk.Title == "" {
		ctx.JSON(http.StatusOK,
			c.makeAMessage(
				"Зайдайте «Название задачи» для которой хотите провести командную оценку времени",
				[]string{"Задать «Название задачи» можно с помощью команды «/tbb_settask»"},
			),
		)
		return
	}
	fmt.Println(ctx)

	hoursBid, err := strconv.ParseInt(cmdText, 10, 64)
	if err != nil {
		ctx.JSON(
			http.StatusOK,
			types.Message{Text: fmt.Sprintf("Пожалуйста, укажите целое число! (вы ввели: «%s»)", cmdText)},
		)
		return
	}

	UserIdentity, _ := ctx.GetPostForm("user_id")
	UserName, _ := ctx.GetPostForm("user_name")
	if UserIdentity == "" || UserName == "" {
		ctx.JSON(
			http.StatusOK,
			types.Message{Text: fmt.Sprintf("Поле хранящее идентификатор пользователя или имя пользователя пусто.")},
		)
		return
	}

	taskHoursBidAndMember, err := c.app.GetUserBidByTaskIDAndUserIdentity(curTsk.ID, UserIdentity)
	c.checkDBError(err, "При запросе из базы данных, оценки времени пользователя, произошла ошибка", ctx)

	oldBid := taskHoursBidAndMember.MemberTimeBid

	taskHoursBidAndMember.MemberTimeBid = hoursBid
	taskHoursBidAndMember.MemberIdentity = UserIdentity
	taskHoursBidAndMember.MemberNick = UserName

	rowsAffected, err := c.app.SetHours(hoursBid, taskHoursBidAndMember)
	c.checkDBError(err, "При сохранении, в базу данных, часовой оценки пользователя, произошла ошибка", ctx)

	if rowsAffected > 1 || rowsAffected == 0 {
		ctx.JSON(
			http.StatusOK,
			types.Message{
				Text: fmt.Sprint(
					"При сохранении, в базу данных, часовой оценки пользователя, произошла неизвестная неполадка. Количество изменённых записей равно %v.",
					rowsAffected,
				),
			},
		)
		return
	}
	if oldBid > 0 {
		ctx.JSON(
			http.StatusOK,
			types.Message{Text: fmt.Sprintf("Ваша оценка для задачи «%s» изменена с %v на %v\nСпасибо!", curTsk.Title, oldBid, hoursBid)},
		)
		return
	}
	ctx.JSON(
		http.StatusOK,
		types.Message{Text: fmt.Sprintf("Ваша оценка для задачи «%s» равная %v, записана.\nСпасибо!", curTsk.Title, hoursBid)},
	)
}

func (c Controller) setBidTask(gCtx *gin.Context) {
	cmdText := gCtx.Params.ByName("text")
	log.Println("cmdText:", cmdText)

	if cmdText == "" {
		gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)})
		return
	}
	task, err := c.app.GetTask(cmdText)
	c.checkDBError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%v", cmdText), gCtx)

	log.Printf("First or create task result:\n%+v", task)

	if task.BiddingDone != 0 {
		gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Ставки времени для задачи «%s» уже сделаны! Голосование закрыто.", cmdText)})
		return
	}

	err = c.app.SetTask(task.ID)
	c.checkDBError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%v", cmdText), gCtx)

	gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Задача «%s» выдвинута для совершения ставок оценки времени.", cmdText)})
	return

}
