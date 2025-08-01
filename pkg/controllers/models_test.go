package controllers_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/ollama"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controllers Suite")
}

type MockOllamaClient struct {
	mock.Mock
}

func (m *MockOllamaClient) Tags() (*ollama.TagsResponse, error) {
	args := m.Called()
	return args.Get(0).(*ollama.TagsResponse), args.Error(1)
}

func (m *MockOllamaClient) Ps() (*ollama.PsResponse, error) {
	args := m.Called()
	return args.Get(0).(*ollama.PsResponse), args.Error(1)
}

func (m *MockOllamaClient) Pull(modelName string) error {
	args := m.Called(modelName)
	return args.Error(0)
}

func (m *MockOllamaClient) Delete(modelName string) error {
	args := m.Called(modelName)
	return args.Error(0)
}

var _ = Describe("ModelsController", func() {
	var (
		mockClient *MockOllamaClient
		controller *controllers.ModelsController
		buffer     *bytes.Buffer
	)

	BeforeEach(func() {
		mockClient = &MockOllamaClient{}
		controller = controllers.NewModelsController(mockClient)
		buffer = &bytes.Buffer{}
	})

	Describe("ListModels", func() {
		Context("when models are available", func() {
			BeforeEach(func() {
				response := &ollama.TagsResponse{
					Models: []ollama.Model{
						{
							Name: "llama3.1:8b",
							Size: 4661211808, // ~4.3GB
							Details: ollama.Details{
								ParameterSize:     "8.0B",
								QuantizationLevel: "Q4_0",
							},
						},
						{
							Name: "mistral:latest",
							Size: 5137025024, // ~4.8GB
							Details: ollama.Details{
								ParameterSize:     "7.2B",
								QuantizationLevel: "Q4_0",
							},
						},
					},
				}
				mockClient.On("Tags").Return(response, nil)
			})

			It("should format and display models correctly", func() {
				err := controller.ListModels(buffer)

				Expect(err).ToNot(HaveOccurred())
				output := buffer.String()
				Expect(output).To(ContainSubstring("NAME"))
				Expect(output).To(ContainSubstring("SIZE"))
				Expect(output).To(ContainSubstring("PARAMETER SIZE"))
				Expect(output).To(ContainSubstring("QUANTIZATION"))
				Expect(output).To(ContainSubstring("llama3.1:8b"))
				Expect(output).To(ContainSubstring("4.3GB"))
				Expect(output).To(ContainSubstring("8.0B"))
				Expect(output).To(ContainSubstring("Q4_0"))
				Expect(output).To(ContainSubstring("mistral:latest"))
				mockClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when no models are available", func() {
			BeforeEach(func() {
				response := &ollama.TagsResponse{
					Models: []ollama.Model{},
				}
				mockClient.On("Tags").Return(response, nil)
			})

			It("should display no models message", func() {
				err := controller.ListModels(buffer)

				Expect(err).ToNot(HaveOccurred())
				Expect(buffer.String()).To(Equal("No models found\n"))
				mockClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when client returns an error", func() {
			BeforeEach(func() {
				mockClient.On("Tags").Return((*ollama.TagsResponse)(nil), errors.New("connection failed"))
			})

			It("should return wrapped error", func() {
				err := controller.ListModels(buffer)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to list models"))
				Expect(err.Error()).To(ContainSubstring("connection failed"))
				mockClient.AssertExpectations(GinkgoT())
			})
		})
	})
})
