package tui_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/killallgit/ryan/pkg/tui"
)

var _ = Describe("SpinnerComponent", func() {
	It("should create spinner with default properties", func() {
		spinner := tui.NewSpinnerComponent()

		Expect(spinner.IsVisible).To(BeFalse())
		Expect(spinner.Frame).To(Equal(0))
		Expect(spinner.Text).To(Equal(""))
	})

	It("should toggle visibility correctly", func() {
		spinner := tui.NewSpinnerComponent()

		visible := spinner.WithVisibility(true)
		hidden := visible.WithVisibility(false)

		Expect(visible.IsVisible).To(BeTrue())
		Expect(hidden.IsVisible).To(BeFalse())
	})

	It("should advance animation frames", func() {
		spinner := tui.NewSpinnerComponent().WithVisibility(true)

		frame1 := spinner.NextFrame()
		frame2 := frame1.NextFrame()

		Expect(frame1.Frame).To(Equal(1))
		Expect(frame2.Frame).To(Equal(2))
	})

	It("should return empty display text when not visible", func() {
		spinner := tui.NewSpinnerComponent()

		displayText := spinner.GetDisplayText()

		Expect(displayText).To(Equal(""))
	})

	It("should return formatted display text when visible", func() {
		spinner := tui.NewSpinnerComponent().WithVisibility(true)

		displayText := spinner.GetDisplayText()

		// Since we removed the text, display text should just be the spinner character
		Expect(displayText).To(Equal("ｦ")) // First character in spinnerFrames only
	})
})
