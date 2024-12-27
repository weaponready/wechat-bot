package utils

import (
	"fmt"
	"sync"
	"time"
)

// 模拟 openwechat.Message 类型
type Message struct {
	Content string
}

// Task represents a single reminder task
type Task struct {
	ID       int
	Message  string
	When     time.Time
	Callback func() // 回调函数，支持接收 *Message 参数
}

// TaskManager manages all tasks
type TaskManager struct {
	tasks map[int]*Task
	mu    sync.Mutex
	idGen int
}

// NewTaskManager creates a new TaskManager
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[int]*Task),
	}
}

// AddTask adds a new task and schedules it
func (tm *TaskManager) AddTask(msg *Message, delay time.Duration, callback func()) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.idGen++
	taskID := tm.idGen
	task := &Task{
		ID:       taskID,
		Message:  msg.Content,
		When:     time.Now().Add(delay),
		Callback: callback,
	}

	tm.tasks[taskID] = task

	// Schedule the task
	go func(t *Task) {
		time.Sleep(delay)
		tm.triggerTask(t.ID)
	}(task)

	fmt.Printf("任务已添加: [ID: %d] %s (将在 %v 提醒)\n", taskID, task.Message, task.When)
}

// triggerTask triggers a task and removes it from the list
func (tm *TaskManager) triggerTask(taskID int) {
	tm.mu.Lock()
	task, exists := tm.tasks[taskID]
	if exists {
		delete(tm.tasks, taskID)
	}
	tm.mu.Unlock()

	if exists {
		fmt.Printf("提醒: %s (任务ID: %d)\n", task.Message, task.ID)
		if task.Callback != nil {
			task.Callback() // 执行回调函数，传递参数
		}
	}
}

// ListTasks lists all pending tasks
func (tm *TaskManager) ListTasks() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if len(tm.tasks) == 0 {
		fmt.Println("当前没有任务")
		return
	}

	fmt.Println("当前任务列表:")
	for id, task := range tm.tasks {
		fmt.Printf("[ID: %d] %s (将在 %v 提醒)\n", id, task.Message, task.When)
	}
}
