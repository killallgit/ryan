package tui

import (
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/controllers"
)

func TestVectorStoreListDisplay_Basic(t *testing.T) {
	display := NewVectorStoreListDisplay(80, 20)

	if display.Width != 80 {
		t.Errorf("Expected width 80, got %d", display.Width)
	}
	if display.Height != 20 {
		t.Errorf("Expected height 20, got %d", display.Height)
	}
	if display.Selected != 0 {
		t.Errorf("Expected selected 0, got %d", display.Selected)
	}
	if len(display.Collections) != 0 {
		t.Errorf("Expected empty collections, got %d", len(display.Collections))
	}
}

func TestVectorStoreListDisplay_WithCollections(t *testing.T) {
	collections := []controllers.CollectionInfo{
		{
			Name:          "test-collection",
			DocumentCount: 100,
			EmbedderModel: "nomic-embed-text",
			LastUpdated:   time.Now(),
		},
		{
			Name:          "another-collection",
			DocumentCount: 50,
			EmbedderModel: "text-embedding-ada-002",
			LastUpdated:   time.Now().Add(-time.Hour),
		},
	}

	display := NewVectorStoreListDisplay(80, 20).WithCollections(collections)

	if len(display.Collections) != 2 {
		t.Errorf("Expected 2 collections, got %d", len(display.Collections))
	}

	if display.Collections[0].Name != "test-collection" {
		t.Errorf("Expected first collection name 'test-collection', got '%s'", display.Collections[0].Name)
	}
}

func TestVectorStoreListDisplay_Selection(t *testing.T) {
	collections := []controllers.CollectionInfo{
		{Name: "collection1", DocumentCount: 10},
		{Name: "collection2", DocumentCount: 20},
		{Name: "collection3", DocumentCount: 30},
	}

	display := NewVectorStoreListDisplay(80, 20).WithCollections(collections)

	// Test SelectNext
	display = display.SelectNext()
	if display.Selected != 1 {
		t.Errorf("Expected selected 1, got %d", display.Selected)
	}

	// Test SelectPrevious
	display = display.SelectPrevious()
	if display.Selected != 0 {
		t.Errorf("Expected selected 0, got %d", display.Selected)
	}

	// Test boundary conditions
	display = display.SelectPrevious() // Should stay at 0
	if display.Selected != 0 {
		t.Errorf("Expected selected to stay at 0, got %d", display.Selected)
	}

	// Go to last item
	display = display.WithSelection(2)
	display = display.SelectNext() // Should stay at 2
	if display.Selected != 2 {
		t.Errorf("Expected selected to stay at 2, got %d", display.Selected)
	}
}

func TestVectorStoreListDisplay_GetSelectedCollection(t *testing.T) {
	collections := []controllers.CollectionInfo{
		{Name: "collection1", DocumentCount: 10},
		{Name: "collection2", DocumentCount: 20},
	}

	display := NewVectorStoreListDisplay(80, 20).WithCollections(collections)

	// Test valid selection
	selected := display.GetSelectedCollection()
	if selected == nil {
		t.Error("Expected selected collection, got nil")
	} else if selected.Name != "collection1" {
		t.Errorf("Expected collection1, got %s", selected.Name)
	}

	// Test with second item
	display = display.WithSelection(1)
	selected = display.GetSelectedCollection()
	if selected == nil {
		t.Error("Expected selected collection, got nil")
	} else if selected.Name != "collection2" {
		t.Errorf("Expected collection2, got %s", selected.Name)
	}

	// Test with empty display
	emptyDisplay := NewVectorStoreListDisplay(80, 20)
	selected = emptyDisplay.GetSelectedCollection()
	if selected != nil {
		t.Error("Expected nil for empty display, got collection")
	}
}

func TestVectorStoreStatsDisplay_Basic(t *testing.T) {
	display := NewVectorStoreStatsDisplay(80, 4)

	if display.Width != 80 {
		t.Errorf("Expected width 80, got %d", display.Width)
	}
	if display.Height != 4 {
		t.Errorf("Expected height 4, got %d", display.Height)
	}
}

func TestVectorStoreStatsDisplay_WithStats(t *testing.T) {
	stats := controllers.VectorStoreStats{
		TotalCollections: 3,
		TotalDocuments:   150,
		Provider:         "chromem",
		PersistenceDir:   "/tmp/vectorstore",
		IsEnabled:        true,
	}

	display := NewVectorStoreStatsDisplay(80, 4).WithStats(stats)

	if display.Stats.TotalCollections != 3 {
		t.Errorf("Expected 3 collections, got %d", display.Stats.TotalCollections)
	}
	if display.Stats.TotalDocuments != 150 {
		t.Errorf("Expected 150 documents, got %d", display.Stats.TotalDocuments)
	}
	if display.Stats.Provider != "chromem" {
		t.Errorf("Expected provider 'chromem', got '%s'", display.Stats.Provider)
	}
	if !display.Stats.IsEnabled {
		t.Error("Expected IsEnabled to be true")
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"seconds ago", now.Add(-30 * time.Second), "30s ago"},
		{"minutes ago", now.Add(-5 * time.Minute), "5m ago"},
		{"hours ago", now.Add(-2 * time.Hour), "2h ago"},
		{"days ago", now.Add(-3 * 24 * time.Hour), "3d ago"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := formatTimeAgo(test.time)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}
