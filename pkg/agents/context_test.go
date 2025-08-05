package agents

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedMemory(t *testing.T) {
	sm := NewSharedMemory()

	t.Run("Set and Get", func(t *testing.T) {
		sm.Set("key1", "value1")
		sm.Set("key2", 42)

		value, exists := sm.Get("key1")
		assert.True(t, exists)
		assert.Equal(t, "value1", value)

		value, exists = sm.Get("key2")
		assert.True(t, exists)
		assert.Equal(t, 42, value)

		_, exists = sm.Get("nonexistent")
		assert.False(t, exists)
	})

	t.Run("GetAll", func(t *testing.T) {
		sm := NewSharedMemory()
		sm.Set("key1", "value1")
		sm.Set("key2", 42)

		data := sm.GetAll()
		assert.Len(t, data, 2)
		assert.Equal(t, "value1", data["key1"])
		assert.Equal(t, 42, data["key2"])

		// Verify it's a copy by modifying returned map
		data["key3"] = "new"
		_, exists := sm.Get("key3")
		assert.False(t, exists)
	})
}

func TestContextTree(t *testing.T) {
	ct := NewContextTree()

	t.Run("Initial State", func(t *testing.T) {
		root, exists := ct.GetNode("root")
		assert.True(t, exists)
		assert.Equal(t, "root", root.ID)
	})

	t.Run("AddNode", func(t *testing.T) {
		data := map[string]interface{}{"test": "data"}
		err := ct.AddNode("child1", "root", data)
		require.NoError(t, err)

		child, exists := ct.GetNode("child1")
		assert.True(t, exists)
		assert.Equal(t, "child1", child.ID)
		assert.Equal(t, "data", child.Data["test"])
		assert.Equal(t, "root", child.Parent.ID)
	})

	t.Run("AddNode with nonexistent parent", func(t *testing.T) {
		err := ct.AddNode("orphan", "nonexistent", nil)
		require.NoError(t, err)

		orphan, exists := ct.GetNode("orphan")
		assert.True(t, exists)
		assert.Equal(t, "root", orphan.Parent.ID) // Should default to root
	})

	t.Run("GetPath", func(t *testing.T) {
		ct := NewContextTree()
		ct.AddNode("level1", "root", nil)
		ct.AddNode("level2", "level1", nil)

		path := ct.GetPath("level2")
		require.Len(t, path, 3)
		assert.Equal(t, "root", path[0].ID)
		assert.Equal(t, "level1", path[1].ID)
		assert.Equal(t, "level2", path[2].ID)

		// Test nonexistent node
		path = ct.GetPath("nonexistent")
		assert.Nil(t, path)
	})
}

func TestContextManager(t *testing.T) {
	cm := NewContextManager()

	t.Run("CreateContext", func(t *testing.T) {
		ctx := cm.CreateContext("session1", "req1", "test prompt")
		assert.Equal(t, "session1", ctx.SessionID)
		assert.Equal(t, "req1", ctx.RequestID)
		assert.Equal(t, "test prompt", ctx.UserPrompt)
		assert.NotNil(t, ctx.SharedData)
		assert.NotNil(t, ctx.FileContext)
		assert.NotNil(t, ctx.Artifacts)
		assert.NotNil(t, ctx.Options)
	})

	t.Run("PropagateContext", func(t *testing.T) {
		from := cm.CreateContext("s1", "r1", "from")
		to := cm.CreateContext("s2", "r2", "to")

		from.SharedData["test"] = "value"
		from.Artifacts["artifact"] = "data"
		from.FileContext = append(from.FileContext, FileInfo{
			Path:         "/test/file.go",
			Size:         100,
			LastModified: time.Now(),
		})

		cm.PropagateContext(from, to, "code_review")

		// Verify propagation happened (rules may filter some data)
		assert.NotEmpty(t, to.FileContext)
		assert.Equal(t, "/test/file.go", to.FileContext[0].Path)
	})
}

func TestPropagationRules(t *testing.T) {
	t.Run("FileContextRule", func(t *testing.T) {
		rule := &FileContextRule{}

		assert.True(t, rule.ShouldApply("any_agent"))

		from := &ExecutionContext{
			FileContext: []FileInfo{{Path: "/test.go", Size: 100}},
			SharedData:  make(map[string]interface{}),
			Artifacts:   make(map[string]interface{}),
			Options:     make(map[string]interface{}),
		}
		to := &ExecutionContext{
			FileContext: []FileInfo{},
			SharedData:  make(map[string]interface{}),
			Artifacts:   make(map[string]interface{}),
			Options:     make(map[string]interface{}),
		}

		rule.Apply(from, to, "test_agent")
		require.Len(t, to.FileContext, 1)
		assert.Equal(t, "/test.go", to.FileContext[0].Path)

		// Test duplicate prevention
		rule.Apply(from, to, "test_agent")
		assert.Len(t, to.FileContext, 1) // Should still be 1
	})

	t.Run("SharedDataRule", func(t *testing.T) {
		rule := &SharedDataRule{}

		assert.True(t, rule.ShouldApply("any_agent"))

		from := &ExecutionContext{
			SharedData: map[string]interface{}{
				"analysis_result": "data",
				"internal_cache":  "secret",
				"file_list":       []string{"a", "b"},
			},
			FileContext: []FileInfo{},
			Artifacts:   make(map[string]interface{}),
			Options:     make(map[string]interface{}),
		}
		to := &ExecutionContext{
			SharedData:  make(map[string]interface{}),
			FileContext: []FileInfo{},
			Artifacts:   make(map[string]interface{}),
			Options:     make(map[string]interface{}),
		}

		rule.Apply(from, to, "code_review")
		assert.Contains(t, to.SharedData, "analysis_result")
		assert.NotContains(t, to.SharedData, "internal_cache")

		rule.Apply(from, to, "file_operations")
		assert.Contains(t, to.SharedData, "file_list")
	})

	t.Run("ArtifactsRule", func(t *testing.T) {
		rule := &ArtifactsRule{}

		assert.True(t, rule.ShouldApply("code_review"))
		assert.False(t, rule.ShouldApply("dispatcher"))

		from := &ExecutionContext{
			Artifacts: map[string]interface{}{
				"ast":    "syntax_tree",
				"report": "review_report",
			},
			SharedData:  make(map[string]interface{}),
			FileContext: []FileInfo{},
			Options:     make(map[string]interface{}),
		}
		to := &ExecutionContext{
			Artifacts:   make(map[string]interface{}),
			SharedData:  make(map[string]interface{}),
			FileContext: []FileInfo{},
			Options:     make(map[string]interface{}),
		}

		rule.Apply(from, to, "code_review")
		assert.Equal(t, "syntax_tree", to.Artifacts["ast"])
		assert.Equal(t, "review_report", to.Artifacts["report"])
	})
}

func TestShouldPropagateKey(t *testing.T) {
	tests := []struct {
		key         string
		targetAgent string
		expected    bool
	}{
		{"analysis_result", "code_review", true},
		{"ast_data", "code_review", true},
		{"file_list", "code_review", false},
		{"file_list", "file_operations", true},
		{"path_info", "file_operations", true},
		{"analysis_result", "file_operations", false},
		{"internal_cache", "any_agent", false},
		{"public_data", "any_agent", true},
	}

	for _, tt := range tests {
		t.Run(tt.key+"_"+tt.targetAgent, func(t *testing.T) {
			result := shouldPropagateKey(tt.key, tt.targetAgent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextPropagator(t *testing.T) {
	cp := NewContextPropagator()

	t.Run("Propagate with rules", func(t *testing.T) {
		from := &ExecutionContext{
			SharedData: map[string]interface{}{
				"analysis_data": "test",
				"file_info":     "path",
			},
			FileContext: []FileInfo{{Path: "/test.go"}},
			Artifacts: map[string]interface{}{
				"report": "data",
			},
			Options: make(map[string]interface{}),
		}
		to := &ExecutionContext{
			SharedData:  make(map[string]interface{}),
			FileContext: []FileInfo{},
			Artifacts:   make(map[string]interface{}),
			Options:     make(map[string]interface{}),
		}

		cp.Propagate(from, to, "code_review")

		// Verify that propagation occurred
		assert.NotEmpty(t, to.FileContext)
		assert.NotEmpty(t, to.Artifacts)
	})
}
