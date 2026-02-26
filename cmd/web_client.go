package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

func getBaseURL(cmd *cobra.Command) string {
	addr, _ := cmd.Flags().GetString("addr")
	port, _ := cmd.Flags().GetInt("port")
	return fmt.Sprintf("http://%s:%d", addr, port)
}

func printJSON(data []byte) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(prettyJSON.String())
}

func makeRequest(cmd *cobra.Command, method, path string) {
	url := getBaseURL(cmd) + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	printJSON(body)
}

var webProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects via API",
}

var webProjectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Run: func(cmd *cobra.Command, args []string) {
		makeRequest(cmd, "GET", "/api/v1/projects")
	},
}

var webJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage jobs via API",
}

var webJobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs for a project",
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			fmt.Println("Error: --project/-P flag is required")
			return
		}
		makeRequest(cmd, "GET", fmt.Sprintf("/api/v1/projects/%s/jobs", project))
	},
}

var webJobShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Get a specific job for a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			fmt.Println("Error: --project/-P flag is required")
			return
		}
		makeRequest(cmd, "GET", fmt.Sprintf("/api/v1/projects/%s/jobs/%s", project, args[0]))
	},
}

var webJobTriggerCmd = &cobra.Command{
	Use:   "trigger <id>",
	Short: "Trigger a specific job for a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			fmt.Println("Error: --project/-P flag is required")
			return
		}
		makeRequest(cmd, "POST", fmt.Sprintf("/api/v1/projects/%s/jobs/%s/trigger", project, args[0]))
	},
}

var webRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Manage job runs via API",
}

var webRunListCmd = &cobra.Command{
	Use:   "list",
	Short: "List runs for a specific job",
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		job, _ := cmd.Flags().GetString("job")
		if project == "" || job == "" {
			fmt.Println("Error: both --project/-P and --job/-J flags are required")
			return
		}
		makeRequest(cmd, "GET", fmt.Sprintf("/api/v1/projects/%s/jobs/%s/runs", project, job))
	},
}

var webTaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks via API",
}

var webWebTaskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks for a project",
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			fmt.Println("Error: --project/-P flag is required")
			return
		}
		makeRequest(cmd, "GET", fmt.Sprintf("/api/v1/projects/%s/tasks", project))
	},
}

var webTaskShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Get a specific task for a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			fmt.Println("Error: --project/-P flag is required")
			return
		}
		makeRequest(cmd, "GET", fmt.Sprintf("/api/v1/projects/%s/tasks/%s", project, args[0]))
	},
}

var webTaskTriggerCmd = &cobra.Command{
	Use:   "trigger <id>",
	Short: "Trigger a specific task for a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			fmt.Println("Error: --project/-P flag is required")
			return
		}
		makeRequest(cmd, "POST", fmt.Sprintf("/api/v1/projects/%s/tasks/%s/trigger", project, args[0]))
	},
}

func init() {
	webCmd.AddCommand(webProjectCmd)
	webProjectCmd.AddCommand(webProjectListCmd)

	webCmd.AddCommand(webJobCmd)
	webJobCmd.PersistentFlags().StringP("project", "P", "", "Project ID")
	webJobCmd.AddCommand(webJobListCmd)
	webJobCmd.AddCommand(webJobShowCmd)
	webJobCmd.AddCommand(webJobTriggerCmd)

	webCmd.AddCommand(webTaskCmd)
	webTaskCmd.PersistentFlags().StringP("project", "P", "", "Project ID")
	webTaskCmd.AddCommand(webWebTaskListCmd)
	webTaskCmd.AddCommand(webTaskShowCmd)
	webTaskCmd.AddCommand(webTaskTriggerCmd)

	webCmd.AddCommand(webRunCmd)
	webRunCmd.PersistentFlags().StringP("project", "P", "", "Project ID")
	webRunCmd.PersistentFlags().StringP("job", "J", "", "Job ID")
	webRunCmd.AddCommand(webRunListCmd)
}
