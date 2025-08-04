# Context Tree UI Documentation

## Overview
The context tree visualization component provides a visual representation of the conversation's branching structure, allowing users to navigate between different conversation contexts and understand the relationships between messages.

## Features

### Visual Tree Display
- Hierarchical tree structure showing all conversation contexts
- Branch points are labeled with the originating message content
- Active context is highlighted with a green indicator
- Message count displayed for each context
- Expandable/collapsible nodes for managing complex trees

### Display Positions
The context tree can be displayed in four positions:
- **Right** (default): Occupies 1/3 of the right side of the screen
- **Left**: Occupies 1/3 of the left side of the screen  
- **Bottom**: Occupies 1/3 of the bottom of the screen
- **Float**: Centered floating overlay

### Keyboard Navigation

#### Toggling Visibility
- `Ctrl+T` - Toggle context tree view on/off
- `Alt+T` - Alternative toggle command
- `Esc` - Hide context tree when it's focused

#### Navigation Within Tree
- `↑/↓` - Navigate up/down through contexts
- `←` - Collapse node or navigate to parent
- `→` - Expand node or navigate to first child
- `Space` - Toggle node expansion
- `Enter` - Switch to the selected context

#### Context Operations
- `Ctrl+B` - Branch from the currently focused message (requires node selection mode)

### Integration with Message Nodes
The context tree integrates with the message node system:
1. Focus a message node using node selection mode (`Ctrl+N`)
2. Press `Ctrl+B` to create a new context branch from that message
3. The new context becomes active and appears in the tree

## Implementation Details

### Components
- `ContextTreeView` - Main visualization component
- `TreeNode` - Internal representation of visual tree nodes
- `BranchContextEvent` - Event fired when branching is requested
- `ContextSwitchEvent` - Event fired when switching contexts

### Rendering
The tree view uses tcell for terminal rendering:
- Unicode box-drawing characters for tree structure
- Color coding for active context and selection
- Scroll indicators when content exceeds viewport

### State Management
- Tree expansion state is maintained per session
- Selected node tracks keyboard navigation
- Scroll position adjusts automatically to keep selected node visible

## Future Enhancements
- Context merging capabilities
- Drag-and-drop message reorganization
- Context search and filtering
- Context metadata (tags, timestamps)
- Export context tree as graph visualization