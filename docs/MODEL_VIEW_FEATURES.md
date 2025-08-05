# Model View Features

The enhanced model view now provides comprehensive model management with the following features:

## 🔑 Keyboard Shortcuts

- **Enter**: Select and switch to the highlighted model
- **+**: Open download modal to pull new models from Ollama Hub
- **-** or **d**: Delete the selected model (with confirmation)
- **r**: Refresh the model list from Ollama
- **Esc**: Return to previous view

## 📋 Model Information Display

### Columns Shown:
1. **Name**: Full model name (e.g., `llama3.1:8b`)
2. **Size**: Actual model size on disk (e.g., `3.8 GB`)
3. **Parameters**: Model parameter count (e.g., `8B`, `7B`)
4. **Quantization**: Quantization level (e.g., `Q4_0`, `F16`)
5. **Tools**: Tool calling compatibility with color coding
6. **Status**: Current model indicator

### Tool Compatibility Colors:
- 🟢 **Green "Excellent ✓"**: Best tool support (llama3.1:8b, qwen2.5:7b+)
- 🔵 **Cyan "Good ✓"**: Reliable tool support (mistral:7b, qwen2.5:3b)
- 🟡 **Yellow "Basic"**: Limited tool support (qwen2.5:0.5b)
- 🔴 **Red "None"**: No tool calling support (gemma, phi)
- ⚪ **Gray "Unknown"**: Untested models (inferred from family)

## 📥 Model Download

### Download Process:
1. Press **+** to open download modal
2. Enter model name (examples provided)
3. Real-time progress bar with percentage
4. Automatic refresh when complete
5. Modal auto-closes on completion

### Supported Model Names:
- `llama3.1:8b` (recommended for tools)
- `qwen2.5:7b` (excellent for coding)
- `mistral:7b` (solid general purpose)
- `gemma:2b` (lightweight, no tools)
- Any model from Ollama Hub

### Progress Tracking:
- Visual progress bar with percentage
- Status updates (downloading, extracting, etc.)
- Cancel button available (returns to main view)
- Error handling with clear messages

## 🗑️ Model Deletion

### Safety Features:
- Confirmation modal with warning
- Cannot delete currently selected model
- Clear warning about irreversible action
- Styled deletion confirmation

### Process:
1. Select model and press **-** or **d**
2. Confirm deletion in warning dialog
3. Model removed from Ollama
4. List automatically refreshed

## 🎨 Visual Enhancements

- **Current Model**: Highlighted in green
- **Tool Compatibility**: Color-coded for easy identification
- **Consistent Theming**: Matches application color scheme
- **Status Bar**: Shows available actions and tool legend
- **Empty State**: Helpful message when no models found

## 🔧 Technical Features

- **Real Ollama Integration**: Fetches live data from Ollama server
- **Thread Safety**: All operations properly synchronized
- **Error Handling**: Graceful fallbacks and user feedback
- **Progress Callbacks**: Real-time download progress
- **Model Validation**: Prevents invalid operations

## 📊 Tool Compatibility Database

The view uses an extensive compatibility database covering 20+ model families:

### Excellent Tool Support:
- Llama 3.1/3.2/3.3 series
- Qwen 2.5 (7B+) and Qwen 3
- Command-R Plus
- Qwen2.5-Coder series

### Good Tool Support:
- Mistral series
- Qwen 2.5 (smaller models)
- DeepSeek-R1
- Granite 3.2

### Limited/No Support:
- Gemma series
- Phi series
- Very small models (<1B parameters)

This comprehensive model management interface makes it easy to discover, download, and manage models with full awareness of their tool-calling capabilities.