package handler

import (
	"fmt"
	"net/http"

	"strconv"

	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/coyotes/brokers"
	broker "github.com/mylxsw/coyotes/brokers/redis"
	"github.com/mylxsw/coyotes/config"
	"github.com/mylxsw/coyotes/http/response"
	"github.com/mylxsw/coyotes/log"
)

// RemoveTask Remove specified task from task queue
func RemoveTask(w http.ResponseWriter, r *http.Request) {

	// vars := mux.Vars(r)
	// channelName := vars["channel_name"]
	// taskName := vars["task_id"]

}

// PushTask 添加任务到任务队列
// 参数：
//    id           任务ID，可以不指定，系统自动生成
//    task         任务名称，不能为空，用于唯一标识一个任务
//    channel_name 任务执行channel，标识任务在哪个channel中执行，如果不指定则在默认的channel中执行
//    delay        如果需要延迟执行，这里指定延迟的秒数，0为不延迟执行
//    command      要执行的命令
func PushTask(w http.ResponseWriter, r *http.Request) {

	taskName := r.PostFormValue("task")
	taskChannel := mux.Vars(r)["channel_name"]
	delaySec, _ := strconv.Atoi(r.PostFormValue("delay"))
	commandName := r.PostFormValue("command")
	id := r.PostFormValue("id")

	var args []string
	for key, values := range r.PostForm {
		if key != "args" {
			continue
		}

		args = append(args, values...)
	}

	if taskName == "" {
		w.Write(response.Failed("任务名称不能为空"))
		return
	}

	if taskChannel == "" {
		taskChannel = config.GetRuntime().Config.DefaultChannel
	}

	if _, ok := config.GetRuntime().Channels[taskChannel]; !ok {
		w.Write(response.Failed("channel不存在"))
		return
	}

	var taskID string
	var err error
	var existence bool

	task := brokers.Task{
		ID:           id,
		TaskName:     taskName,
		WriteBackend: true,
		Channel:      taskChannel,
		Command: brokers.TaskCommand{
			Name: commandName,
			Args: func() []interface{} {
				res := make([]interface{}, len(args))
				for i, s := range args {
					res[i] = s
				}

				return res
			}(),
		},
	}

	if delaySec != 0 {
		taskID, existence, err = broker.GetTaskManager().AddDelayTask(
			time.Now().Add(time.Duration(delaySec)*time.Second),
			task,
		)
	} else {
		taskID, existence, err = broker.GetTaskManager().AddTask(task)
	}

	if err != nil {
		message := fmt.Sprintf("failed push task [%s] to redis queue [%s]: %v", taskName, taskChannel, err)
		log.Error(message)
		w.Write(response.Failed(message))
		return
	}

	w.Write(response.Success(struct {
		TaskID   string `json:"task_id"`
		TaskName string `json:"task_name"`
		Result   bool   `json:"result"`
	}{
		TaskID:   taskID,
		TaskName: taskName,
		Result:   !existence,
	}))
}
