package projects

// TaskHandler executes a task and returns its result.
type TaskHandler func(tc TaskContext) *TaskResult

// TaskHandlerRegistry maps handler names to task executors.
type TaskHandlerRegistry map[string]TaskHandler

var globalTaskHandlers = make(TaskHandlerRegistry)

// RegisterTaskHandler registers a named task handler.
func RegisterTaskHandler(name string, handler TaskHandler) {
	globalTaskHandlers[name] = handler
}

// GetTaskHandler returns a registered task handler by name.
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
	RegisterTaskHandler("tmpl", runTpl)
	RegisterTaskHandler("docker", runDockerTask)
	RegisterTaskHandler("cast", runCastCrossProjectTask)
}
