package ollama_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/killallgit/ryan/pkg/ollama"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOllama(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ollama Suite")
}

var _ = Describe("Client", func() {
	var (
		client *ollama.Client
		server *httptest.Server
	)

	BeforeEach(func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/tags":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"models": [
						{
							"name": "llama3.1:8b",
							"model": "llama3.1:8b", 
							"size": 4661211808,
							"digest": "abc123",
							"details": {
								"parent_model": "",
								"format": "gguf",
								"family": "llama",
								"families": ["llama"],
								"parameter_size": "8.0B",
								"quantization_level": "Q4_0"
							},
							"expires_at": "2024-06-04T14:38:31.83753-07:00",
							"size_vram": 4661211808
						}
					]
				}`))
			case "/api/ps":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"models": [
						{
							"name": "mistral:latest",
							"model": "mistral:latest",
							"size": 5137025024,
							"digest": "def456",
							"details": {
								"parent_model": "",
								"format": "gguf", 
								"family": "llama",
								"families": ["llama"],
								"parameter_size": "7.2B",
								"quantization_level": "Q4_0"
							},
							"expires_at": "2024-06-04T14:38:31.83753-07:00",
							"size_vram": 5137025024
						}
					]
				}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		client = ollama.NewClient(server.URL)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Tags", func() {
		It("should return list of available models", func() {
			response, err := client.Tags()

			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Models).To(HaveLen(1))

			model := response.Models[0]
			Expect(model.Name).To(Equal("llama3.1:8b"))
			Expect(model.Size).To(Equal(int64(4661211808)))
			Expect(model.Details.ParameterSize).To(Equal("8.0B"))
			Expect(model.Details.QuantizationLevel).To(Equal("Q4_0"))
		})
	})

	Describe("Ps", func() {
		It("should return list of running models", func() {
			response, err := client.Ps()

			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Models).To(HaveLen(1))

			model := response.Models[0]
			Expect(model.Name).To(Equal("mistral:latest"))
			Expect(model.Size).To(Equal(int64(5137025024)))
			Expect(model.Details.ParameterSize).To(Equal("7.2B"))
		})
	})

	Describe("Error handling", func() {
		BeforeEach(func() {
			client = ollama.NewClient("http://invalid-url")
		})

		It("should handle connection errors for Tags", func() {
			_, err := client.Tags()
			Expect(err).To(HaveOccurred())
		})

		It("should handle connection errors for Ps", func() {
			_, err := client.Ps()
			Expect(err).To(HaveOccurred())
		})
	})
})
