package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TicketData represents a single ticket
type TicketData struct {
	Key       string                 `json:"key"`
	Project   string                 `json:"project"`
	Data      map[string]interface{} `json:"data"`
	Collected time.Time              `json:"collected"`
	Updated   time.Time              `json:"updated"`
	Version   int                    `json:"version"`
}

// ProjectDataset represents a collection of tickets
type ProjectDataset struct {
	ProjectKey string                 `json:"project_key"`
	LastUpdate time.Time              `json:"last_update"`
	TotalCount int                    `json:"total_count"`
	Tickets    map[string]*TicketData `json:"tickets"`
}

// DashboardData represents the complete dashboard dataset
type DashboardData struct {
	Projects   []string       `json:"projects"`
	Tickets    []TicketData   `json:"tickets"`
	Stats      DashboardStats `json:"stats"`
	LastUpdate time.Time      `json:"last_update"`
}

// DashboardStats contains aggregated statistics
type DashboardStats struct {
	TotalTickets   int            `json:"total_tickets"`
	StatusCounts   map[string]int `json:"status_counts"`
	PriorityCounts map[string]int `json:"priority_counts"`
	ProjectCounts  map[string]int `json:"project_counts"`
	TypeCounts     map[string]int `json:"type_counts"`
	AssigneeCounts map[string]int `json:"assignee_counts"`
}

var dataDir = "./data"

func main() {
	// Check if data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		fmt.Printf("Data directory '%s' does not exist. Creating it...\n", dataDir)
		os.MkdirAll(dataDir, 0755)
	}

	// Set up HTTP routes
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/api/dashboard", getDashboardData)
	http.HandleFunc("/api/projects", getProjects)
	http.HandleFunc("/api/tickets", getTickets)
	http.HandleFunc("/api/refresh", refreshData)

	// Serve static files
	fs := http.FileServer(http.Dir("."))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("🚀 Jira Collector Dashboard Server")
	fmt.Println("📊 Dashboard: http://localhost:8080")
	fmt.Println("🔌 API endpoints:")
	fmt.Println("   GET /api/dashboard - Get complete dashboard data")
	fmt.Println("   GET /api/projects - Get project list")
	fmt.Println("   GET /api/tickets - Get all tickets")
	fmt.Println("   POST /api/refresh - Refresh data")
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop the server")

	// Start server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Serve the index.html file
	data, err := ioutil.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Could not read index.html", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

func getDashboardData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data, err := loadAllData()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load data: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(data)
}

func getProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projects, err := getProjectList()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get projects: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(projects)
}

func getTickets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectFilter := r.URL.Query().Get("project")
	statusFilter := r.URL.Query().Get("status")
	priorityFilter := r.URL.Query().Get("priority")

	allTickets := []TicketData{}

	files, err := ioutil.ReadDir(dataDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read data directory: %v", err), http.StatusInternalServerError)
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" && strings.Contains(file.Name(), "_tickets") {
			dataset, err := loadProjectDataset(filepath.Join(dataDir, file.Name()))
			if err != nil {
				continue // Skip problematic files
			}

			for _, ticket := range dataset.Tickets {
				// Apply filters
				if projectFilter != "" && ticket.Project != projectFilter {
					continue
				}
				if statusFilter != "" {
					if status, ok := ticket.Data["status"].(string); !ok || status != statusFilter {
						continue
					}
				}
				if priorityFilter != "" {
					if priority, ok := ticket.Data["priority"].(string); !ok || priority != priorityFilter {
						continue
					}
				}

				allTickets = append(allTickets, *ticket)
			}
		}
	}

	json.NewEncoder(w).Encode(allTickets)
}

func refreshData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// In a real implementation, this would trigger the collector to run
	// For now, we'll just return a success message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Data refresh triggered",
		"timestamp": time.Now(),
	})
}

func loadAllData() (*DashboardData, error) {
	data := &DashboardData{
		Projects: []string{},
		Tickets:  []TicketData{},
		Stats: DashboardStats{
			StatusCounts:   make(map[string]int),
			PriorityCounts: make(map[string]int),
			ProjectCounts:  make(map[string]int),
			TypeCounts:     make(map[string]int),
			AssigneeCounts: make(map[string]int),
		},
		LastUpdate: time.Now(),
	}

	files, err := ioutil.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	projectSet := make(map[string]bool)

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" && strings.Contains(file.Name(), "_tickets") {
			dataset, err := loadProjectDataset(filepath.Join(dataDir, file.Name()))
			if err != nil {
				continue // Skip problematic files
			}

			projectSet[dataset.ProjectKey] = true
			data.Stats.ProjectCounts[dataset.ProjectKey] = dataset.TotalCount

			for _, ticket := range dataset.Tickets {
				data.Tickets = append(data.Tickets, *ticket)
				data.Stats.TotalTickets++

				// Count statuses
				if status, ok := ticket.Data["status"].(string); ok {
					data.Stats.StatusCounts[status]++
				}

				// Count priorities
				if priority, ok := ticket.Data["priority"].(string); ok {
					data.Stats.PriorityCounts[priority]++
				}

				// Count types
				if issueType, ok := ticket.Data["issue_type"].(string); ok {
					data.Stats.TypeCounts[issueType]++
				}

				// Count assignees
				if assignee, ok := ticket.Data["assignee"].(string); ok {
					data.Stats.AssigneeCounts[assignee]++
				}
			}

			if dataset.LastUpdate.After(data.LastUpdate) {
				data.LastUpdate = dataset.LastUpdate
			}
		}
	}

	// Convert project set to slice
	for project := range projectSet {
		data.Projects = append(data.Projects, project)
	}

	return data, nil
}

func loadProjectDataset(filePath string) (*ProjectDataset, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var dataset ProjectDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &dataset, nil
}

func getProjectList() ([]string, error) {
	projects := make(map[string]bool)

	files, err := ioutil.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" && strings.Contains(file.Name(), "_tickets") {
			// Extract project key from filename (e.g., "dev_tickets.json" -> "dev")
			project := strings.TrimSuffix(file.Name(), "_tickets.json")
			projects[project] = true
		}
	}

	result := make([]string, 0, len(projects))
	for project := range projects {
		result = append(result, project)
	}

	return result, nil
}
