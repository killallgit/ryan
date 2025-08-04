package tui_test

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/tui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContextTreeView", func() {
	var (
		tree     *chat.ContextTree
		treeView *tui.ContextTreeView
		screen   tcell.SimulationScreen
	)

	BeforeEach(func() {
		// Create a test context tree
		tree = &chat.ContextTree{
			RootContextID: "root",
			Contexts: map[string]*chat.Context{
				"root": {
					ID:       "root",
					ParentID: nil,
					Created:  time.Now(),
				},
				"branch1": {
					ID:          "branch1",
					ParentID:    &[]string{"root"}[0],
					Created:     time.Now().Add(1 * time.Minute),
					BranchPoint: &[]string{"msg1"}[0],
				},
				"branch2": {
					ID:          "branch2",
					ParentID:    &[]string{"root"}[0],
					Created:     time.Now().Add(2 * time.Minute),
					BranchPoint: &[]string{"msg2"}[0],
				},
			},
			Messages: map[string]*chat.Message{
				"msg1": {
					ID:        "msg1",
					ContextID: "root",
					Content:   "First message",
					Role:      chat.RoleUser,
				},
				"msg2": {
					ID:        "msg2",
					ContextID: "root",
					Content:   "Second message",
					Role:      chat.RoleAssistant,
				},
				"msg3": {
					ID:        "msg3",
					ContextID: "branch1",
					Content:   "Branch 1 message",
					Role:      chat.RoleUser,
				},
			},
			ParentIndex: map[string][]string{
				"root": {"branch1", "branch2"},
			},
			ChildIndex: map[string]string{
				"branch1": "root",
				"branch2": "root",
			},
			ActiveContext: "root",
		}

		// Create simulation screen
		screen = tcell.NewSimulationScreen("UTF-8")
		err := screen.Init()
		Expect(err).ToNot(HaveOccurred())
		screen.SetSize(80, 25)

		// Create tree view
		treeView = tui.NewContextTreeView(tree, 30, 20)
		treeView = treeView.WithVisibility(true)
	})

	AfterEach(func() {
		screen.Fini()
	})

	Describe("Initialization", func() {
		It("should create a new context tree view", func() {
			Expect(treeView).ToNot(BeNil())
			Expect(treeView.GetSelectedContext()).To(Equal("root"))
			Expect(treeView.IsVisible()).To(BeTrue())
		})

		It("should start with root context selected", func() {
			Expect(treeView.GetSelectedContext()).To(Equal(tree.ActiveContext))
		})
	})

	Describe("Navigation", func() {
		It("should navigate down through contexts", func() {
			// Expand root to see children
			treeView = treeView.ToggleExpanded("root")

			// Navigate down
			treeView = treeView.NavigateDown()
			Expect(treeView.GetSelectedContext()).To(Equal("branch1"))

			treeView = treeView.NavigateDown()
			Expect(treeView.GetSelectedContext()).To(Equal("branch2"))
		})

		It("should navigate up through contexts", func() {
			// Start at branch2
			treeView = treeView.SelectNode("branch2")
			Expect(treeView.GetSelectedContext()).To(Equal("branch2"))

			// Navigate up
			treeView = treeView.NavigateUp()
			Expect(treeView.GetSelectedContext()).To(Equal("branch1"))

			treeView = treeView.NavigateUp()
			Expect(treeView.GetSelectedContext()).To(Equal("root"))
		})

		It("should navigate to parent context", func() {
			treeView = treeView.SelectNode("branch1")
			treeView = treeView.NavigateToParent()
			Expect(treeView.GetSelectedContext()).To(Equal("root"))
		})

		It("should navigate to child context", func() {
			treeView = treeView.ToggleExpanded("root")
			treeView = treeView.NavigateToChild()
			Expect(treeView.GetSelectedContext()).To(Equal("branch1"))
		})
	})

	Describe("Visibility", func() {
		It("should toggle visibility", func() {
			Expect(treeView.IsVisible()).To(BeTrue())

			treeView = treeView.Toggle()
			Expect(treeView.IsVisible()).To(BeFalse())

			treeView = treeView.Toggle()
			Expect(treeView.IsVisible()).To(BeTrue())
		})

		It("should hide on escape key", func() {
			ev := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
			handled := treeView.HandleKeyEvent(ev)
			Expect(handled).To(BeTrue())
			Expect(treeView.IsVisible()).To(BeFalse())
		})
	})

	Describe("Expansion", func() {
		It("should toggle node expansion", func() {
			// Initially not expanded
			ev := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
			handled := treeView.HandleKeyEvent(ev)
			Expect(handled).To(BeTrue())

			// Should now be expanded (implementation would check internal state)
			// Navigate down should now reach branch1
			treeView = treeView.NavigateDown()
			Expect(treeView.GetSelectedContext()).To(Equal("branch1"))
		})
	})

	Describe("Rendering", func() {
		It("should render without errors", func() {
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 25}

			// Should not panic
			Expect(func() {
				treeView.Render(screen, area)
				screen.Show()
			}).ToNot(Panic())
		})

		It("should render with different positions", func() {
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 25}

			positions := []tui.ContextTreePosition{
				tui.TreePositionRight,
				tui.TreePositionLeft,
				tui.TreePositionBottom,
				tui.TreePositionFloat,
			}

			for _, pos := range positions {
				treeView = treeView.WithPosition(pos)
				Expect(func() {
					treeView.Render(screen, area)
					screen.Show()
				}).ToNot(Panic())
			}
		})
	})

	Describe("Key Handling", func() {
		It("should handle arrow keys for navigation", func() {
			// Expand root
			treeView = treeView.ToggleExpanded("root")

			// Test down arrow
			ev := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			handled := treeView.HandleKeyEvent(ev)
			Expect(handled).To(BeTrue())
			Expect(treeView.GetSelectedContext()).To(Equal("branch1"))

			// Test up arrow
			ev = tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			handled = treeView.HandleKeyEvent(ev)
			Expect(handled).To(BeTrue())
			Expect(treeView.GetSelectedContext()).To(Equal("root"))

			// Test right arrow (expand)
			ev = tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
			handled = treeView.HandleKeyEvent(ev)
			Expect(handled).To(BeTrue())

			// Test left arrow (collapse/parent)
			ev = tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
			handled = treeView.HandleKeyEvent(ev)
			Expect(handled).To(BeTrue())
		})

		It("should not handle keys when invisible", func() {
			treeView = treeView.WithVisibility(false)

			ev := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			handled := treeView.HandleKeyEvent(ev)
			Expect(handled).To(BeFalse())
		})
	})

	Describe("Context Selection", func() {
		It("should select valid context nodes", func() {
			treeView = treeView.SelectNode("branch1")
			Expect(treeView.GetSelectedContext()).To(Equal("branch1"))

			treeView = treeView.SelectNode("branch2")
			Expect(treeView.GetSelectedContext()).To(Equal("branch2"))
		})

		It("should not select invalid context nodes", func() {
			original := treeView.GetSelectedContext()
			treeView = treeView.SelectNode("invalid")
			Expect(treeView.GetSelectedContext()).To(Equal(original))
		})
	})

	Describe("Size Updates", func() {
		It("should update size", func() {
			treeView = treeView.WithSize(40, 30)
			// Size would be reflected in rendering
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 40}
			Expect(func() {
				treeView.Render(screen, area)
			}).ToNot(Panic())
		})
	})
})
