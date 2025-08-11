package orchestrator

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/logger"
)

// Router handles routing decisions between agents
type Router struct {
	registry *AgentRegistry
	rules    []RoutingRule
}

// RoutingRule defines a rule for routing decisions
type RoutingRule struct {
	Name      string
	Condition func(*TaskIntent) bool
	Target    AgentType
	Priority  int
}

// NewRouter creates a new router instance
func NewRouter(registry *AgentRegistry) *Router {
	r := &Router{
		registry: registry,
		rules:    make([]RoutingRule, 0),
	}

	// Initialize default routing rules
	r.initializeDefaultRules()
	return r
}

// initializeDefaultRules sets up the default routing rules
func (r *Router) initializeDefaultRules() {
	r.rules = []RoutingRule{
		{
			Name: "tool_use_routing",
			Condition: func(intent *TaskIntent) bool {
				return intent.Type == "tool_use" ||
					containsAny(intent.RequiredCapabilities, []string{"bash", "file", "git", "web"})
			},
			Target:   AgentToolCaller,
			Priority: 10,
		},
		{
			Name: "code_generation_routing",
			Condition: func(intent *TaskIntent) bool {
				return intent.Type == "code_generation" ||
					containsAny(intent.RequiredCapabilities, []string{"coding", "implementation", "refactoring"})
			},
			Target:   AgentCodeGen,
			Priority: 9,
		},
		{
			Name: "search_routing",
			Condition: func(intent *TaskIntent) bool {
				return intent.Type == "search" ||
					containsAny(intent.RequiredCapabilities, []string{"search", "find", "locate"})
			},
			Target:   AgentSearcher,
			Priority: 8,
		},
		{
			Name: "planning_routing",
			Condition: func(intent *TaskIntent) bool {
				return intent.Type == "planning" ||
					containsAny(intent.RequiredCapabilities, []string{"planning", "decomposition", "strategy"})
			},
			Target:   AgentPlanner,
			Priority: 7,
		},
		{
			Name: "reasoning_routing",
			Condition: func(intent *TaskIntent) bool {
				// Default fallback for general reasoning
				return true
			},
			Target:   AgentReasoner,
			Priority: 1,
		},
	}
}

// Route determines the best agent for a given intent
func (r *Router) Route(ctx context.Context, intent *TaskIntent) (AgentType, error) {
	logger.Debug("Routing intent: type=%s, confidence=%.2f", intent.Type, intent.Confidence)

	// Find the highest priority matching rule
	var selectedRule *RoutingRule
	for _, rule := range r.rules {
		if rule.Condition(intent) {
			if selectedRule == nil || rule.Priority > selectedRule.Priority {
				selectedRule = &rule
			}
		}
	}

	if selectedRule == nil {
		return "", fmt.Errorf("no routing rule matched intent")
	}

	// Verify agent is available
	if !r.registry.HasAgent(selectedRule.Target) {
		logger.Warn("Target agent %s not available, falling back to reasoner", selectedRule.Target)
		return AgentReasoner, nil
	}

	logger.Info("Routed to agent: %s (rule: %s)", selectedRule.Target, selectedRule.Name)
	return selectedRule.Target, nil
}

// AddRule adds a custom routing rule
func (r *Router) AddRule(rule RoutingRule) {
	r.rules = append(r.rules, rule)
	// Sort rules by priority (highest first)
	r.sortRulesByPriority()
}

// sortRulesByPriority sorts routing rules by priority in descending order
func (r *Router) sortRulesByPriority() {
	// Simple bubble sort for small rule sets
	for i := 0; i < len(r.rules)-1; i++ {
		for j := 0; j < len(r.rules)-i-1; j++ {
			if r.rules[j].Priority < r.rules[j+1].Priority {
				r.rules[j], r.rules[j+1] = r.rules[j+1], r.rules[j]
			}
		}
	}
}

// containsAny checks if any of the target strings are in the source slice
func containsAny(source []string, targets []string) bool {
	for _, s := range source {
		for _, t := range targets {
			if s == t {
				return true
			}
		}
	}
	return false
}
