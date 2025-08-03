package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLangChainClient(t *testing.T) {
	client, err := NewLangChainClient("http://localhost:11434", "llama2")
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:11434", client.baseURL)
	assert.Equal(t, "llama2", client.model)
}

func TestNewLangChainStreamingClient(t *testing.T) {
	client, err := NewLangChainStreamingClient("http://localhost:11434", "llama2")
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.LangChainClient)
}

func TestLangChainClientImplementsInterface(t *testing.T) {
	client, err := NewLangChainClient("http://localhost:11434", "llama2")
	require.NoError(t, err)

	// Verify it implements ChatClient interface
	var _ ChatClient = client
}

func TestLangChainStreamingClientImplementsInterface(t *testing.T) {
	client, err := NewLangChainStreamingClient("http://localhost:11434", "llama2")
	require.NoError(t, err)

	// Verify it implements both interfaces
	var _ ChatClient = client
	var _ StreamingChatClient = client
}

func TestClientFactory(t *testing.T) {
	tests := []struct {
		name            string
		config          ClientConfig
		expectError     bool
		expectStreaming bool
	}{
		{
			name: "Original client",
			config: ClientConfig{
				BaseURL:    "http://localhost:11434",
				ClientType: ClientTypeOriginal,
			},
			expectError: false,
		},
		{
			name: "LangChain client",
			config: ClientConfig{
				BaseURL:    "http://localhost:11434",
				Model:      "llama2",
				ClientType: ClientTypeLangChain,
			},
			expectError: false,
		},
		{
			name: "LangChain streaming client",
			config: ClientConfig{
				BaseURL:      "http://localhost:11434",
				Model:        "llama2",
				ClientType:   ClientTypeLangChain,
				UseStreaming: true,
			},
			expectError:     false,
			expectStreaming: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectStreaming {
				client, err := NewStreamingChatClient(tt.config)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, client)
				}
			} else {
				client, err := NewChatClient(tt.config)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, client)
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "http://localhost:11434", config.BaseURL)
	assert.Equal(t, ClientTypeOriginal, config.ClientType)
	assert.False(t, config.UseStreaming)
}

func TestLangChainConfig(t *testing.T) {
	config := LangChainConfig("http://custom:11434", "custom-model")
	assert.Equal(t, "http://custom:11434", config.BaseURL)
	assert.Equal(t, "custom-model", config.Model)
	assert.Equal(t, ClientTypeLangChain, config.ClientType)
	assert.False(t, config.UseStreaming)
}

func TestLangChainStreamingConfig(t *testing.T) {
	config := LangChainStreamingConfig("http://custom:11434", "custom-model")
	assert.Equal(t, "http://custom:11434", config.BaseURL)
	assert.Equal(t, "custom-model", config.Model)
	assert.Equal(t, ClientTypeLangChain, config.ClientType)
	assert.True(t, config.UseStreaming)
}
