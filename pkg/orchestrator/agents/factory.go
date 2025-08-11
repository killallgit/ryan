package agents

import (
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/tmc/langchaingo/llms"
)

// RegisterRealAgents registers all real agent implementations with the orchestrator
func RegisterRealAgents(orch *orchestrator.Orchestrator, llm llms.Model, skipPermissions bool) error {
	logger.Info("Registering real agents with orchestrator")

	// Create and register tool caller agent
	toolCaller := NewToolCallerAgent(llm, skipPermissions)
	if err := orch.RegisterAgent(orchestrator.AgentToolCaller, toolCaller); err != nil {
		return err
	}
	logger.Debug("Registered ToolCallerAgent")

	// Create and register reasoner agent
	reasoner := NewReasonerAgent(llm)
	if err := orch.RegisterAgent(orchestrator.AgentReasoner, reasoner); err != nil {
		return err
	}
	logger.Debug("Registered ReasonerAgent")

	// Create and register code generation agent
	codeGen := NewCodeGenAgent(llm)
	if err := orch.RegisterAgent(orchestrator.AgentCodeGen, codeGen); err != nil {
		return err
	}
	logger.Debug("Registered CodeGenAgent")

	// Create and register searcher agent
	searcher := NewSearcherAgent(llm, skipPermissions)
	if err := orch.RegisterAgent(orchestrator.AgentSearcher, searcher); err != nil {
		return err
	}
	logger.Debug("Registered SearcherAgent")

	// Create and register planner agent
	planner := NewPlannerAgent(llm)
	if err := orch.RegisterAgent(orchestrator.AgentPlanner, planner); err != nil {
		return err
	}
	logger.Debug("Registered PlannerAgent")

	logger.Info("Successfully registered 5 real agents")
	return nil
}
