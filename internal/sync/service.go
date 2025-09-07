package sync

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github-todoist-sync/internal/config"
	"github-todoist-sync/internal/github"
	"github-todoist-sync/internal/todoist"
)

type Service struct {
	githubClient  *github.Client
	todoistClient *todoist.Client
	config        *config.Config
	project       *todoist.Project
}

func NewService(cfg *config.Config) (*Service, error) {
	githubClient := github.NewClient(cfg.GitHub.Token, cfg.GitHub.Owner, cfg.GitHub.Repo)
	todoistClient := todoist.NewClient(cfg.Todoist.Token)

	service := &Service{
		githubClient:  githubClient,
		todoistClient: todoistClient,
		config:        cfg,
	}

	project, err := service.ensureProject()
	if err != nil {
		return nil, fmt.Errorf("chyba při nastavování projektu: %v", err)
	}
	service.project = project

	return service, nil
}

func (s *Service) ensureProject() (*todoist.Project, error) {
	project, err := s.todoistClient.GetProjectByName(s.config.Todoist.ProjectName)
	if err == nil {
		log.Printf("Používám existující Todoist projekt: %s (ID: %s)", project.Name, project.ID)
		return project, nil
	}

	log.Printf("Vytvářím nový Todoist projekt: %s", s.config.Todoist.ProjectName)
	project, err = s.todoistClient.CreateProject(s.config.Todoist.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("nepodařilo se vytvořit projekt: %v", err)
	}

	log.Printf("Vytvořen nový Todoist projekt: %s (ID: %s)", project.Name, project.ID)
	return project, nil
}

func (s *Service) SyncFromGitHub(ctx context.Context) error {
	log.Printf("Začínám synchronizaci GitHub → Todoist...")

	issues, err := s.githubClient.GetIssues(ctx)
	if err != nil {
		return fmt.Errorf("chyba při získávání GitHub issues: %v", err)
	}

	log.Printf("Nalezeno %d GitHub issues", len(issues))

	existingTasks, err := s.todoistClient.GetTasks(s.project.ID)
	if err != nil {
		return fmt.Errorf("chyba při získávání Todoist úkolů: %v", err)
	}

	taskMap := make(map[int]*todoist.Task)
	for _, task := range existingTasks {
		if issueNum := s.extractGitHubIssueNumber(task.Description); issueNum != 0 {
			taskMap[issueNum] = task
		}
	}

	var syncedCount int
	for _, issue := range issues {
		if issue.IsPullReq {
			continue
		}

		existingTask, exists := taskMap[issue.Number]

		if !exists {
			if err := s.createTodoistTask(issue); err != nil {
				log.Printf("Chyba při vytváření úkolu pro issue #%d: %v", issue.Number, err)
				continue
			}
			syncedCount++
			log.Printf("Vytvořen úkol pro issue #%d: %s", issue.Number, issue.Title)
		} else {
			if err := s.updateTodoistTask(existingTask, issue); err != nil {
				log.Printf("Chyba při aktualizaci úkolu pro issue #%d: %v", issue.Number, err)
				continue
			}
			log.Printf("Aktualizován úkol pro issue #%d", issue.Number)
		}
	}

	log.Printf("GitHub → Todoist synchronizace dokončena. Zpracováno: %d úkolů", syncedCount)
	return nil
}

func (s *Service) SyncToGitHub(ctx context.Context) error {
	log.Printf("Začínám synchronizaci Todoist → GitHub...")

	tasks, err := s.todoistClient.GetTasks(s.project.ID)
	if err != nil {
		return fmt.Errorf("chyba při získávání Todoist úkolů: %v", err)
	}

	var syncedCount int
	for _, task := range tasks {
		issueNumber := s.extractGitHubIssueNumber(task.Description)
		if issueNumber == 0 {
			continue // Není to GitHub issue
		}

		issue, err := s.githubClient.GetIssue(ctx, issueNumber)
		if err != nil {
			log.Printf("Chyba při získávání issue #%d: %v", issueNumber, err)
			continue
		}

		if err := s.syncTaskStateToGitHub(ctx, task, issue); err != nil {
			log.Printf("Chyba při synchronizaci stavu issue #%d: %v", issueNumber, err)
			continue
		}

		syncedCount++
	}

	log.Printf("Todoist → GitHub synchronizace dokončena. Zpracováno: %d úkolů", syncedCount)
	return nil
}

func (s *Service) FullSync(ctx context.Context) error {
	log.Printf("Spouštím úplnou synchronizaci...")

	if err := s.SyncFromGitHub(ctx); err != nil {
		return fmt.Errorf("chyba při synchronizaci GitHub → Todoist: %v", err)
	}

	if err := s.SyncToGitHub(ctx); err != nil {
		return fmt.Errorf("chyba při synchronizaci Todoist → GitHub: %v", err)
	}

	log.Printf("Úplná synchronizace dokončena")
	return nil
}

func (s *Service) createTodoistTask(issue *github.Issue) error {
	task := &todoist.CreateTaskRequest{
		Content:     issue.Title,
		Description: todoist.FormatGitHubReference(issue.Number, issue.HTMLURL),
		ProjectID:   s.project.ID,
		Priority:    todoist.GetLabelPriority(issue.Labels),
		Labels:      s.convertLabels(issue.Labels),
	}

	_, err := s.todoistClient.CreateTask(task)
	return err
}

func (s *Service) updateTodoistTask(task *todoist.Task, issue *github.Issue) error {
	updates := make(map[string]interface{})

	if task.Content != issue.Title {
		updates["content"] = issue.Title
	}

	newPriority := todoist.GetLabelPriority(issue.Labels)
	if task.Priority != newPriority {
		updates["priority"] = newPriority
	}

	shouldBeClosed := issue.State == "closed"
	if task.IsCompleted != shouldBeClosed {
		if shouldBeClosed {
			return s.todoistClient.CloseTask(task.ID)
		} else {
			return s.todoistClient.ReopenTask(task.ID)
		}
	}

	if len(updates) > 0 {
		return s.todoistClient.UpdateTask(task.ID, updates)
	}

	return nil
}

func (s *Service) syncTaskStateToGitHub(ctx context.Context, task *todoist.Task, issue *github.Issue) error {
	if task.IsCompleted && issue.State == "open" {
		log.Printf("Uzavírám GitHub issue #%d (dokončeno v Todoist)", issue.Number)
		return s.githubClient.UpdateIssueState(ctx, issue.Number, "closed")
	}

	if !task.IsCompleted && issue.State == "closed" {
		log.Printf("Otevírám GitHub issue #%d (znovu otevřeno v Todoist)", issue.Number)
		return s.githubClient.UpdateIssueState(ctx, issue.Number, "open")
	}

	return nil
}

func (s *Service) extractGitHubIssueNumber(description string) int {
	if strings.Contains(description, "GitHub Issue #") {
		parts := strings.Split(description, "GitHub Issue #")
		if len(parts) > 1 {
			numberPart := strings.Split(parts[1], ":")[0]
			if number, err := strconv.Atoi(numberPart); err == nil {
				return number
			}
		}
	}
	return 0
}

func (s *Service) convertLabels(githubLabels []string) []string {
	var todoistLabels []string
	for _, label := range githubLabels {
		cleanLabel := strings.ReplaceAll(strings.ToLower(label), " ", "_")
		cleanLabel = strings.ReplaceAll(cleanLabel, "-", "_")
		if cleanLabel != "" {
			todoistLabels = append(todoistLabels, cleanLabel)
		}
	}
	return todoistLabels
}
