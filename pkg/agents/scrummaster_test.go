package agents

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrumMaster_CanHandle(t *testing.T) {
	orchestrator := NewOrchestrator()
	sm := NewScrumMaster(orchestrator)

	tests := []struct {
		name            string
		request         string
		expectCanHandle bool
		minConfidence   float64
	}{
		{
			name:            "complex project request",
			request:         "Build a complete web application with frontend, backend, and database",
			expectCanHandle: true,
			minConfidence:   0.5,
		},
		{
			name:            "multi-phase project",
			request:         "Implement a distributed system with multiple microservices and orchestration",
			expectCanHandle: true,
			minConfidence:   0.7,
		},
		{
			name:            "simple task",
			request:         "Fix a typo in the README",
			expectCanHandle: false,
			minConfidence:   0.0,
		},
		{
			name:            "project keyword",
			request:         "Create a project to manage customer data with multiple phases",
			expectCanHandle: true,
			minConfidence:   0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canHandle, confidence := sm.CanHandle(tt.request)
			assert.Equal(t, tt.expectCanHandle, canHandle, "CanHandle mismatch")
			if tt.expectCanHandle {
				assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence too low")
			}
		})
	}
}

func TestScrumMaster_ProjectCreation(t *testing.T) {
	orchestrator := NewOrchestrator()
	sm := NewScrumMaster(orchestrator)

	ctx := context.Background()
	request := AgentRequest{
		Prompt: "Build a complete e-commerce system with product catalog, shopping cart, and checkout",
		Options: map[string]interface{}{
			"timeout": 3600,
		},
	}

	// Test project analysis and creation
	project, err := sm.analyzeAndCreateProject(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, project)

	// Verify project structure
	assert.NotEmpty(t, project.ID)
	assert.NotEmpty(t, project.Name)
	assert.Equal(t, request.Prompt, project.Description)
	assert.Equal(t, ProjectStatusPlanning, project.Status)

	// Verify epics were created
	assert.NotEmpty(t, project.Epics, "Should have at least one epic")

	// Verify sprints were created
	assert.NotEmpty(t, project.Sprints, "Should have at least one sprint")

	// Verify context
	assert.NotNil(t, project.Context)
	assert.Equal(t, request.Prompt, project.Context.UserPrompt)
	assert.NotEmpty(t, project.Context.Goals)
}

func TestScrumMaster_EpicDecomposition(t *testing.T) {
	orchestrator := NewOrchestrator()
	sm := NewScrumMaster(orchestrator)

	ctx := context.Background()
	projectContext := &ProjectContext{
		ExecutionContext: &ExecutionContext{
			SessionID:   "test-session",
			RequestID:   "test-request",
			UserPrompt:  "Build a system with frontend and backend components",
			SharedData:  make(map[string]interface{}),
			FileContext: []FileInfo{},
		},
		StartTime: time.Now(),
		Goals:     []string{"Complete implementation"},
	}

	epics, err := sm.decomposeIntoEpics(ctx, projectContext.UserPrompt, projectContext)
	require.NoError(t, err)
	require.NotEmpty(t, epics)

	// Verify epic structure
	for _, epic := range epics {
		assert.NotEmpty(t, epic.ID)
		assert.NotEmpty(t, epic.Title)
		assert.NotEmpty(t, epic.Description)
		assert.Equal(t, EpicStatusTodo, epic.Status)
		assert.NotEmpty(t, epic.Stories, "Epic should have stories")

		// Verify stories
		for _, story := range epic.Stories {
			assert.NotEmpty(t, story.ID)
			assert.NotEmpty(t, story.Title)
			assert.Greater(t, story.Points, 0)
			assert.Equal(t, StoryStatusTodo, story.Status)
		}
	}
}

func TestScrumMaster_SprintCreation(t *testing.T) {
	orchestrator := NewOrchestrator()
	sm := NewScrumMaster(orchestrator)

	// Create test epics
	epics := []*Epic{
		{
			ID:          "epic-1",
			Title:       "Frontend Development",
			Description: "Build the frontend",
			Priority:    PriorityHigh,
			Status:      EpicStatusTodo,
			Stories: []*UserStory{
				{
					ID:          "story-1",
					Title:       "Create UI components",
					Description: "Build reusable components",
					Points:      5,
					Priority:    PriorityHigh,
					Status:      StoryStatusTodo,
				},
				{
					ID:          "story-2",
					Title:       "Implement routing",
					Description: "Set up application routing",
					Points:      3,
					Priority:    PriorityMedium,
					Status:      StoryStatusTodo,
				},
			},
		},
		{
			ID:          "epic-2",
			Title:       "Backend Development",
			Description: "Build the backend",
			Priority:    PriorityHigh,
			Status:      EpicStatusTodo,
			Stories: []*UserStory{
				{
					ID:          "story-3",
					Title:       "Create API endpoints",
					Description: "Build REST API",
					Points:      8,
					Priority:    PriorityHigh,
					Status:      StoryStatusTodo,
				},
			},
		},
	}

	projectContext := &ProjectContext{
		ExecutionContext: &ExecutionContext{
			SessionID:  "test-session",
			RequestID:  "test-request",
			SharedData: make(map[string]interface{}),
		},
	}

	sprints := sm.createSprintsFromEpics(epics, projectContext)
	require.NotEmpty(t, sprints)

	// Verify sprint structure
	totalStoryPoints := 0
	for _, sprint := range sprints {
		assert.NotEmpty(t, sprint.ID)
		assert.NotEmpty(t, sprint.Name)
		assert.Greater(t, sprint.Number, 0)
		assert.Equal(t, SprintStatusPlanning, sprint.Status)
		assert.Greater(t, sprint.Capacity, 0)

		// Count story points
		for _, story := range sprint.Stories {
			totalStoryPoints += story.Points
		}
	}

	// Verify all stories are assigned to sprints
	assert.Equal(t, 16, totalStoryPoints) // 5 + 3 + 8 = 16
}

func TestHierarchicalPlanner_ProjectAnalysis(t *testing.T) {
	analyzer := NewProjectAnalyzer()

	tests := []struct {
		name               string
		request            string
		expectedScope      ProjectScope
		minComplexity      float64
		expectedComponents []string
	}{
		{
			name:               "complex system",
			request:            "Build a distributed microservices architecture with frontend, backend, database, and API gateway",
			expectedScope:      ProjectScopeSmall,
			minComplexity:      0.5,
			expectedComponents: []string{"frontend", "backend", "database", "api"},
		},
		{
			name:               "simple project",
			request:            "Create a basic todo app",
			expectedScope:      ProjectScopeSmall,
			minComplexity:      0.3,
			expectedComponents: []string{},
		},
		{
			name:               "medium project",
			request:            strings.Repeat("Implement feature ", 10) + "with testing and documentation",
			expectedScope:      ProjectScopeSmall, // Only 23 words, not enough for medium
			minComplexity:      0.3,
			expectedComponents: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := analyzer.Analyze(tt.request)

			assert.Equal(t, tt.expectedScope, analysis.Scope)
			assert.GreaterOrEqual(t, analysis.Complexity, tt.minComplexity)

			for _, component := range tt.expectedComponents {
				assert.Contains(t, analysis.Components, component)
			}
		})
	}
}

func TestPlanner_IsProjectLevelRequest(t *testing.T) {
	planner := NewPlanner()

	tests := []struct {
		name      string
		request   string
		isProject bool
	}{
		{
			name:      "project keyword",
			request:   "Create a project to manage inventory with multiple components",
			isProject: true,
		},
		{
			name:      "system keyword",
			request:   "Build a complete system for processing payments",
			isProject: true,
		},
		{
			name:      "architecture keyword",
			request:   "Design and implement a microservices architecture",
			isProject: true,
		},
		{
			name:      "simple task",
			request:   "Fix the bug in the login function",
			isProject: false,
		},
		{
			name:      "long request",
			request:   strings.Repeat("Build feature ", 50),
			isProject: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := planner.IsProjectLevelRequest(tt.request)
			assert.Equal(t, tt.isProject, result)
		})
	}
}

func TestSprintPlanner_CapacityCalculation(t *testing.T) {
	planner := NewSprintPlanner()

	tests := []struct {
		complexity       float64
		expectedCapacity int
	}{
		{0.0, 40}, // No complexity = full capacity
		{0.5, 34}, // Medium complexity
		{1.0, 28}, // High complexity
	}

	for _, tt := range tests {
		capacity := planner.calculateSprintCapacity(tt.complexity)
		assert.Equal(t, tt.expectedCapacity, capacity)
	}
}

func TestTaskDecomposer_StoryDecomposition(t *testing.T) {
	decomposer := NewTaskDecomposer()

	stories := []struct {
		story         *UserStory
		expectedAgent string
	}{
		{
			story: &UserStory{
				ID:          "story-1",
				Title:       "Design system architecture",
				Description: "Create architectural design",
				Points:      5,
				Priority:    PriorityHigh,
			},
			expectedAgent: "code_analysis",
		},
		{
			story: &UserStory{
				ID:          "story-2",
				Title:       "Implement core features",
				Description: "Build the main functionality",
				Points:      8,
				Priority:    PriorityHigh,
			},
			expectedAgent: "dispatcher",
		},
		{
			story: &UserStory{
				ID:          "story-3",
				Title:       "Test the system",
				Description: "Create comprehensive tests",
				Points:      3,
				Priority:    PriorityMedium,
			},
			expectedAgent: "dispatcher",
		},
	}

	for _, tt := range stories {
		tasks := decomposer.DecomposeStory(tt.story)
		require.NotEmpty(t, tasks)

		// Verify task structure
		for _, task := range tasks {
			assert.NotEmpty(t, task.ID)
			assert.Equal(t, tt.expectedAgent, task.Agent)
			assert.NotEmpty(t, task.Request.Prompt)
			assert.Equal(t, int(tt.story.Priority), task.Priority)
			assert.Greater(t, task.Timeout, time.Duration(0))
		}
	}
}
