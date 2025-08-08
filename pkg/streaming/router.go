package streaming

import (
	"strings"
)

// Router determines which provider/agent to use for a given prompt
type Router struct {
	registry     *Registry
	rules        []RoutingRule
	defaultRoute string
}

// RoutingRule defines a pattern-based routing rule
type RoutingRule struct {
	Pattern  string // Substring or regex pattern to match
	Provider string // Provider ID to route to
	Priority int    // Higher priority rules are evaluated first
}

// NewRouter creates a new router with a default provider
func NewRouter(registry *Registry, defaultProvider string) *Router {
	return &Router{
		registry:     registry,
		rules:        []RoutingRule{},
		defaultRoute: defaultProvider,
	}
}

// AddRule adds a routing rule to the router
func (r *Router) AddRule(pattern string, provider string, priority int) {
	r.rules = append(r.rules, RoutingRule{
		Pattern:  pattern,
		Provider: provider,
		Priority: priority,
	})

	// Sort rules by priority (highest first)
	for i := len(r.rules) - 1; i > 0; i-- {
		if r.rules[i].Priority > r.rules[i-1].Priority {
			r.rules[i], r.rules[i-1] = r.rules[i-1], r.rules[i]
		}
	}
}

// Route determines which provider to use based on the prompt
func (r *Router) Route(prompt string) string {
	// Check rules in priority order
	promptLower := strings.ToLower(prompt)

	for _, rule := range r.rules {
		if strings.Contains(promptLower, strings.ToLower(rule.Pattern)) {
			// Verify provider exists
			if _, exists := r.registry.Get(rule.Provider); exists {
				return rule.Provider
			}
		}
	}

	// Return default if no rules match
	return r.defaultRoute
}

// RouteWithMetadata provides routing decision with additional context
func (r *Router) RouteWithMetadata(prompt string) RoutingDecision {
	provider := r.Route(prompt)

	// Find which rule matched
	var matchedRule *RoutingRule
	promptLower := strings.ToLower(prompt)

	for _, rule := range r.rules {
		if strings.Contains(promptLower, strings.ToLower(rule.Pattern)) {
			if _, exists := r.registry.Get(rule.Provider); exists {
				matchedRule = &rule
				break
			}
		}
	}

	return RoutingDecision{
		Provider:    provider,
		MatchedRule: matchedRule,
		IsDefault:   matchedRule == nil,
	}
}

// RoutingDecision contains the routing result with metadata
type RoutingDecision struct {
	Provider    string
	MatchedRule *RoutingRule
	IsDefault   bool
}
