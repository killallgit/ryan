package tui

import (
	"testing"

	"github.com/rivo/tview"
)

func TestNewModal(t *testing.T) {
	app := tview.NewApplication()

	modal := NewModal(app, "Test Title", "Test message", func(result ModalResult) {
		// Callback function for testing
	})

	if modal == nil {
		t.Fatal("NewModal should return a non-nil modal")
	}

	if modal.GetResult() != ModalResultNone {
		t.Error("Initial modal result should be ModalResultNone")
	}

	// Test setting a message
	modal.SetMessage("Updated message")

	// The message should be updated (we can't easily test the actual display)
	// But we can ensure the method doesn't crash
}

func TestOllamaSetupModal(t *testing.T) {
	app := tview.NewApplication()

	// Test different issue types
	issues := []string{"not_running", "no_model", "download_model", "custom_issue"}

	for _, issue := range issues {
		modal := OllamaSetupModal(app, issue, func(result ModalResult) {
			// Callback function
		})

		if modal == nil {
			t.Errorf("OllamaSetupModal should return a non-nil modal for issue: %s", issue)
		}
	}
}

func TestModalResult(t *testing.T) {
	// Test that modal results are different
	if ModalResultNone == ModalResultConfirm {
		t.Error("ModalResultNone should be different from ModalResultConfirm")
	}

	if ModalResultConfirm == ModalResultCancel {
		t.Error("ModalResultConfirm should be different from ModalResultCancel")
	}

	if ModalResultNone == ModalResultCancel {
		t.Error("ModalResultNone should be different from ModalResultCancel")
	}
}

func TestNewInputModal(t *testing.T) {
	app := tview.NewApplication()

	modal := NewInputModal(app, "Test Title", "Test message", "placeholder", func(result ModalResult, input string) {
		// Callback function for testing
	})

	if modal == nil {
		t.Fatal("NewInputModal should return a non-nil modal")
	}

	if !modal.hasInput {
		t.Error("Input modal should have hasInput set to true")
	}

	if modal.GetResult() != ModalResultNone {
		t.Error("Initial modal result should be ModalResultNone")
	}
}

func TestOllamaURLInputModal(t *testing.T) {
	app := tview.NewApplication()

	modal := OllamaURLInputModal(app, "http://localhost:11434", func(result ModalResult, url string) {
		// Callback function for testing
	})

	if modal == nil {
		t.Fatal("OllamaURLInputModal should return a non-nil modal")
	}

	if !modal.hasInput {
		t.Error("URL input modal should have hasInput set to true")
	}
}

func TestOllamaModelDownloadModal(t *testing.T) {
	app := tview.NewApplication()

	modal := OllamaModelDownloadModal(app, "qwen3:latest", func(result ModalResult, modelName string) {
		// Callback function for testing
	})

	if modal == nil {
		t.Fatal("OllamaModelDownloadModal should return a non-nil modal")
	}

	if !modal.hasInput {
		t.Error("Model download modal should have hasInput set to true")
	}

	if !modal.hasProgress {
		t.Error("Model download modal should have hasProgress set to true")
	}
}

func TestModalProgressVisibility(t *testing.T) {
	app := tview.NewApplication()

	// Create a modal with progress capability
	modal := OllamaModelDownloadModal(app, "qwen3:latest", func(result ModalResult, modelName string) {
		// Callback function for testing
	})

	if modal == nil {
		t.Fatal("OllamaModelDownloadModal should return a non-nil modal")
	}

	// Progress bar should be created but not initially visible in layout
	if modal.progressBar == nil {
		t.Error("Progress bar should be created for download modal")
	}

	// Initially, progress bar should not be in the layout
	progressInLayout := false
	for i := 0; i < modal.Flex.GetItemCount(); i++ {
		if modal.Flex.GetItem(i) == modal.progressBar {
			progressInLayout = true
			break
		}
	}

	if progressInLayout {
		t.Error("Progress bar should not be initially visible in layout")
	}

	// After calling ShowProgress, it should be in the layout
	modal.ShowProgress()

	progressInLayoutAfter := false
	for i := 0; i < modal.Flex.GetItemCount(); i++ {
		if modal.Flex.GetItem(i) == modal.progressBar {
			progressInLayoutAfter = true
			break
		}
	}

	if !progressInLayoutAfter {
		t.Error("Progress bar should be visible in layout after ShowProgress()")
	}
}
