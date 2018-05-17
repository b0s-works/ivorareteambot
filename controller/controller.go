package controller

import (
	"net/http"
	"fmt"
	"log"
	"strconv"
	"github.com/gin-gonic/gin"
	"ivorareteambot/app"
	"ivorareteambot/types"
	"runtime"
)

type Controller struct {
	app        app.Application
	gin        *gin.Engine
	slackToken string
}

func GetFunctionName() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return fmt.Sprintf("%s:%d", frame.Line, frame.Function)
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
	{
		slack.POST("/", func(gCtx *gin.Context) {
			log.Println(gCtx.Request.URL.Path, gCtx)
		})
		slack.POST("/tbb_sethours", func(gCtx *gin.Context) {
			cmdText := gCtx.Params.ByName("text")
			log.Println("cmdText:", cmdText)

			curTsk, err := c.app.GetCurrentTask()
			c.checkDBError(err, "При запросе текущей задачи произошла ошибка", gCtx)

			if curTsk.Title == "" {
				gCtx.JSON(http.StatusOK,
					c.makeAMessage(
						"Зайдайте «Название задачи» для которой хотите провести командную оценку времени",
						[]string{"Задать «Название задачи» можно с помощью команды «/tbb_settask»"},
					),
				)
				return
			}
			fmt.Println(gCtx)

			hoursBid, err := strconv.ParseInt(cmdText, 10, 64)
			if err != nil {
				gCtx.JSON(
					http.StatusOK,
					types.Message{Text: fmt.Sprintf("Пожалуйста, укажите целое число! (вы ввели: «%s»)", cmdText)},
				)
				return
			}

			UserIdentity, _ := gCtx.GetPostForm("user_id")
			UserName, _ := gCtx.GetPostForm("user_name")
			if UserIdentity == "" || UserName == "" {
				gCtx.JSON(
					http.StatusOK,
					types.Message{Text: fmt.Sprintf("Поле хранящее идентификатор пользователя или имя пользователя пусто.")},
				)
				return
			}

			taskHoursBidAndMember, err := c.app.GetUserBidByTaskIDAndUserIdentity(curTsk.ID, UserIdentity)
			c.checkDBError(err, "При запросе из базы данных, оценки времени пользователя, произошла ошибка", gCtx)

			oldBid := taskHoursBidAndMember.MemberTimeBid

			taskHoursBidAndMember.MemberTimeBid = hoursBid
			taskHoursBidAndMember.MemberIdentity = UserIdentity
			taskHoursBidAndMember.MemberNick = UserName


			rowsAffected, err := c.app.SetHours(hoursBid, taskHoursBidAndMember)
			c.checkDBError(err, "При сохранении, в базу данных, часовой оценки пользователя, произошла ошибка", gCtx)

			if rowsAffected > 1 || rowsAffected == 0 {
				gCtx.JSON(
					http.StatusOK,
					types.Message{
						Text:fmt.Sprint(
							"При сохранении, в базу данных, часовой оценки пользователя, произошла неизвестная неполадка. Количество изменённых записей равно %v.",
							rowsAffected,
						),
					},
				)
				return
			}
			if oldBid > 0 {
				gCtx.JSON(
					http.StatusOK,
					types.Message{Text: fmt.Sprintf("Ваша оценка для задачи «%s» изменена с %v на %v\nСпасибо!", curTsk.Title, oldBid, hoursBid)},
				)
				return
			}
			gCtx.JSON(
				http.StatusOK,
				types.Message{Text: fmt.Sprintf("Ваша оценка для задачи «%s» равная %v, записана.\nСпасибо!", curTsk.Title, hoursBid)},
			)
		})
		slack.POST("/tbb_setbidtask", func(gCtx *gin.Context) {
			cmdText := gCtx.Params.ByName("text")
			log.Println("cmdText:", cmdText)

			if cmdText == "" {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)})
				return
			}
			task, err := c.app.GetTask(cmdText);
			c.checkDBError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%v", cmdText), gCtx)

			log.Printf("First or create task result:\n%+v", task)

			if task.BiddingDone != 0 {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Ставки времени для задачи «%s» уже сделаны! Голосование закрыто.", cmdText)})
				return
			}

			err = c.app.SetTask(task.ID);
			c.checkDBError(err, fmt.Sprintf("При выборе задачи «%s» произошла ошибка:\n%v", cmdText), gCtx)

			gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Задача «%s» выдвинута для совершения ставок оценки времени.", cmdText)})
			return

		})
		slack.POST("/tbb_listtaskbids", func(gCtx *gin.Context) {
			cmdText := gCtx.Params.ByName("text")
			log.Println("cmdText:", cmdText)

			if cmdText == "" {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)})
				return
			}
			var task, err = c.app.GetTask(cmdText);
			c.checkDBError(err, "Произошла ошибка при обращении к базе данных", gCtx)

			if task.ID == 0 {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Задача с точным названием «%s» не найдена.", cmdText)})
				return
			}
			membersAndBids, err := c.app.GetTaskHoursBids(task.ID);
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
		slack.POST("/tbb_list", func(gCtx *gin.Context) {
			// Unfound
			tasks, err := c.app.GetAllTasks()
			c.checkDBError(err, "При запросе всех задач в базу данных, произошла ошибка", gCtx)

			if len(tasks) > 0 {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Список задач пуст.")})
				return
			}
			var TaskBiddingDoneString string
			for _, task := range tasks {
				TaskBiddingDoneString = "открыто"
				if task.BiddingDone > 0 {
					TaskBiddingDoneString = "совершено"
				}
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("%v. «%s» (голосование %s)", task.ID, task.Title, TaskBiddingDoneString)})
			}
		})
		slack.POST("/tbb_removetask", func(gCtx *gin.Context) {
			cmdText := gCtx.Params.ByName("text")
			log.Println("cmdText:", cmdText)

			/*curTsk, err := c.app.GetCurrentTask(); c.checkDBError(err, "При запросе текущей задачи произошла ошибка", gCtx)*/

			if cmdText == "" {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Укажите Название задачи например:\n«%s Название задачи».", gCtx.Request.URL.Path)})
				return
			}
			// Unfound
			task, err := c.app.GetTask(cmdText);
			c.checkDBError(err, fmt.Sprintf("При поиске задачи «%s» в базе данных произошла ошибка:\n%v", cmdText), gCtx)

			if task.ID == 0 {
				gCtx.JSON(http.StatusOK, types.Message{Text: fmt.Sprintf("Задача «%s» не найдена.", cmdText)})
				return
			}

			rowsAffected, childRowsAffected, err := c.app.RemoveTaskByIdAndChildHours(task.ID);
			c.checkDBError(err, fmt.Sprintf("При удалении задачи «%s» из базы данных произошла ошибка:\n%v", cmdText), gCtx)

			var message string
			var additionalMessages []string
			if rowsAffected == 1 {
				message = fmt.Sprintf("Задача «%s» удалена.", cmdText)
				if childRowsAffected > 0 {
					additionalMessages = append(additionalMessages, "Так же удалены все связанные с ней ставки участников.")
				}
				currentTask, err := c.app.GetCurrentTask();
				c.checkDBError(err, "При запросе текущей задачи голосованияиз базы данных, произошла ошибка", gCtx)
				if currentTask.ID == task.ID {
					err := c.app.SetTask(0)
					c.checkDBError(err, "При выполнении запроса в базу данных для обнуления текущей задачи, произошла ошибка", gCtx)
					additionalMessages = append(additionalMessages, fmt.Sprintf("Задача «%s» так же снята с голосования.", cmdText))
				}

				gCtx.JSON(
					http.StatusOK,
					c.makeAMessage(message, additionalMessages),
				)
			}
		})
	}
}

func (c Controller) checkDBError(err error, prefixMsg string, gCtx *gin.Context) {
	if err != nil {
		var msg = types.Message{Text: fmt.Sprintf("%s:\n%v", prefixMsg, err)}
		gCtx.JSON(http.StatusInternalServerError, msg)
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

func (c Controller) slackAuth(gCtx *gin.Context) {
	form := gCtx.Request.MultipartForm
	if t := form.Value["token"]; len(t) == 0 || t[0] != c.slackToken {
		//TODO respond error
		gCtx.Abort()
		return
	}

	gCtx.Next()
}