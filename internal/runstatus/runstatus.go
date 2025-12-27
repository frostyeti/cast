package runstatus

const (
	None      = 0
	Running   = 1
	Ok        = 2
	Error     = 3
	Skipped   = 4
	Cancelled = 5
)

func ToString(status int) string {
	switch status {
	case None:
		return "none"
	case Running:
		return "running"
	case Ok:
		return "ok"
	case Error:
		return "error"
	case Skipped:
		return "skipped"
	case Cancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func FromString(status string) int {
	switch status {
	case "none":
		return None
	case "running":
		return Running
	case "ok":
		return Ok
	case "error":
		return Error
	case "skipped":
		return Skipped
	case "cancelled":
		return Cancelled
	default:
		return None
	}
}
