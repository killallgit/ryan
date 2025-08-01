package tui_test

import (
	"github.com/killallgit/ryan/pkg/tui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rect", func() {
	Describe("NewRect", func() {
		It("should create rect with specified dimensions", func() {
			rect := tui.NewRect(10, 20, 30, 40)
			
			Expect(rect.X).To(Equal(10))
			Expect(rect.Y).To(Equal(20))
			Expect(rect.Width).To(Equal(30))
			Expect(rect.Height).To(Equal(40))
		})
	})

	Describe("Right", func() {
		It("should return right edge coordinate", func() {
			rect := tui.NewRect(10, 20, 30, 40)
			
			Expect(rect.Right()).To(Equal(40)) // 10 + 30
		})
	})

	Describe("Bottom", func() {
		It("should return bottom edge coordinate", func() {
			rect := tui.NewRect(10, 20, 30, 40)
			
			Expect(rect.Bottom()).To(Equal(60)) // 20 + 40
		})
	})

	Describe("Contains", func() {
		It("should return true for points inside rect", func() {
			rect := tui.NewRect(10, 20, 30, 40)
			
			Expect(rect.Contains(10, 20)).To(BeTrue())   // Top-left corner
			Expect(rect.Contains(25, 30)).To(BeTrue())   // Inside
			Expect(rect.Contains(39, 59)).To(BeTrue())   // Bottom-right inside
		})

		It("should return false for points outside rect", func() {
			rect := tui.NewRect(10, 20, 30, 40)
			
			Expect(rect.Contains(9, 20)).To(BeFalse())   // Left of rect
			Expect(rect.Contains(10, 19)).To(BeFalse())  // Above rect
			Expect(rect.Contains(40, 30)).To(BeFalse())  // Right of rect
			Expect(rect.Contains(25, 60)).To(BeFalse())  // Below rect
		})
	})

	Describe("Intersects", func() {
		It("should return true for overlapping rects", func() {
			rect1 := tui.NewRect(10, 10, 20, 20)
			rect2 := tui.NewRect(15, 15, 20, 20)
			
			Expect(rect1.Intersects(rect2)).To(BeTrue())
			Expect(rect2.Intersects(rect1)).To(BeTrue())
		})

		It("should return false for non-overlapping rects", func() {
			rect1 := tui.NewRect(10, 10, 20, 20)
			rect2 := tui.NewRect(40, 40, 20, 20)
			
			Expect(rect1.Intersects(rect2)).To(BeFalse())
			Expect(rect2.Intersects(rect1)).To(BeFalse())
		})
	})
})

var _ = Describe("Layout", func() {
	Describe("NewLayout", func() {
		It("should create layout with specified dimensions", func() {
			layout := tui.NewLayout(100, 50)
			
			Expect(layout.ScreenWidth).To(Equal(100))
			Expect(layout.ScreenHeight).To(Equal(50))
		})
	})

	Describe("CalculateAreas", func() {
		It("should divide screen into message, alert, input, and status areas", func() {
			layout := tui.NewLayout(100, 50)
			
			messageArea, alertArea, inputArea, statusArea := layout.CalculateAreas()
			
			// Status bar: 1 line (full width, no padding)
			Expect(statusArea.Height).To(Equal(1))
			Expect(statusArea.Y).To(Equal(49)) // Bottom of screen
			Expect(statusArea.Width).To(Equal(100))
			Expect(statusArea.X).To(Equal(0))
			
			// Input area: 3 lines (with padding)
			Expect(inputArea.Height).To(Equal(3))
			Expect(inputArea.Y).To(Equal(46)) // Above status
			Expect(inputArea.Width).To(Equal(98)) // 100 - 2 padding
			Expect(inputArea.X).To(Equal(1)) // 1 character padding
			
			// Alert area: 1 line (with padding)
			Expect(alertArea.Height).To(Equal(1))
			Expect(alertArea.Y).To(Equal(45)) // Above input
			Expect(alertArea.Width).To(Equal(98)) // 100 - 2 padding
			Expect(alertArea.X).To(Equal(1)) // 1 character padding
			
			// Message area: remaining space (with padding)
			Expect(messageArea.Height).To(Equal(45)) // 50 - 3 - 1 - 1
			Expect(messageArea.Y).To(Equal(0))
			Expect(messageArea.Width).To(Equal(98)) // 100 - 2 padding
			Expect(messageArea.X).To(Equal(1)) // 1 character padding
		})

		It("should handle minimum dimensions gracefully", func() {
			layout := tui.NewLayout(10, 5)
			
			messageArea, alertArea, inputArea, statusArea := layout.CalculateAreas()
			
			Expect(messageArea.Height).To(Equal(1)) // Minimum 1 line
			Expect(alertArea.Height).To(Equal(1))
			Expect(inputArea.Height).To(Equal(3))
			Expect(statusArea.Height).To(Equal(1))
		})
	})
})

var _ = Describe("WrapText", func() {
	It("should return single line for text shorter than width", func() {
		lines := tui.WrapText("Hello", 10)
		
		Expect(lines).To(HaveLen(1))
		Expect(lines[0]).To(Equal("Hello"))
	})

	It("should wrap text at word boundaries when possible", func() {
		lines := tui.WrapText("Hello world this is a test", 10)
		
		Expect(lines).To(HaveLen(4))
		Expect(lines[0]).To(Equal("Hello"))
		Expect(lines[1]).To(Equal("world"))
		Expect(lines[2]).To(Equal("this is a"))
		Expect(lines[3]).To(Equal("test"))
	})

	It("should break long words when necessary", func() {
		lines := tui.WrapText("verylongwordthatcannotfitononelinealone", 10)
		
		Expect(lines).To(HaveLen(4))
		Expect(lines[0]).To(HaveLen(10))
		Expect(lines[1]).To(HaveLen(10))
		Expect(lines[2]).To(HaveLen(10))
		Expect(lines[3]).To(HaveLen(9)) // Remaining characters
	})

	It("should handle zero or negative width", func() {
		lines := tui.WrapText("Hello", 0)
		Expect(lines).To(HaveLen(0))
		
		lines = tui.WrapText("Hello", -5)
		Expect(lines).To(HaveLen(0))
	})

	It("should handle empty text", func() {
		lines := tui.WrapText("", 10)
		Expect(lines).To(HaveLen(0))
	})
})

var _ = Describe("CalculateVisibleLines", func() {
	BeforeEach(func() {
		// Helper function will be tested with sample data
	})

	It("should return subset of lines based on scroll and height", func() {
		allLines := []string{"Line 1", "Line 2", "Line 3", "Line 4", "Line 5"}
		
		visibleLines, startLine := tui.CalculateVisibleLines(allLines, 3, 1)
		
		Expect(visibleLines).To(Equal([]string{"Line 2", "Line 3", "Line 4"}))
		Expect(startLine).To(Equal(1))
	})

	It("should handle scroll beyond available lines", func() {
		allLines := []string{"Line 1", "Line 2", "Line 3"}
		
		visibleLines, startLine := tui.CalculateVisibleLines(allLines, 3, 10)
		
		Expect(visibleLines).To(Equal([]string{"Line 3"}))
		Expect(startLine).To(Equal(2)) // Clamped to last line
	})

	It("should handle negative scroll", func() {
		allLines := []string{"Line 1", "Line 2", "Line 3"}
		
		visibleLines, startLine := tui.CalculateVisibleLines(allLines, 2, -5)
		
		Expect(visibleLines).To(Equal([]string{"Line 1", "Line 2"}))
		Expect(startLine).To(Equal(0)) // Clamped to 0
	})

	It("should handle zero height", func() {
		allLines := []string{"Line 1", "Line 2", "Line 3"}
		
		visibleLines, startLine := tui.CalculateVisibleLines(allLines, 0, 0)
		
		Expect(visibleLines).To(HaveLen(0))
		Expect(startLine).To(Equal(0))
	})

	It("should handle empty lines", func() {
		allLines := []string{}
		
		visibleLines, startLine := tui.CalculateVisibleLines(allLines, 5, 0)
		
		Expect(visibleLines).To(HaveLen(0))
		Expect(startLine).To(Equal(0))
	})
})