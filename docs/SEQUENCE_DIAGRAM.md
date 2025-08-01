# Ryan - Core Component Interactions

## High-Level Sequence Diagram

```mermaid
sequenceDiagram
    participant User
    participant TUI as TUI Layer<br/>(model_view.go, chat_view.go)
    participant Controller as Controller Layer<br/>(chat.go, models.go)
    participant Client as Ollama Client<br/>(client.go)
    participant Ollama as Ollama API

    Note over User, Ollama: Chat Message Flow
    User->>TUI: Type message & press Enter
    TUI->>Controller: SendMessage(content)
    Controller->>Client: Chat(message, history)
    Client->>Ollama: POST /api/chat
    Ollama-->>Client: Response with message
    Client-->>Controller: chat.Message
    Controller-->>TUI: MessageResponseEvent
    TUI-->>User: Display response

    Note over User, Ollama: Model Management Flow
    User->>TUI: Press Tab (switch to models view)
    TUI->>Controller: Tags() (refresh models)
    Controller->>Client: Tags()
    Client->>Ollama: GET /api/tags
    Ollama-->>Client: Available models list
    Client-->>Controller: TagsResponse
    Controller-->>TUI: ModelListUpdateEvent
    TUI-->>User: Display model list

    Note over User, Ollama: Model Deletion Flow
    User->>TUI: Select model & press Ctrl-D
    TUI->>TUI: Show confirmation modal
    User->>TUI: Confirm deletion
    TUI->>Controller: Delete(modelName)
    Controller->>Client: Delete(modelName)
    Client->>Ollama: DELETE /api/delete
    Ollama-->>Client: Success response
    Client-->>Controller: Success
    Controller-->>TUI: ModelDeletedEvent
    TUI-->>User: Refresh model list
```

## Key Components

| Component | Responsibility |
|-----------|---------------|
| **TUI Layer** | User interface, event handling, rendering |
| **Controller Layer** | Business logic orchestration, state management |
| **Ollama Client** | HTTP API communication, error handling |
| **Ollama API** | AI model serving, chat completion |

## Event Flow Patterns

- **Non-blocking UI**: All API calls run in goroutines with event-based updates
- **Event System**: Components communicate via tcell events (MessageResponseEvent, ModelListUpdateEvent, etc.)
- **Functional Design**: Immutable data structures, pure functions where possible