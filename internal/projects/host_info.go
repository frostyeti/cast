package projects

import "github.com/frostyeti/cast/internal/types"

type HostInfo struct {
	Host         string
	Port         uint
	User         string
	Password     string
	IdentityFile string
	Tags         []string
	Meta         types.Meta
	OS           types.OsInfo
}
