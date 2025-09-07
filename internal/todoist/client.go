package todoist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL = "https://api.todoist.com/rest/v2"
)

type Client struct {
	token      string
	httpClient *http.Client
}

type Project struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CommentCount int    `json:"comment_count"`
	Order        int    `json:"order"`
	Color        string `json:"color"`
	Shared       bool   `json:"shared"`
	Favorite     bool   `json:"favorite"`
	InboxProject bool   `json:"inbox_project"`
	TeamInbox    bool   `json:"team_inbox"`
	ViewStyle    string `json:"view_style"`
	URL          string `json:"url"`
	ParentID     string `json:"parent_id,omitempty"`
}

type Task struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	SectionID    string    `json:"section_id,omitempty"`
	Content      string    `json:"content"`
	Description  string    `json:"description,omitempty"`
	IsCompleted  bool      `json:"is_completed"`
	Labels       []string  `json:"labels"`
	ParentID     string    `json:"parent_id,omitempty"`
	Order        int       `json:"order"`
	Priority     int       `json:"priority"`
	Due          *Due      `json:"due,omitempty"`
	URL          string    `json:"url"`
	CommentCount int       `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	CreatorID    string    `json:"creator_id"`
	AssigneeID   string    `json:"assignee_id,omitempty"`
	AssignerID   string    `json:"assigner_id,omitempty"`
}

type Due struct {
	String      string `json:"string"`
	Date        string `json:"date"`
	IsRecurring bool   `json:"is_recurring"`
	Datetime    string `json:"datetime,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
}

type CreateTaskRequest struct {
	Content     string   `json:"content"`
	Description string   `json:"description,omitempty"`
	ProjectID   string   `json:"project_id,omitempty"`
	SectionID   string   `json:"section_id,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"`
	Order       int      `json:"order,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	DueString   string   `json:"due_string,omitempty"`
	DueDate     string   `json:"due_date,omitempty"`
	DueDatetime string   `json:"due_datetime,omitempty"`
	DueLang     string   `json:"due_lang,omitempty"`
	AssigneeID  string   `json:"assignee_id,omitempty"`
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) GetProjects() ([]*Project, error) {
	req, err := c.createRequest("GET", "/projects", nil)
	if err != nil {
		return nil, err
	}

	var projects []*Project
	if err := c.doRequest(req, &projects); err != nil {
		return nil, fmt.Errorf("chyba při získávání projektů: %v", err)
	}

	return projects, nil
}

func (c *Client) GetProjectByName(name string) (*Project, error) {
	projects, err := c.GetProjects()
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if project.Name == name {
			return project, nil
		}
	}

	return nil, fmt.Errorf("projekt '%s' nebyl nalezen", name)
}

func (c *Client) CreateProject(name string) (*Project, error) {
	payload := map[string]string{"name": name}

	req, err := c.createRequest("POST", "/projects", payload)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := c.doRequest(req, &project); err != nil {
		return nil, fmt.Errorf("chyba při vytváření projektu: %v", err)
	}

	return &project, nil
}

func (c *Client) GetTasks(projectID string) ([]*Task, error) {
	url := "/tasks"
	if projectID != "" {
		url += "?project_id=" + projectID
	}

	req, err := c.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var tasks []*Task
	if err := c.doRequest(req, &tasks); err != nil {
		return nil, fmt.Errorf("chyba při získávání úkolů: %v", err)
	}

	return tasks, nil
}

func (c *Client) CreateTask(task *CreateTaskRequest) (*Task, error) {
	req, err := c.createRequest("POST", "/tasks", task)
	if err != nil {
		return nil, err
	}

	var createdTask Task
	if err := c.doRequest(req, &createdTask); err != nil {
		return nil, fmt.Errorf("chyba při vytváření úkolu: %v", err)
	}

	return &createdTask, nil
}

func (c *Client) UpdateTask(taskID string, updates map[string]interface{}) error {
	req, err := c.createRequest("POST", "/tasks/"+taskID, updates)
	if err != nil {
		return err
	}

	if err := c.doRequest(req, nil); err != nil {
		return fmt.Errorf("chyba při aktualizaci úkolu: %v", err)
	}

	return nil
}

func (c *Client) CloseTask(taskID string) error {
	req, err := c.createRequest("POST", "/tasks/"+taskID+"/close", nil)
	if err != nil {
		return err
	}

	if err := c.doRequest(req, nil); err != nil {
		return fmt.Errorf("chyba při uzavírání úkolu: %v", err)
	}

	return nil
}

func (c *Client) ReopenTask(taskID string) error {
	req, err := c.createRequest("POST", "/tasks/"+taskID+"/reopen", nil)
	if err != nil {
		return err
	}

	if err := c.doRequest(req, nil); err != nil {
		return fmt.Errorf("chyba při znovuotevření úkolu: %v", err)
	}

	return nil
}

func (c *Client) FindTaskByDescription(projectID, description string) (*Task, error) {
	tasks, err := c.GetTasks(projectID)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if strings.Contains(task.Description, description) {
			return task, nil
		}
	}

	return nil, nil
}

func (c *Client) createRequest(method, path string, body interface{}) (*http.Request, error) {
	url := baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *Client) doRequest(req *http.Request, target interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if target != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(target)
	}

	return nil
}

func GetLabelPriority(labels []string) int {
	priorityMap := map[string]int{
		"urgent": 4,
		"high":   3,
		"medium": 2,
		"low":    1,
	}

	for _, label := range labels {
		if priority, exists := priorityMap[strings.ToLower(label)]; exists {
			return priority
		}
	}

	return 1 // Default priority
}

func FormatGitHubReference(issueNumber int, url string) string {
	return fmt.Sprintf("GitHub Issue #%d: %s", issueNumber, url)
}
