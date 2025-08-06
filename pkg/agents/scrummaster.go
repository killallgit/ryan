package agents

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// ScrumMaster is a meta-orchestrator that manages complex, multi-phase projects
type ScrumMaster struct {
	orchestrator     *Orchestrator
	planner          *Planner
	projectMonitor   *ProjectMonitor
	resourceManager  *ResourceManager
	strategySelector *StrategySelector
	activeProjects   map[string]*Project
	projectHistory   map[string]*ProjectHistory
	log              *logger.Logger
	mu               sync.RWMutex
}

// NewScrumMaster creates a new ScrumMaster agent
func NewScrumMaster(orchestrator *Orchestrator) *ScrumMaster {
	return &ScrumMaster{
		orchestrator:     orchestrator,
		planner:          orchestrator.planner,
		projectMonitor:   NewProjectMonitor(),
		resourceManager:  NewResourceManager(),
		strategySelector: NewStrategySelector(),
		activeProjects:   make(map[string]*Project),
		projectHistory:   make(map[string]*ProjectHistory),
		log:              logger.WithComponent("scrummaster"),
	}
}

// Name returns the agent name
func (sm *ScrumMaster) Name() string {
	return "scrummaster"
}

// Description returns the agent description
func (sm *ScrumMaster) Description() string {
	return "Meta-orchestrator for complex multi-phase projects with adaptive planning and resource management"
}

// CanHandle determines if this agent can handle the request
func (sm *ScrumMaster) CanHandle(request string) (bool, float64) {
	lowerRequest := strings.ToLower(request)

	// Look for indicators of complex, multi-phase work
	complexIndicators := []string{
		"project", "implement", "build", "create system", "develop",
		"multiple", "phases", "sprints", "epic", "feature set",
		"coordinate", "manage", "orchestrate", "plan and execute",
		"application", "complete", "full",
	}

	score := 0.0
	for _, indicator := range complexIndicators {
		if strings.Contains(lowerRequest, indicator) {
			score += 0.15
		}
	}

	// Check for architectural/system indicators
	systemIndicators := []string{
		"system", "architecture", "distributed", "microservices",
		"frontend", "backend", "database", "infrastructure",
	}
	for _, indicator := range systemIndicators {
		if strings.Contains(lowerRequest, indicator) {
			score += 0.15
		}
	}

	// Check for size/complexity indicators
	if strings.Contains(lowerRequest, "large") || strings.Contains(lowerRequest, "complex") {
		score += 0.2
	}

	// Check if request mentions multiple components or systems
	componentCount := 0
	components := []string{"frontend", "backend", "database", "api", "service", "module"}
	for _, comp := range components {
		if strings.Contains(lowerRequest, comp) {
			componentCount++
		}
	}
	if componentCount >= 2 {
		score += 0.3
	}

	// Additional score for "and" connectives suggesting multiple parts
	if strings.Count(lowerRequest, "and") >= 2 {
		score += 0.1
	}

	// Cap the score at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score > 0.3, score
}

// Execute performs the ScrumMaster's project management task
func (sm *ScrumMaster) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	startTime := time.Now()
	sm.log.Info("Starting project execution", "prompt_preview", truncateString(request.Prompt, 100))

	// Analyze the request to create a project
	project, err := sm.analyzeAndCreateProject(ctx, request)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Failed to create project",
			Details: err.Error(),
			Metadata: AgentMetadata{
				AgentName: sm.Name(),
				StartTime: startTime,
				EndTime:   time.Now(),
				Duration:  time.Since(startTime),
			},
		}, err
	}

	// Store the active project
	sm.mu.Lock()
	sm.activeProjects[project.ID] = project
	sm.mu.Unlock()

	// Start project monitoring
	monitorCtx, cancelMonitor := context.WithCancel(ctx)
	defer cancelMonitor()
	go sm.projectMonitor.MonitorProject(monitorCtx, project)

	// Execute the project
	projectResult, err := sm.executeProject(ctx, project)
	if err != nil {
		sm.log.Error("Project execution failed", "project_id", project.ID, "error", err)
		projectResult.Success = false
		projectResult.Details = fmt.Sprintf("Project failed: %v", err)
	}

	// Archive the project
	sm.archiveProject(project, projectResult)

	return projectResult, nil
}

// analyzeAndCreateProject analyzes the request and creates a project structure
func (sm *ScrumMaster) analyzeAndCreateProject(ctx context.Context, request AgentRequest) (*Project, error) {
	sm.log.Debug("Analyzing request for project creation")

	// Create project context
	projectContext := &ProjectContext{
		ExecutionContext: &ExecutionContext{
			SessionID:   generateID(),
			RequestID:   generateID(),
			UserPrompt:  request.Prompt,
			SharedData:  make(map[string]interface{}),
			FileContext: []FileInfo{},
			Artifacts:   make(map[string]interface{}),
			Options:     request.Options,
		},
		StartTime:    time.Now(),
		Constraints:  sm.extractConstraints(request),
		Goals:        sm.extractGoals(request),
		Stakeholders: []string{"user"},
	}

	// Decompose the request into epics and sprints
	epics, err := sm.decomposeIntoEpics(ctx, request.Prompt, projectContext)
	if err != nil {
		return nil, fmt.Errorf("failed to decompose into epics: %w", err)
	}

	// Create project
	project := &Project{
		ID:          generateID(),
		Name:        sm.extractProjectName(request.Prompt),
		Description: request.Prompt,
		Epics:       epics,
		Context:     projectContext,
		Status:      ProjectStatusPlanning,
		CreatedAt:   time.Now(),
	}

	// Create sprints from epics
	project.Sprints = sm.createSprintsFromEpics(epics, projectContext)

	sm.log.Info("Created project",
		"project_id", project.ID,
		"epics", len(project.Epics),
		"sprints", len(project.Sprints))

	return project, nil
}

// decomposeIntoEpics breaks down the request into epic-level components
func (sm *ScrumMaster) decomposeIntoEpics(ctx context.Context, prompt string, context *ProjectContext) ([]*Epic, error) {
	// Use the planner to analyze intent and identify major components
	intent, err := sm.planner.intentAnalyzer.Analyze(prompt)
	if err != nil {
		return nil, err
	}

	epics := []*Epic{}

	// Create epics based on intent and request analysis
	// This is a simplified version - in practice, this would be more sophisticated
	primaryEpic := &Epic{
		ID:          generateID(),
		Title:       fmt.Sprintf("Implement %s", intent.Primary),
		Description: prompt,
		Priority:    PriorityHigh,
		Status:      EpicStatusTodo,
		Stories:     []*UserStory{},
	}

	// Add stories based on secondary intents
	for _, secondary := range intent.Secondary {
		story := &UserStory{
			ID:          generateID(),
			Title:       fmt.Sprintf("Add %s support", secondary),
			Description: fmt.Sprintf("Implement %s functionality", secondary),
			Points:      3, // Default story points
			Priority:    PriorityMedium,
			Status:      StoryStatusTodo,
		}
		primaryEpic.Stories = append(primaryEpic.Stories, story)
	}

	// If no secondary intents, create default story
	if len(primaryEpic.Stories) == 0 {
		primaryEpic.Stories = append(primaryEpic.Stories, &UserStory{
			ID:          generateID(),
			Title:       "Core implementation",
			Description: prompt,
			Points:      5,
			Priority:    PriorityHigh,
			Status:      StoryStatusTodo,
		})
	}

	epics = append(epics, primaryEpic)

	// Add testing epic if relevant
	if containsAny(strings.ToLower(prompt), []string{"test", "quality", "verify"}) {
		testEpic := &Epic{
			ID:          generateID(),
			Title:       "Testing and Validation",
			Description: "Ensure quality and correctness",
			Priority:    PriorityMedium,
			Status:      EpicStatusTodo,
			Stories: []*UserStory{
				{
					ID:          generateID(),
					Title:       "Create test suite",
					Description: "Develop comprehensive tests",
					Points:      3,
					Priority:    PriorityMedium,
					Status:      StoryStatusTodo,
				},
			},
		}
		epics = append(epics, testEpic)
	}

	return epics, nil
}

// createSprintsFromEpics creates sprints from the epics
func (sm *ScrumMaster) createSprintsFromEpics(epics []*Epic, context *ProjectContext) []*Sprint {
	sprints := []*Sprint{}
	sprintNum := 1

	for _, epic := range epics {
		// Create a sprint for each epic (simplified approach)
		sprint := &Sprint{
			ID:        generateID(),
			Number:    sprintNum,
			Name:      fmt.Sprintf("Sprint %d: %s", sprintNum, epic.Title),
			Goal:      epic.Description,
			StartDate: time.Now().Add(time.Duration(sprintNum-1) * 7 * 24 * time.Hour),
			EndDate:   time.Now().Add(time.Duration(sprintNum) * 7 * 24 * time.Hour),
			Status:    SprintStatusPlanning,
			Stories:   epic.Stories,
			Capacity:  40, // Default capacity in story points
		}

		// Create execution plans for the sprint
		sprint.Plans = sm.createSprintPlans(sprint, context)

		sprints = append(sprints, sprint)
		sprintNum++
	}

	return sprints
}

// createSprintPlans creates execution plans for a sprint
func (sm *ScrumMaster) createSprintPlans(sprint *Sprint, context *ProjectContext) []*ExecutionPlan {
	plans := []*ExecutionPlan{}

	for _, story := range sprint.Stories {
		// Create a simple execution plan for each story
		plan := &ExecutionPlan{
			ID:                generateID(),
			RequestID:         context.RequestID,
			Tasks:             sm.createTasksForStory(story),
			Stages:            []Stage{},
			EstimatedDuration: fmt.Sprintf("%dh", story.Points*2), // Rough estimate
			CreatedAt:         time.Now(),
		}

		// Create stages
		if len(plan.Tasks) > 0 {
			plan.Stages = []Stage{
				{
					ID:    fmt.Sprintf("stage-%s", story.ID),
					Name:  fmt.Sprintf("Story %s", story.Title),
					Tasks: plan.Tasks,
				},
			}
		}

		plans = append(plans, plan)
	}

	return plans
}

// createTasksForStory creates tasks for a user story
func (sm *ScrumMaster) createTasksForStory(story *UserStory) []Task {
	tasks := []Task{}

	// Create a task based on the story
	task := Task{
		ID:          generateID(),
		Name:        story.Title,
		Description: story.Description,
		Agent:       "dispatcher", // Use dispatcher for flexibility
		Request: AgentRequest{
			Prompt: story.Description,
			Context: map[string]interface{}{
				"story_id":    story.ID,
				"story_title": story.Title,
				"priority":    story.Priority,
			},
		},
		Priority:     int(story.Priority),
		Dependencies: []string{},
	}

	tasks = append(tasks, task)
	return tasks
}

// executeProject executes the entire project
func (sm *ScrumMaster) executeProject(ctx context.Context, project *Project) (AgentResult, error) {
	sm.log.Info("Executing project", "project_id", project.ID, "sprints", len(project.Sprints))

	project.Status = ProjectStatusInProgress
	allResults := []SprintResult{}
	overallSuccess := true

	// Execute each sprint sequentially
	for _, sprint := range project.Sprints {
		sm.log.Info("Starting sprint", "sprint_id", sprint.ID, "sprint_name", sprint.Name)

		// Check if we should continue based on previous results
		if !sm.shouldContinueSprint(ctx, project, allResults) {
			sm.log.Info("Stopping project execution based on feedback")
			break
		}

		// Execute the sprint
		sprintResult, err := sm.executeSprint(ctx, sprint, project)
		if err != nil {
			sm.log.Error("Sprint execution failed", "sprint_id", sprint.ID, "error", err)
			overallSuccess = false
			sprintResult.Success = false
			sprintResult.Error = err
		}

		allResults = append(allResults, sprintResult)

		// Adapt plan based on sprint results if needed
		if !sprintResult.Success && sm.shouldAdaptPlan(sprintResult) {
			sm.adaptProjectPlan(ctx, project, sprintResult)
		}
	}

	// Update project status
	if overallSuccess {
		project.Status = ProjectStatusCompleted
	} else {
		project.Status = ProjectStatusFailed
	}

	// Generate final project report
	summary, details := sm.generateProjectReport(project, allResults)

	return AgentResult{
		Success: overallSuccess,
		Summary: summary,
		Details: details,
		Artifacts: map[string]interface{}{
			"project":        project,
			"sprint_results": allResults,
		},
		Metadata: AgentMetadata{
			AgentName: sm.Name(),
			StartTime: project.Context.StartTime,
			EndTime:   time.Now(),
			Duration:  time.Since(project.Context.StartTime),
		},
	}, nil
}

// executeSprint executes a single sprint
func (sm *ScrumMaster) executeSprint(ctx context.Context, sprint *Sprint, project *Project) (SprintResult, error) {
	sprint.Status = SprintStatusActive
	startTime := time.Now()

	result := SprintResult{
		SprintID:  sprint.ID,
		Success:   true,
		StartTime: startTime,
	}

	// Execute each plan in the sprint
	for _, plan := range sprint.Plans {
		sm.log.Debug("Executing sprint plan", "plan_id", plan.ID)

		// Use the orchestrator to execute the plan
		taskResults, err := sm.orchestrator.ExecuteWithPlan(ctx, plan, project.Context.ExecutionContext)
		if err != nil {
			result.Success = false
			result.Error = err
			continue
		}

		// Collect results
		for _, taskResult := range taskResults {
			// Check if result is an AgentResult type
			if agentResult, ok := taskResult.Result.(AgentResult); ok {
				if !agentResult.Success {
					result.Success = false
				}
			} else {
				// If not AgentResult, assume failure
				result.Success = false
			}
			result.CompletedStories = append(result.CompletedStories, taskResult.Task.ID)
		}
	}

	sprint.Status = SprintStatusCompleted
	result.EndTime = time.Now()
	result.Velocity = sm.calculateVelocity(sprint, result)

	return result, nil
}

// Helper methods

func (sm *ScrumMaster) extractProjectName(prompt string) string {
	// Simple extraction - take first few words or look for "project" keyword
	words := strings.Fields(prompt)
	if len(words) > 3 {
		return strings.Join(words[:3], " ")
	}
	return "Project"
}

func (sm *ScrumMaster) extractConstraints(request AgentRequest) map[string]interface{} {
	constraints := make(map[string]interface{})

	// Extract from options if available
	if timeout, ok := request.Options["timeout"]; ok {
		constraints["timeout"] = timeout
	}
	if budget, ok := request.Options["budget"]; ok {
		constraints["budget"] = budget
	}

	// Default constraints
	constraints["max_concurrent_tasks"] = 5
	constraints["max_retries"] = 3

	return constraints
}

func (sm *ScrumMaster) extractGoals(request AgentRequest) []string {
	goals := []string{}

	// Extract goals from prompt
	prompt := strings.ToLower(request.Prompt)
	if strings.Contains(prompt, "implement") {
		goals = append(goals, "Complete implementation")
	}
	if strings.Contains(prompt, "test") {
		goals = append(goals, "Ensure quality through testing")
	}
	if strings.Contains(prompt, "document") {
		goals = append(goals, "Provide comprehensive documentation")
	}

	// Default goal if none found
	if len(goals) == 0 {
		goals = append(goals, "Successfully complete the requested task")
	}

	return goals
}

func (sm *ScrumMaster) shouldContinueSprint(ctx context.Context, project *Project, previousResults []SprintResult) bool {
	// Check if we should continue based on previous results
	if len(previousResults) == 0 {
		return true
	}

	// Check failure rate
	failureCount := 0
	for _, result := range previousResults {
		if !result.Success {
			failureCount++
		}
	}

	// Stop if too many failures
	if float64(failureCount)/float64(len(previousResults)) > 0.5 {
		return false
	}

	return true
}

func (sm *ScrumMaster) shouldAdaptPlan(result SprintResult) bool {
	// Determine if we should adapt the plan based on results
	return !result.Success && result.Error != nil
}

func (sm *ScrumMaster) adaptProjectPlan(ctx context.Context, project *Project, failedSprint SprintResult) {
	sm.log.Info("Adapting project plan based on sprint failure", "sprint_id", failedSprint.SprintID)

	// Simple adaptation: mark remaining sprints as lower priority
	// In a real implementation, this would be much more sophisticated
	for _, sprint := range project.Sprints {
		if sprint.Status == SprintStatusPlanning {
			// Reduce scope or adjust priorities
			for _, story := range sprint.Stories {
				if story.Priority == PriorityHigh {
					story.Priority = PriorityMedium
				}
			}
		}
	}
}

func (sm *ScrumMaster) calculateVelocity(sprint *Sprint, result SprintResult) int {
	// Calculate velocity based on completed stories
	velocity := 0
	completedMap := make(map[string]bool)
	for _, storyID := range result.CompletedStories {
		completedMap[storyID] = true
	}

	for _, story := range sprint.Stories {
		if completedMap[story.ID] {
			velocity += story.Points
		}
	}

	return velocity
}

func (sm *ScrumMaster) generateProjectReport(project *Project, results []SprintResult) (string, string) {
	// Generate summary
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	summary := fmt.Sprintf("Project '%s' completed: %d/%d sprints successful",
		project.Name, successCount, len(results))

	// Generate detailed report
	var details strings.Builder
	details.WriteString(fmt.Sprintf("Project: %s\n", project.Name))
	details.WriteString(fmt.Sprintf("ID: %s\n", project.ID))
	details.WriteString(fmt.Sprintf("Status: %s\n", project.Status))
	details.WriteString(fmt.Sprintf("Duration: %v\n\n", time.Since(project.Context.StartTime)))

	details.WriteString("Sprint Results:\n")
	for i, result := range results {
		sprint := project.Sprints[i]
		details.WriteString(fmt.Sprintf("\n%s:\n", sprint.Name))
		details.WriteString(fmt.Sprintf("  Status: %s\n", sprint.Status))
		details.WriteString(fmt.Sprintf("  Success: %v\n", result.Success))
		details.WriteString(fmt.Sprintf("  Velocity: %d\n", result.Velocity))
		details.WriteString(fmt.Sprintf("  Duration: %v\n", result.EndTime.Sub(result.StartTime)))
		if result.Error != nil {
			details.WriteString(fmt.Sprintf("  Error: %v\n", result.Error))
		}
	}

	details.WriteString("\nGoals Achievement:\n")
	for _, goal := range project.Context.Goals {
		details.WriteString(fmt.Sprintf("- %s\n", goal))
	}

	return summary, details.String()
}

func (sm *ScrumMaster) archiveProject(project *Project, result AgentResult) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Remove from active projects
	delete(sm.activeProjects, project.ID)

	// Add to history
	history := &ProjectHistory{
		Project:      project,
		Result:       result,
		ArchivedAt:   time.Now(),
		LessonsLearn: sm.extractLessonsLearned(project, result),
	}

	sm.projectHistory[project.ID] = history
}

func (sm *ScrumMaster) extractLessonsLearned(project *Project, result AgentResult) []string {
	lessons := []string{}

	if !result.Success {
		lessons = append(lessons, "Project encountered challenges that need addressing")
	}

	// Add more sophisticated lesson extraction in real implementation

	return lessons
}

// Helper function
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func extractTaskIDs(tasks []Task) []string {
	ids := make([]string, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	return ids
}
