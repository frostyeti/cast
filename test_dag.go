package main

import (
	"fmt"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/cast/internal/projects"
)

func main() {
    p := &projects.Project{
        Schema: types.Project{
            Jobs: types.NewJobMap(),
        },
    }
    
    // Add jobs
    jA := types.NewJob()
    jA.Id = "jobA"
    
    jB := types.NewJob()
    jB.Id = "jobB"
    needsB := types.Needs{types.Need{Id: "jobA"}}
    jB.Needs = &needsB
    
    jC := types.NewJob()
    jC.Id = "jobC"
    needsC := types.Needs{types.Need{Id: "jobB"}}
    jC.Needs = &needsC
    
    jD := types.NewJob()
    jD.Id = "jobD"
    
    p.Schema.Jobs.Set(jA)
    p.Schema.Jobs.Set(jB)
    p.Schema.Jobs.Set(jC)
    p.Schema.Jobs.Set(jD)
    
    res, err := p.GetDownstreamJobs("jobA")
    fmt.Printf("Downstream of jobA: %v, err: %v\n", res, err)
}
