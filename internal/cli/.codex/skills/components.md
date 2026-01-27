# Garden Component Index

Quick reference for Garden React components, their packages, and key usage patterns.

## Component Packages

### Buttons (`@zendeskgarden/react-buttons`)

**Components:**
- `Button` - Primary button component
- `IconButton` - Icon-only button (requires `aria-label`)
- `SplitButton` - Button with dropdown menu
- `ToggleButton` - Toggleable button state
- `ToggleIconButton` - Toggleable icon button

**Key Props:**
- `isPrimary`, `isBasic`, `isDanger` - Style variants
- `isPill` - Rounded pill shape
- `size` - 'small' | 'medium' | 'large'
- `isDisabled` - Disabled state

**v8 → v9 Changes:**
- `ButtonGroup` removed - use flex layout with individual Buttons
- `Anchor` and `<Button isLink>` now underlined by default

---

### Forms (`@zendeskgarden/react-forms`)

**Components:**
- `Field` - Form field wrapper (use dot-notation: `Field.Label`, `Field.Hint`, `Field.Message`)
- `Input` - Text input
- `Textarea` - Multi-line text input
- `Select` - Dropdown select
- `Checkbox` - Checkbox input
- `Radio` - Radio button
- `Toggle` - Toggle switch
- `Range` - Slider input
- `FileUpload` - File upload component
- `InputGroup` - Input with icons/buttons

**Key Props:**
- `validation` - 'success' | 'error' | undefined
- `required` - Required field
- `disabled` - Disabled state

**v8 → v9 Changes:**
- Use `Field.Label`, `Field.Hint`, `Field.Message` (dot-notation)
- `MultiThumbRange` removed

**See:** `patterns/forms.md` for comprehensive examples

---

### Modals (`@zendeskgarden/react-modals`)

**Components:**
- `Modal` - Standard modal dialog
- `Drawer` - Side drawer (was `DrawerModal` in v8)
- `TooltipDialog` - Tooltip-style dialog (was `TooltipModal` in v8)

**Subcomponents (dot-notation):**
- `Modal.Header` - Modal header
- `Modal.Body` - Modal content
- `Modal.Footer` - Modal footer
- `Modal.FooterItem` - Footer button wrapper
- `Modal.Close` - Close button

**Key Props:**
- `isOpen` - Control visibility
- `onClose` - Close handler
- Focus trap handled automatically

**See:** `patterns/accessibility.md` for modal accessibility patterns

---

### Dropdowns (`@zendeskgarden/react-dropdowns`)

**Components:**
- `Combobox` - Autocomplete input
- `Menu` - Dropdown menu
- `Select` - Select dropdown (also in forms package)

**Subcomponents:**
- `Combobox.Input` - Input field
- `Combobox.Listbox` - Options container
- `Combobox.Option` - Option item
- `Combobox.OptGroup` - Option group (use `legend` prop, not `label`)
- `Menu.Item` - Menu item
- `Menu.Separator` - Menu divider

**v8 → v9 Changes:**
- Package restructure: v8 `@zendeskgarden/react-dropdowns` → v9 `@zendeskgarden/react-dropdowns.legacy`
- `Combobox.Option` no longer accepts object values (string only)
- `Combobox.OptGroup` uses `legend` instead of `label`

---

### Notifications (`@zendeskgarden/react-notification`)

**Components:**
- `Alert` - Alert notification
- `Notification` - Toast notification
- `Well` - Informational well
- `GlobalAlert` - Global alert banner

**Subcomponents:**
- `Alert.Title`, `Alert.Paragraph`, `Alert.Close`
- `Notification.Title`, `Notification.Paragraph`, `Notification.Close`

**Key Props:**
- `type` - 'success' | 'error' | 'warning' | 'info'

---

### Loaders (`@zendeskgarden/react-loaders`)

**Components:**
- `Spinner` - Loading spinner
- `Dots` - Dot loader
- `Inline` - Inline loader wrapper
- `Progress` - Progress bar
- `Skeleton` - Skeleton loading state

**Key Props:**
- `size` - 'small' | 'medium' | 'large'

---

### Typography (`@zendeskgarden/react-typography`)

**Components:**
- `Paragraph` - Paragraph text
- `Span` - Inline span
- `Code` - Inline code
- `Kbd` - Keyboard key display
- `CodeBlock` - Code block (13 languages in v9, down from 32 in v8)
- `Lists` - List components

---

### Layout (`@zendeskgarden/react-grid`)

**Components:**
- `Grid` - Grid container
- `Grid.Row` - Grid row (was `Row` in v8)
- `Grid.Col` - Grid column (was `Col` in v8)
- `Pane` - Pane container
- `Sheet` - Sheet container
- `Drawer` - Drawer container

**v8 → v9 Changes:**
- Use `Grid.Row` and `Grid.Col` (dot-notation)
- Removed type exports: `ALIGN_ITEMS`, `DIRECTION`, etc.

---

### Navigation (`@zendeskgarden/react-*`)

**Breadcrumbs** (`@zendeskgarden/react-breadcrumbs`):
- `Breadcrumb` - Breadcrumb navigation

**Tabs** (`@zendeskgarden/react-tabs`):
- `Tabs` - Tabs container
- `Tabs.Tab`, `Tabs.TabList`, `Tabs.TabPanel` - Subcomponents

**Pagination** (`@zendeskgarden/react-pagination`):
- `OffsetPagination` - Pagination (was `Pagination` in v8)

**Menu** (`@zendeskgarden/react-dropdowns`):
- See Dropdowns section above

---

### Data Display (`@zendeskgarden/react-*`)

**Tables** (`@zendeskgarden/react-tables`):
- `Table` - Table container
- `Table.Head`, `Table.Body`, `Table.Row`, `Table.Cell` - Subcomponents
- `Table.Caption` - Table caption
- `Table.SortableCell` - Sortable header cell

**Accordion** (`@zendeskgarden/react-accordions`):
- `Accordion` - Accordion container
- `Accordion.Section`, `Accordion.Header`, `Accordion.Panel` - Subcomponents

**Timeline** (`@zendeskgarden/react-accordions`):
- `Timeline` - Timeline container
- `Timeline.Item` - Timeline item

---

### Overlays (`@zendeskgarden/react-tooltips`)

**Components:**
- `Tooltip` - Tooltip overlay
- `Tooltip.Paragraph`, `Tooltip.Title` - Subcomponents

**Key Props:**
- `content` - Tooltip content
- `placement` - Placement position

**v8 → v9 Changes:**
- `popperModifiers` removed (Floating UI handles positioning)
- `eventsEnabled` prop removed

---

### Pickers (`@zendeskgarden/react-*`)

**Color Picker** (`@zendeskgarden/react-colorpickers`):
- `ColorPicker` - Color picker (was `Colorpicker` in v8)
- `ColorPickerDialog` - Color picker dialog
- `ColorSwatch` - Color swatch selector (requires `name` prop in v9)

**Date Picker** (`@zendeskgarden/react-datepickers`):
- `DatePicker` - Date picker (was `Datepicker` in v8)
- `DatePickerRange` - Date range picker

**v8 → v9 Changes:**
- Capitalization: `Colorpicker` → `ColorPicker`, `Datepicker` → `DatePicker`
- `ColorSwatch` requires `name` prop

---

### Other Components

**Avatar** (`@zendeskgarden/react-avatars`):
- `Avatar` - User avatar

**Tags** (`@zendeskgarden/react-tags`):
- `Tag` - Tag component

**Tiles** (`@zendeskgarden/react-tiles`):
- `Tile` - Tile component

**Status Indicator** (`@zendeskgarden/react-status-indicators`):
- `StatusIndicator` - Status indicator

**Stepper** (`@zendeskgarden/react-steppers`):
- `Stepper` - Step indicator

**Anchor** (`@zendeskgarden/react-buttons`):
- `Anchor` - Link component

**Draggable** (`@zendeskgarden/react-draggable`):
- `Draggable` - Draggable element (was `@zendeskgarden/react-drag-drop` in v8)

---

## Theming Package

**`@zendeskgarden/react-theming`** - Required for all Garden components

**Exports:**
- `ThemeProvider` - Required wrapper (must wrap app)
- `ColorSchemeProvider` - Dark mode support
- `DEFAULT_THEME` - Default theme object
- `PALETTE` - Color palette
- `getColor()` - Color utility (v9 signature: `getColor({ theme, hue, shade })` or `getColor({ theme, variable })`)
- `focusStyles()` - Focus styling utility
- `useTheme()` - Theme hook

**See:** `patterns/theming.md` for comprehensive theming guide

---

## Version Detection

**Check installed version:**
```bash
cat package.json | grep "@zendeskgarden/react-" | head -1
```

**v9 Requirements:**
- React 16.8+ (hooks support)
- styled-components ^5.3.1
- All packages must be v9 (no mixing v8/v9)

**v8 → v9 Migration:**
See `migration-guide.md` for complete migration reference

---

## Quick Import Reference

```typescript
// Buttons
import { Button } from '@zendeskgarden/react-buttons';

// Forms
import { Field, Input, Textarea, Select } from '@zendeskgarden/react-forms';

// Modals
import { Modal } from '@zendeskgarden/react-modals';

// Theming (required)
import { ThemeProvider } from '@zendeskgarden/react-theming';

// Dropdowns
import { Combobox, Menu } from '@zendeskgarden/react-dropdowns';

// Notifications
import { Alert, Notification } from '@zendeskgarden/react-notification';

// Loaders
import { Spinner, Inline } from '@zendeskgarden/react-loaders';
```