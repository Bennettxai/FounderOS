# OSA Workflow Viewer Component

## Overview
A comprehensive Svelte 5 component for viewing OSA workflow files and details, built for the BusinessOS frontend.

## Features

### 1. Workflow Metadata Display
- Workflow name and display name
- Status badge with color-coded indicators (completed, processing, failed)
- Description and creation timestamps
- File count summary

### 2. File Organization
- **Categorized Tabs**: Files organized by type
  - All Files
  - Code (TypeScript, JavaScript, Go, etc.)
  - Schema (Database schemas, SQL)
  - Config (YAML, JSON, environment files)
  - Documentation (Markdown, README)
  - Deployment (Docker, CI/CD configs)
- Dynamic tab visibility based on available file types
- File count badges on each tab

### 3. File Preview
- **Syntax Highlighting**: Auto-detected language support for 20+ languages
- **Markdown Rendering**: Rich markdown preview with styled elements
  - Headers, lists, tables
  - Code blocks with syntax highlighting
  - Links, blockquotes, images
- **File Metadata**: Size, type, last updated timestamp

### 4. User Actions
- **Copy to Clipboard**: One-click code copying with success indicator
- **Download**: Download individual files
- **Install as Module**: Install entire workflow as a reusable module
- **Navigation**: Back button to return to desktop

### 5. Svelte 5 Features
- **Runes-based State**: Uses `$state`, `$derived`, `$effect` for reactive state
- **Modern Syntax**: Props destructuring with `$props()`
- **Type Safety**: Full TypeScript integration

## File Structure

```
/window/osa/[id]/
├── +page.svelte          # Main component
└── README.md             # This file
```

## API Integration

### Endpoints Used
1. `GET /api/osa/workflows/:id` - Fetch workflow details
2. `GET /api/osa/workflows/:id/files` - Fetch workflow files
3. `GET /api/osa/files/:id/content` - Fetch file content
4. `POST /api/osa/modules/install` - Install workflow as module

### Data Flow
```
Mount → Load Workflow → Load Files → Select First File → Display Content
                                                       ↓
                                          User Selects File → Load Content
```

## Styling

### Color Scheme
- **Background**: Dark mode (#0f172a, #1e293b)
- **Primary**: Blue (#3b82f6, #60a5fa)
- **Success**: Green (#059669, #10b981)
- **Error**: Red (#ef4444, #dc2626)
- **Text**: Slate grays (#e2e8f0, #94a3b8, #64748b)

### Layout
- **Header**: Fixed, contains back button, workflow info, install button
- **Sidebar**: 320px fixed width, scrollable file list
- **Main**: Flex-fill, file preview with header and content area

## State Management

```typescript
// Workflow data
workflow: any | null           // Current workflow metadata
files: any[]                   // All files in workflow
selectedFile: any | null       // Currently selected file

// Content
fileContent: string | null     // Raw file content
renderedMarkdown: string       // Rendered markdown HTML

// UI State
activeTab: string              // Current file category filter
loading: boolean               // Initial load state
loadingContent: boolean        // File content load state
error: string | null           // Error messages

// Actions
copied: boolean                // Copy button feedback
installing: boolean            // Install in progress
installSuccess: boolean        // Install success feedback
installError: string | null    // Install error message
```

## Language Support

Auto-detected syntax highlighting for:
- JavaScript, TypeScript, JSX, TSX
- Python, Go, Rust, Java, C, C++, C#
- Ruby, PHP, Shell/Bash
- HTML, CSS, SCSS
- JSON, YAML, XML
- SQL, Markdown, Dockerfile

## Usage

### Navigation
```typescript
// From another component
goto('/window/osa/[workflow-id]');

// Example
goto('/window/osa/550e8400-e29b-41d4-a716-446655440000');
```

### URL Parameters
- `:id` - Workflow ID (UUID or OSA workflow ID)

## Error Handling

- **Workflow Not Found**: Shows error message with back button
- **File Load Errors**: Displays error in content area
- **Network Errors**: Graceful error messages with retry options
- **Install Errors**: Red banner with error details

## Accessibility

- Keyboard navigation support
- Semantic HTML structure
- ARIA labels on interactive elements
- Focus indicators on all interactive elements
- Color contrast compliance

## Performance

- Lazy loading of file content (only loads when selected)
- Efficient re-rendering with Svelte 5 runes
- Debounced file selection
- Optimized markdown parsing

## Dependencies

```json
{
  "svelte": "^5.x",
  "lucide-svelte": "^0.x",
  "marked": "^x.x",
  "$app/stores": "SvelteKit",
  "$app/navigation": "SvelteKit",
  "$lib/api/base": "BusinessOS API",
  "$lib/api/osa/files": "OSA API"
}
```

## Future Enhancements

- [ ] Syntax highlighting with Shiki
- [ ] File search within workflow
- [ ] Multi-file download as ZIP
- [ ] File comparison/diff view
- [ ] Inline file editing
- [ ] Version history
- [ ] Comments and annotations
- [ ] Share workflow link
- [ ] Export to various formats

## Related Components

- `/lib/components/osa/FilePreview.svelte` - Base file preview component
- `/lib/components/osa/FileTree.svelte` - File tree navigation
- `/lib/components/osa/InstallButton.svelte` - Module installation

## Testing

### Manual Testing Checklist
- [ ] Workflow loads correctly
- [ ] All file tabs display correct counts
- [ ] File selection updates preview
- [ ] Markdown renders properly
- [ ] Code syntax highlighting works
- [ ] Copy button copies content
- [ ] Download button downloads file
- [ ] Install button triggers module installation
- [ ] Error states display correctly
- [ ] Back button navigates correctly

### Test Data
Use the OSA-5 backend endpoints to generate test workflows with various file types.

## Troubleshooting

### Component Not Rendering
- Check that workflow ID is valid UUID or matches OSA workflow ID pattern
- Verify API endpoints are accessible
- Check browser console for errors

### Files Not Loading
- Verify `/api/osa/workflows/:id/files` returns data
- Check network tab for failed requests
- Ensure user has permission to access workflow

### Markdown Not Rendering
- Check that `marked` package is installed
- Verify file type is detected as markdown
- Check for HTML sanitization issues

### Syntax Highlighting Not Working
- Consider adding Shiki for better highlighting
- Verify language detection in `getLanguageFromFile()`
- Check CSS class application

## Contributing

When modifying this component:
1. Maintain Svelte 5 runes syntax
2. Keep TypeScript types up to date
3. Update this README with new features
4. Test all file type categories
5. Verify responsive design
6. Check accessibility compliance

## License
Part of BusinessOS - See root LICENSE file
