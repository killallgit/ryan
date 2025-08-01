package tui_test

import (
	"testing"

	"github.com/killallgit/ryan/pkg/tui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestModelDownload(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Model Download Suite")
}

var _ = Describe("Model Download Events", func() {
	Describe("ModelDownloadProgressEvent", func() {
		It("should create event with correct fields", func() {
			event := tui.NewModelDownloadProgressEvent("llama3.1:8b", "downloading", 0.5)
			
			Expect(event.ModelName).To(Equal("llama3.1:8b"))
			Expect(event.Status).To(Equal("downloading"))
			Expect(event.Progress).To(Equal(0.5))
		})
	})

	Describe("ModelDownloadCompleteEvent", func() {
		It("should create event with correct model name", func() {
			event := tui.NewModelDownloadCompleteEvent("llama3.1:8b")
			
			Expect(event.ModelName).To(Equal("llama3.1:8b"))
		})
	})

	Describe("ModelDownloadErrorEvent", func() {
		It("should create event with error", func() {
			testErr := &TestError{message: "download failed"}
			event := tui.NewModelDownloadErrorEvent("llama3.1:8b", testErr)
			
			Expect(event.ModelName).To(Equal("llama3.1:8b"))
			Expect(event.Error).To(Equal(testErr))
		})
	})

	Describe("ModelNotFoundEvent", func() {
		It("should create event with model name", func() {
			event := tui.NewModelNotFoundEvent("nonexistent:model")
			
			Expect(event.ModelName).To(Equal("nonexistent:model"))
		})
	})
})

var _ = Describe("Download Modal Components", func() {
	Describe("DownloadPromptModal", func() {
		It("should create modal with default values", func() {
			modal := tui.NewDownloadPromptModal()
			
			Expect(modal.Visible).To(BeFalse())
			Expect(modal.ModelName).To(BeEmpty())
			Expect(modal.Width).To(Equal(60))
			Expect(modal.Height).To(Equal(10))
		})

		It("should show modal with model name", func() {
			modal := tui.NewDownloadPromptModal()
			modal = modal.Show("llama3.1:8b")
			
			Expect(modal.Visible).To(BeTrue())
			Expect(modal.ModelName).To(Equal("llama3.1:8b"))
		})

		It("should hide modal", func() {
			modal := tui.NewDownloadPromptModal().Show("test:model")
			modal = modal.Hide()
			
			Expect(modal.Visible).To(BeFalse())
		})
	})

	Describe("ProgressModal", func() {
		It("should create modal with default values", func() {
			modal := tui.NewProgressModal()
			
			Expect(modal.Visible).To(BeFalse())
			Expect(modal.Progress).To(Equal(0.0))
			Expect(modal.Cancellable).To(BeTrue())
		})

		It("should show modal with details", func() {
			modal := tui.NewProgressModal()
			modal = modal.Show("Downloading", "llama3.1:8b", "Starting...", true)
			
			Expect(modal.Visible).To(BeTrue())
			Expect(modal.Title).To(Equal("Downloading"))
			Expect(modal.ModelName).To(Equal("llama3.1:8b"))
			Expect(modal.Status).To(Equal("Starting..."))
			Expect(modal.Cancellable).To(BeTrue())
		})

		It("should update progress", func() {
			modal := tui.NewProgressModal()
			modal = modal.WithProgress(0.75, "Almost done...")
			
			Expect(modal.Progress).To(Equal(0.75))
			Expect(modal.Status).To(Equal("Almost done..."))
		})
	})
})

// TestError is a simple error implementation for testing
type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}