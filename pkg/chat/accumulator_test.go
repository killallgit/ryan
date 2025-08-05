package chat_test

import (
	"github.com/killallgit/ryan/pkg/chat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MessageAccumulator", func() {
	var accumulator *chat.MessageAccumulator

	BeforeEach(func() {
		accumulator = chat.NewMessageAccumulator()
	})

	Describe("NewMessageAccumulator", func() {
		It("should create a new accumulator", func() {
			Expect(accumulator).ToNot(BeNil())
		})
	})

	Describe("GetMessage", func() {
		It("should return false for non-existent stream", func() {
			msg, exists := accumulator.GetMessage("non-existent")
			Expect(exists).To(BeFalse())
			Expect(msg).To(BeNil())
		})
	})

	Describe("CleanupStream", func() {
		It("should not panic when cleaning up non-existent stream", func() {
			Expect(func() {
				accumulator.CleanupStream("non-existent")
			}).ToNot(Panic())
		})
	})

	Describe("GetActiveStreams", func() {
		It("should return empty list when no active streams", func() {
			streams := accumulator.GetActiveStreams()
			Expect(streams).To(BeEmpty())
		})
	})

	Describe("GetStreamStats", func() {
		It("should return false for non-existent stream stats", func() {
			stats, exists := accumulator.GetStreamStats("non-existent")
			Expect(exists).To(BeFalse())
			Expect(stats).To(Equal(chat.StreamStats{}))
		})
	})
})
