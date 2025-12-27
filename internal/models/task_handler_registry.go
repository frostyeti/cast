package models

type TaskHandler func(tc TaskContext) *TaskResult

type TaskHandlerRegistry map[string]TaskHandler

func (thr TaskHandlerRegistry) Register(name string, handler TaskHandler) {
	thr[name] = handler
}

func (thr TaskHandlerRegistry) Get(name string) (TaskHandler, bool) {
	handler, exists := thr[name]
	return handler, exists
}

var GlobalTaskHandlerRegistry = make(TaskHandlerRegistry)
