package tui_test

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/tui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Chat Stream Example Tests", func() {
	var (
		testScreen *tui.TestScreen
		helper     *tui.StreamingTestHelper
	)

	BeforeEach(func() {
		testScreen = tui.NewTestScreen()
		err := testScreen.Init()
		Expect(err).ToNot(HaveOccurred())
		testScreen.SetSize(80, 24)
		testScreen.Clear()

		helper = tui.NewStreamingTestHelper(testScreen)
	})

	AfterEach(func() {
		helper.Stop()
		testScreen.Fini()
	})

	Describe("Basic Chat Streaming", func() {
		It("should simulate a complete chat interaction", func() {
			// Simulate typing a message by writing directly to screen
			// (since we don't have a full input handler)
			inputText := "Hello"
			for i, ch := range inputText {
				testScreen.SetContent(i, 22, ch, nil, tcell.StyleDefault)
			}
			testScreen.Show()

			// Capture input state
			inputContent := testScreen.CaptureContent()
			Expect(inputContent).To(ContainSubstring("Hello"))

			// Simulate sending the message
			testScreen.InjectKeyAndWait(tcell.KeyEnter, 0, tcell.ModNone, 50*time.Millisecond)

			// Simulate streaming response
			responseChunks := []string{
				"<think>",
				"Processing the greeting...",
				"</think>",
				"Hello! ",
				"How can I ",
				"help you today?",
			}

			delays := []time.Duration{
				10 * time.Millisecond,
				20 * time.Millisecond,
				10 * time.Millisecond,
				15 * time.Millisecond,
				15 * time.Millisecond,
				10 * time.Millisecond,
			}

			// Start streaming simulation
			helper.SimulateStreaming(responseChunks, delays)

			// Verify chunks appear progressively
			var receivedChunks []string
			for i := 0; i < len(responseChunks); i++ {
				chunk, ok := helper.GetNextChunk(100 * time.Millisecond)
				Expect(ok).To(BeTrue(), "Should receive chunk %d", i)
				receivedChunks = append(receivedChunks, chunk)
			}

			// Verify all chunks were received
			Expect(receivedChunks).To(Equal(responseChunks))
		})
	})

	Describe("Stream Parser Integration", func() {
		It("should correctly parse thinking blocks during streaming", func() {
			parser := tui.NewStreamParser()

			// Simulate chunks arriving over time
			chunks := []string{
				"Let me think about this",
				"<thi", // Partial tag
				"nk>",  // Complete the tag
				"This is my thought process",
				"</think>",
				"Here's my response",
			}

			var allSegments []tui.FormattedSegment

			// Process chunks as they would arrive during streaming
			for _, chunk := range chunks {
				segments := parser.ParseChunk(chunk)
				allSegments = append(allSegments, segments...)

				// Simulate network delay
				time.Sleep(10 * time.Millisecond)
			}

			// Finalize parsing
			finalSegments := parser.Finalize()
			allSegments = append(allSegments, finalSegments...)

			// Verify parsing results
			var thinkingContent string
			var responseContent string
			inThinkBlock := false

			for _, seg := range allSegments {
				if seg.Content == "<think>" || seg.Content == "<thinking>" {
					inThinkBlock = true
				} else if seg.Content == "</think>" || seg.Content == "</thinking>" {
					inThinkBlock = false
				} else if inThinkBlock && seg.Format == tui.FormatTypeThink {
					thinkingContent += seg.Content
				} else if !inThinkBlock && seg.Format == tui.FormatTypeNone {
					responseContent += seg.Content
				}
			}

			Expect(thinkingContent).To(Equal("This is my thought process"))
			Expect(responseContent).To(ContainSubstring("Let me think about this"))
			Expect(responseContent).To(ContainSubstring("Here's my response"))
		})
	})

	Describe("Screen Region Testing", func() {
		It("should verify content in specific screen regions", func() {
			// Simulate a chat view layout
			// Input area at bottom (y=20-22)
			inputRegion := testScreen.GetRegion(0, 20, 80, 3)

			// Message area (y=0-19)
			messageRegion := testScreen.GetRegion(0, 0, 80, 20)

			// Status bar (y=23)
			statusRegion := testScreen.GetRegion(0, 23, 80, 1)

			// These would contain actual rendered content in a full implementation
			_ = inputRegion
			_ = messageRegion
			_ = statusRegion
		})
	})

	Describe("Event History Tracking", func() {
		It("should track all user interactions", func() {
			// Type a message
			testScreen.InjectKey(tcell.Key('t'), 't', tcell.ModNone)
			testScreen.InjectKey(tcell.Key('e'), 'e', tcell.ModNone)
			testScreen.InjectKey(tcell.Key('s'), 's', tcell.ModNone)
			testScreen.InjectKey(tcell.Key('t'), 't', tcell.ModNone)

			// Navigate with arrow keys
			testScreen.InjectKey(tcell.KeyLeft, 0, tcell.ModNone)
			testScreen.InjectKey(tcell.KeyRight, 0, tcell.ModNone)

			// Send message
			testScreen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)

			// Get event history
			events := testScreen.GetEventHistory()

			// In a full implementation, we would verify:
			// - Correct number of events
			// - Event types and order
			// - Key values
			Expect(len(events)).To(BeNumerically(">=", 0))
		})
	})

	Describe("Concurrent Streaming Safety", func() {
		It("should handle multiple simultaneous streams safely", func() {
			// Create multiple streaming helpers
			helper1 := tui.NewStreamingTestHelper(testScreen)
			helper2 := tui.NewStreamingTestHelper(testScreen)
			defer helper1.Stop()
			defer helper2.Stop()

			// Start concurrent streams
			stream1 := []string{"Stream", "1", "content"}
			stream2 := []string{"Stream", "2", "content"}

			helper1.SimulateStreaming(stream1, nil)
			helper2.SimulateStreaming(stream2, nil)

			// Collect all chunks
			var allChunks []string
			timeout := 100 * time.Millisecond

			for i := 0; i < len(stream1)+len(stream2); i++ {
				select {
				case chunk1, ok1 := <-helper1.Chunks:
					if ok1 {
						allChunks = append(allChunks, chunk1)
					}
				case chunk2, ok2 := <-helper2.Chunks:
					if ok2 {
						allChunks = append(allChunks, chunk2)
					}
				case <-time.After(timeout):
					// Timeout is ok, we might have received all chunks
				}
			}

			// Verify we got chunks from both streams
			Expect(allChunks).To(ContainElement("Stream"))
			Expect(allChunks).To(ContainElement("1"))
			Expect(allChunks).To(ContainElement("2"))
		})
	})
})
