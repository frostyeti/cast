package projects

type TaskHandler func(tc TaskContext) *TaskResult

type TaskHandlerRegistry map[string]TaskHandler

var globalTaskHandlers = make(TaskHandlerRegistry)

func RegisterTaskHandler(name string, handler TaskHandler) {
	globalTaskHandlers[name] = handler
}

func GetTaskHandler(name string) (TaskHandler, bool) {
	handler, ok := globalTaskHandlers[name]
	return handler, ok
}

func init() {
	RegisterTaskHandler("ssh", runSshTask)
	RegisterTaskHandler("scp", runScpTask)
	RegisterTaskHandler("bash", runShell)
	RegisterTaskHandler("sh", runShell)
	RegisterTaskHandler("pwsh", runShell)
	RegisterTaskHandler("powershell", runShell)
	RegisterTaskHandler("shell", runShell)
	RegisterTaskHandler("go", runShell)
	RegisterTaskHandler("golang", runShell)
	RegisterTaskHandler("dotnet", runShell)
	RegisterTaskHandler("csharp", runShell)
	RegisterTaskHandler("deno", runShell)
	RegisterTaskHandler("node", runShell)
	RegisterTaskHandler("bun", runShell)
	RegisterTaskHandler("python", runShell)
	RegisterTaskHandler("ruby", runShell)
}
