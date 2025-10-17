# Odin API Gateway - Template Architecture

## Overview

This document describes the refactored template architecture for the Odin API Gateway admin panel. The new architecture promotes code reusability, maintainability, and consistency across all admin pages.

## Directory Structure

```
pkg/admin/
├── static/
│   ├── css/
│   │   ├── base.css           # Variables, resets, typography (189 lines)
│   │   ├── components.css     # UI components (482 lines)
│   │   ├── layout.css         # Grid, flexbox, navigation (335 lines)
│   │   └── utilities.css      # Helper classes (363 lines)
│   └── js/
│       ├── api.js             # API communication (202 lines)
│       ├── ui.js              # UI helpers & utilities (349 lines)
│       └── plugins.js         # Plugin-specific logic (422 lines)
├── templates/
│   ├── base.html              # Master template with blocks
│   ├── partials/
│   │   ├── header.html        # Navigation header component
│   │   ├── footer.html        # Footer component
│   │   ├── form-fields.html   # Form input components
│   │   ├── alerts.html        # Alert components
│   │   └── loading.html       # Loading spinner components
│   └── [page templates]
│       ├── dashboard.html
│       ├── plugins.html
│       ├── plugin_new.html
│       ├── plugin_detail.html
│       ├── login.html
│       └── ... (more pages)
```

## CSS Architecture

### 1. base.css (189 lines)
**Purpose**: Foundation styles, CSS variables, and global resets

**Key Features**:
- **CSS Variables**: Colors, spacing, typography, borders, shadows, transitions
- **Typography**: Headings, paragraphs, links, code blocks
- **Resets**: Box-sizing, margins, padding
- **Scrollbar**: Custom scrollbar styling

**Usage Example**:
```css
/* Using CSS variables */
.my-component {
  color: var(--primary-color);
  padding: var(--spacing-md);
  border-radius: var(--border-radius);
  box-shadow: var(--shadow-sm);
}
```

### 2. components.css (482 lines)
**Purpose**: Reusable UI component styles

**Components Included**:
- **Buttons**: Button groups, sizes
- **Forms**: Inputs, selects, checkboxes, radios, textareas
- **Cards**: Card header, body, footer
- **Badges**: Color variants
- **Alerts**: Success, danger, warning, info
- **Tables**: Striped, hover, responsive
- **Modals**: Dialog, header, body, footer
- **Spinners**: Loading indicators
- **Pagination**: Page navigation

**Usage Example**:
```html
<div class="card">
  <div class="card-header">
    <h5 class="card-title">Title</h5>
  </div>
  <div class="card-body">
    Content here
  </div>
  <div class="card-footer">
    <button class="btn btn-primary">Action</button>
  </div>
</div>
```

### 3. layout.css (335 lines)
**Purpose**: Layout system and structural styles

**Features**:
- **Grid System**: 12-column responsive grid
- **Flexbox Utilities**: Display, direction, alignment, justify
- **Container**: Fixed-width and fluid containers
- **Navigation**: Nav pills, nav links
- **Header/Footer**: Standard page structure
- **Responsive**: Mobile-first with breakpoints

**Usage Example**:
```html
<div class="container">
  <div class="row">
    <div class="col-md-6">Left column</div>
    <div class="col-md-6">Right column</div>
  </div>
</div>

<div class="d-flex justify-content-between align-items-center">
  <h1>Title</h1>
  <button>Action</button>
</div>
```

### 4. utilities.css (363 lines)
**Purpose**: Helper classes for quick styling

**Utilities Included**:
- **Spacing**: Margin (m-*), padding (p-*), with directions (mt, mb, ml, mr, mx, my)
- **Text**: Alignment, colors, decoration, transform, weight, size
- **Background**: Color utilities
- **Display**: Block, inline, flex, none
- **Width/Height**: Percentage classes
- **Shadows**: Box shadow variants
- **Visibility**: Show/hide helpers

**Usage Example**:
```html
<div class="mb-3 p-4 bg-light text-center shadow-sm">
  <h2 class="text-primary fw-bold">Heading</h2>
  <p class="text-muted">Description</p>
</div>
```

## JavaScript Modules

### 1. api.js (202 lines)
**Purpose**: Centralized API communication layer

**Exports**:
```javascript
OdinAPI.plugins.list()              // GET /admin/api/plugins
OdinAPI.plugins.get(name)           // GET /admin/api/plugins/:name
OdinAPI.plugins.create(data)        // POST /admin/api/plugins
OdinAPI.plugins.update(name, data)  // PUT /admin/api/plugins/:name
OdinAPI.plugins.delete(name)        // DELETE /admin/api/plugins/:name
OdinAPI.plugins.enable(name)        // POST /admin/api/plugins/:name/enable
OdinAPI.plugins.disable(name)       // POST /admin/api/plugins/:name/disable
OdinAPI.plugins.upload(formData)    // POST /admin/api/plugins/upload
OdinAPI.plugins.build(data)         // POST /admin/api/plugins/build
OdinAPI.plugins.test(name, data)    // POST /admin/api/plugins/test/:name

OdinAPI.services.list()             // Service CRUD operations
OdinAPI.postman.listCollections()   // Postman integration
OdinAPI.monitoring.health()         // Health checks
```

**Usage Example**:
```javascript
// Using the API module
async function deletePlugin(name) {
  try {
    await OdinAPI.plugins.delete(name);
    OdinUI.showToast('Plugin deleted successfully', 'success');
    location.reload();
  } catch (error) {
    OdinUI.showToast(`Error: ${error.message}`, 'danger');
  }
}
```

### 2. ui.js (349 lines)
**Purpose**: UI helpers and utilities

**Functions**:
```javascript
// Notifications
OdinUI.showToast(message, type, duration)    // Show toast notification
OdinUI.showError(element, message, details)  // Display error
OdinUI.showLoading(element, message)         // Show loading spinner
OdinUI.hideLoading(element, content)         // Hide loading

// Modals
OdinUI.confirm(message, title)               // Confirmation dialog
OdinUI.createModal(title, body, buttons)     // Create custom modal

// Utilities
OdinUI.formatDate(date, format)              // Format dates
OdinUI.formatBytes(bytes)                    // Human-readable bytes
OdinUI.copyToClipboard(text)                 // Copy to clipboard
OdinUI.debounce(func, wait)                  // Debounce function
OdinUI.throttle(func, limit)                 // Throttle function
OdinUI.escapeHtml(text)                      // XSS protection
```

**Usage Example**:
```javascript
// Show success notification
OdinUI.showToast('Operation completed!', 'success');

// Confirm before delete
const confirmed = await OdinUI.confirm(
  'Are you sure you want to delete this item?',
  'Confirm Delete'
);
if (confirmed) {
  // Perform delete
}

// Format date
const formatted = OdinUI.formatDate(new Date(), 'datetime');
// Output: "2025-10-16 14:30:45"
```

### 3. plugins.js (422 lines)
**Purpose**: Plugin-specific functionality

**Functions**:
```javascript
OdinPlugins.initPluginForm()        // Initialize plugin form handlers
OdinPlugins.initPluginDetail()      // Initialize plugin detail page
OdinPlugins.togglePlugin(name, enabled)  // Enable/disable plugin
OdinPlugins.deletePlugin(name)      // Delete with confirmation
OdinPlugins.testPlugin(name)        // Test plugin
OdinPlugins.parseConfig(json)       // Parse JSON config
```

**Features**:
- Source type selection (file, path, template)
- Template selection with descriptions
- Plugin type handling (hooks vs middleware)
- Form validation and submission
- Error handling

## Template System

### Base Template (base.html)

All pages should extend the base template for consistency:

```html
{{define "page-name.html"}}
{{template "base" .}}
{{end}}

{{define "title"}}Page Title{{end}}

{{define "head-extra"}}
<!-- Optional: Additional CSS/JS for this page -->
<style>
  /* Page-specific styles */
</style>
{{end}}

{{define "container-class"}}container{{end}} <!-- or "container-fluid" -->

{{define "content"}}
<!-- Your page content here -->
<h1>Welcome</h1>
<p>Page content...</p>
{{end}}

{{define "scripts"}}
<!-- Optional: Page-specific JavaScript -->
<script>
  // Page-specific code
</script>
{{end}}
```

### Available Blocks

1. **`title`**: Page title (shows in browser tab)
2. **`head-extra`**: Additional CSS/JS in `<head>`
3. **`container-class`**: Container class (`container` or `container-fluid`)
4. **`body-content`**: Complete body override (rare)
5. **`content`**: Main page content
6. **`scripts`**: Page-specific JavaScript

### Using Partials

#### Header
```html
{{template "header" .}}
```

Renders the navigation header with all admin links. Active link is automatically highlighted by JavaScript.

#### Footer
```html
{{template "footer" .}}
```

Renders the standard footer with copyright information.

#### Form Fields

**Text Input**:
```html
{{template "form-field-text" (dict 
  "ID" "username" 
  "Name" "username" 
  "Label" "Username" 
  "Placeholder" "Enter username" 
  "Required" true
  "Help" "Your unique username"
)}}
```

**Textarea**:
```html
{{template "form-field-textarea" (dict 
  "ID" "description" 
  "Name" "description" 
  "Label" "Description" 
  "Rows" 5
  "Placeholder" "Enter description"
)}}
```

**Select**:
```html
{{template "form-field-select" (dict 
  "ID" "type" 
  "Name" "type" 
  "Label" "Type" 
  "Options" (list
    (dict "Value" "hooks" "Label" "Hook-based" "Selected" true)
    (dict "Value" "middleware" "Label" "Middleware")
  )
)}}
```

**Checkbox**:
```html
{{template "form-field-checkbox" (dict 
  "ID" "enabled" 
  "Name" "enabled" 
  "Label" "Enable plugin" 
  "Checked" true
)}}
```

#### Alerts

```html
<!-- Success alert -->
{{template "alert-success" (dict "Message" "Operation successful!" "Dismissible" true)}}

<!-- Error alert -->
{{template "alert-danger" (dict 
  "Title" "Error:" 
  "Message" "Something went wrong" 
  "Dismissible" true
)}}

<!-- Info alert -->
{{template "alert-info" (dict "Message" "Here's some helpful information")}}

<!-- Warning alert -->
{{template "alert-warning" (dict "Message" "Please be careful!")}}
```

#### Loading Spinners

```html
<!-- Full loading section -->
{{template "loading-spinner" (dict "Message" "Loading data...")}}

<!-- Small inline spinner -->
<button class="btn btn-primary" disabled>
  {{template "loading-spinner-sm"}} Loading...
</button>
```

## Migration Guide

### Converting Existing Templates

**Before** (Old Style):
```html
{{define "page.html"}}
<!DOCTYPE html>
<html>
  <head>
    <title>Page</title>
    <link href="https://cdn.../bootstrap.min.css" rel="stylesheet" />
    <style>
      /* Inline styles */
    </style>
  </head>
  <body>
    <div class="container">
      <header>
        <!-- Duplicated header -->
      </header>
      
      <main>
        <!-- Content -->
      </main>
      
      <footer>
        <!-- Duplicated footer -->
      </footer>
    </div>
    
    <script>
      // Inline JavaScript
    </script>
  </body>
</html>
{{end}}
```

**After** (New Style):
```html
{{define "page.html"}}
{{template "base" .}}
{{end}}

{{define "title"}}Page Title{{end}}

{{define "content"}}
<!-- Content only -->
<h1>Page Content</h1>
{{end}}

{{define "scripts"}}
<script src="/static/js/page-specific.js"></script>
{{end}}
```

### Step-by-Step Migration

1. **Create new template definition**:
   ```html
   {{define "page.html"}}
   {{template "base" .}}
   {{end}}
   ```

2. **Add title block**:
   ```html
   {{define "title"}}Your Page Title{{end}}
   ```

3. **Extract content**:
   - Copy only the `<main>` content
   - Remove header/footer HTML
   - Paste into `{{define "content"}}` block

4. **Move inline styles to CSS files**:
   - Extract CSS from `<style>` tags
   - Add to appropriate CSS file (base, components, layout, or utilities)
   - Or add to `head-extra` block if truly page-specific

5. **Move inline JavaScript**:
   - Extract from `<script>` tags
   - Create `/static/js/page-name.js` if substantial
   - Or add to `scripts` block if minimal

6. **Replace manual HTML with partials**:
   - Forms: Use `form-field-*` templates
   - Alerts: Use `alert-*` templates
   - Loading: Use `loading-spinner` templates

7. **Test the page**:
   - Verify rendering
   - Test JavaScript functionality
   - Check responsive design

## Best Practices

### CSS

1. **Use CSS variables** for consistency:
   ```css
   .my-component {
     color: var(--primary-color);
     padding: var(--spacing-md);
   }
   ```

2. **Use utility classes** for quick styling:
   ```html
   <div class="mb-3 p-4 text-center">...</div>
   ```

3. **Keep page-specific styles minimal**:
   - Reuse components CSS when possible
   - Only add to `head-extra` if truly unique

### JavaScript

1. **Use provided APIs**:
   ```javascript
   // Good
   await OdinAPI.plugins.delete(name);
   
   // Avoid
   fetch('/admin/api/plugins/' + name, {method: 'DELETE'});
   ```

2. **Use UI helpers**:
   ```javascript
   // Good
   OdinUI.showToast('Success!', 'success');
   
   // Avoid
   alert('Success!');
   ```

3. **Handle errors properly**:
   ```javascript
   try {
     await OdinAPI.plugins.create(data);
     OdinUI.showToast('Created!', 'success');
   } catch (error) {
     OdinUI.showError('#form', error.message);
   }
   ```

### Templates

1. **Always extend base template**:
   ```html
   {{define "page.html"}}
   {{template "base" .}}
   {{end}}
   ```

2. **Use partials for reusable components**:
   ```html
   {{template "form-field-text" (dict ...)}}
   ```

3. **Keep templates focused**:
   - One responsibility per template
   - Extract repeated sections into partials

## Performance Considerations

1. **Static file caching**:
   - CSS/JS served from `/static/` with proper cache headers
   - Browser caching reduces load time

2. **Minification** (future):
   - Consider minifying CSS/JS for production
   - Combine files to reduce HTTP requests

3. **Lazy loading** (future):
   - Load heavy JavaScript only when needed
   - Use dynamic imports for large modules

## Adding New Pages

### Quick Start Template

Create `templates/new-page.html`:

```html
{{define "new-page.html"}}
{{template "base" .}}
{{end}}

{{define "title"}}New Page - Odin API Gateway{{end}}

{{define "content"}}
<div class="d-flex justify-content-between align-items-center mb-4">
  <h1>
    <i class="bi bi-icon"></i> Page Title
  </h1>
  <a href="/admin/back" class="btn btn-secondary">
    <i class="bi bi-arrow-left"></i> Back
  </a>
</div>

<div class="card">
  <div class="card-body">
    <!-- Your content here -->
  </div>
</div>
{{end}}

{{define "scripts"}}
<script>
  // Page-specific JavaScript
</script>
{{end}}
```

### Adding a Route

In `pkg/admin/routes.go`:

```go
protected.GET("/new-page", h.handleNewPage)
```

In `pkg/admin/admin.go`:

```go
func (h *AdminHandler) handleNewPage(c echo.Context) error {
  data := map[string]interface{}{
    "Title": "New Page",
    // ... other data
  }
  return h.renderTemplate(c, "new-page.html", data)
}
```

## Testing

### Browser Testing Checklist

- [ ] Page loads without errors
- [ ] CSS applies correctly
- [ ] JavaScript functions work
- [ ] Forms submit properly
- [ ] HTMX interactions work
- [ ] Responsive design works on mobile
- [ ] No console errors
- [ ] Links navigate correctly
- [ ] Modals and alerts display

### Debugging Tips

1. **Check browser console** for JavaScript errors
2. **Verify static files load** in Network tab
3. **Check template rendering** by viewing source
4. **Test with browser DevTools** responsive mode
5. **Clear cache** if styles don't update

## Future Enhancements

1. **Theme Support**:
   - CSS variable-based theming
   - Light/dark mode toggle
   - Custom color schemes

2. **Internationalization**:
   - Multi-language support
   - Localized date/time formatting
   - RTL layout support

3. **Accessibility**:
   - ARIA labels
   - Keyboard navigation
   - Screen reader support

4. **Advanced Components**:
   - Data tables with sorting/filtering
   - Rich text editor
   - File upload with progress
   - Charts and graphs

## Support

For questions or issues:
- Review examples in existing templates
- Check browser console for errors
- Refer to Bootstrap 5.3 documentation
- Check HTMX documentation for AJAX behavior

## Version History

- **v1.0.0** (2025-10-16): Initial refactored architecture
  - Created CSS module system (4 files, 1,369 lines)
  - Created JavaScript modules (3 files, 973 lines)
  - Created template partials (5 components)
  - Migrated 3 core templates (login, dashboard, plugins)
  - Established base template inheritance

---

**Maintained by**: Odin API Gateway Team  
**Last Updated**: October 16, 2025
