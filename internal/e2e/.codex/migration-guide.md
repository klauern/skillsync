# Garden v8 to v9 Migration Guide

This guide documents all breaking changes and migration patterns for upgrading from Garden v8 to v9.

## Pre-Migration Checklist

1. Update all v8 packages to **v8.75.0** minimum
2. Upgrade `@zendeskgarden/react-theming` to v9 **first** (central dependency)
3. Update `styled-components` to `^5.3.1`
4. Ensure build pipeline supports `.mjs` files (react-merge-refs v2)

## Core Breaking Changes

### Popper → Floating UI Migration

All `popperModifiers` props removed. Floating UI handles positioning automatically.

**Affected packages:**
- `@zendeskgarden/react-datepickers` (removed `eventsEnabled`)
- `@zendeskgarden/react-dropdowns`
- `@zendeskgarden/react-tooltips`

### Theming System Overhaul

**Removed theme values:**
- `colors.background` → use `'background.default'` with `getColor()`
- `colors.foreground` → use `'foreground.default'` with `getColor()`
- `message` and `connect` product palette values removed

**Changed utility signatures:**
```typescript
// v8
getColor('blue', 600, theme);

// v9
getColor({ theme, variable: 'background.default' });
getColor({ theme, hue: 'blue', shade: 600 });
```

**Removed utilities:**
- `getDocument()` - use standard DOM APIs
- `isRtl()` - check `theme.rtl` directly
- `withTheme()` - use hooks or context
- `focusVisibleRef` prop from ThemeProvider (polyfill no longer needed)

**Temporary fallback:** `getColorV8()` available for legacy color schemes

## Component-Specific Changes

### Buttons (`@zendeskgarden/react-buttons`)

**Removed:**
- `ButtonGroup` component (use individual Buttons)
- `IIconProps` type (use `IButtonStartIconProps` or `IButtonEndIconProps`)

**Changed:**
- `Anchor` and `<Button isLink>` now underlined by default (accessible)
- Disable with `isUnderlined={false}` if needed

**Migration:**
```tsx
// v8
<ButtonGroup>
  <Button>First</Button>
  <Button>Second</Button>
</ButtonGroup>

// v9
<div style={{ display: 'flex', gap: '8px' }}>
  <Button>First</Button>
  <Button>Second</Button>
</div>
```

### Chrome (`@zendeskgarden/react-chrome`)

**Removed:**
- `Sidebar` component (no direct replacement)
- `Subnav` component (no direct replacement)
- `PRODUCT` type export (use `IHeaderItemProps['product']`)
- `hasFooter` prop from `Body`

**Changed:**
- Icon components use `SVGAttributes<SVGElement>` (not `HTMLAttributes`)
- Added `Nav.List` as semantic wrapper

**Deprecated (still works, remove before v10):**
```tsx
// Deprecated
<NavItem />
<HeaderItem />
<FooterItem />

// New syntax
<Nav.Item />
<Header.Item />
<Footer.Item />
```

### Color Pickers (`@zendeskgarden/react-colorpickers`)

**Renamed:**
- `Colorpicker` → `ColorPicker`
- `ColorpickerDialog` → `ColorPickerDialog`

**New requirements:**
```tsx
// v9 - name prop now required
<ColorSwatch name="color-selector" colors={colors} />
<ColorSwatchDialog name="color-dialog" colors={colors} />
```

**Removed:** Row/column index props replaced with controlled selection

### Date Pickers (`@zendeskgarden/react-datepickers`)

**Renamed:**
- `Datepicker` → `DatePicker`
- `DatepickerRange` → `DatePickerRange`

**Removed:**
- `GardenPlacement` type (use `IDatePickerProps['placement']`)
- `eventsEnabled` prop

### Dropdowns (`@zendeskgarden/react-dropdowns`)

**Package restructure:**
- v8 `@zendeskgarden/react-dropdowns` → v9 `@zendeskgarden/react-dropdowns.legacy`
- v8 `@zendeskgarden/react-dropdowns.next` → v9 `@zendeskgarden/react-dropdowns`

**Changes:**
```tsx
// Combobox.Option - no longer accepts object values
<Combobox.Option value="string-only" /> {/* v9 */}

// Combobox.OptGroup - label renamed to legend
<Combobox.OptGroup legend="Group Label"> {/* v9 */}

// Menu - new restoreFocus prop (default: true)
<Menu restoreFocus={false}> {/* v9 */}
```

### Drag and Drop

**Package replacement:**
- v8: `@zendeskgarden/react-drag-drop`
- v9: `@zendeskgarden/react-draggable`

### Forms (`@zendeskgarden/react-forms`)

**Removed:**
- `MultiThumbRange` component
- `IFieldProps` type export
- `IIconProps` type export

**Changed:**
- Icon props require `ReactElement` (not `any`)

**Deprecated subcomponents:**
```tsx
// Deprecated
<Hint />
<Label />
<Message />

// New syntax
<Field.Hint />
<Field.Label />
<Field.Message />
```

### Grid (`@zendeskgarden/react-grid`)

**Removed type exports:**
- `ALIGN_ITEMS`, `ALIGN_SELF`, `DIRECTION`, `JUSTIFY_CONTENT`, `TEXT_ALIGN`
- `GRID_NUMBER`, `BREAKPOINT`, `SPACE`, `WRAP`
- Constants no longer prefixed with `ARRAY_`

**Deprecated subcomponents:**
```tsx
// Deprecated
<Row />
<Col />

// New syntax
<Grid.Row />
<Grid.Col />
```

### Modals (`@zendeskgarden/react-modals`)

**Renamed:**
- `DrawerModal` → `Drawer`
- `TooltipModal` → `TooltipDialog`

**Removed:**
- Internal `useFocusVisible` hook (use theming utils instead)
- `GARDEN_PLACEMENT` type (use `ITooltipDialogProps['placement']`)

**Migration for focus styles:**
```tsx
// v8
import { useFocusVisible } from '@zendeskgarden/react-modals';

// v9
import { focusStyles, getFocusBoxShadow } from '@zendeskgarden/react-theming';
```

**Deprecated subcomponents:**
```tsx
// Deprecated
<Body />
<Close />
<Footer />
<FooterItem />
<Header />

// New syntax
<Modal.Body />
<Modal.Close />
<Modal.Footer />
<Modal.FooterItem />
<Modal.Header />
```

### Notifications (`@zendeskgarden/react-notification`)

**Removed type exports:**
- `ToastPlacement` (use `IToastOptions['placement']`)
- `ToastContent` (use `IToast['content']`)

**Deprecated subcomponents:**
```tsx
// Deprecated
<Close />
<Paragraph />
<Title />

// New syntax
<Alert.Close /> <Notification.Close /> <Well.Close />
<Alert.Paragraph /> <Notification.Paragraph /> <Well.Paragraph />
<Alert.Title /> <Notification.Title /> <Well.Title />
```

### Pagination (`@zendeskgarden/react-pagination`)

**Renamed:**
- `Pagination` → `OffsetPagination`
- `PAGE_TYPE` → `PageType`

**Changed:**
```tsx
// v8
<Pagination transformPageProps={(page) => customProps} />

// v9
<OffsetPagination labels={{ page: 'Custom' }} />
```

### Tables (`@zendeskgarden/react-tables`)

**Removed props from `Table.OverflowButton`:**
- `isHovered`
- `isActive`
- `isFocused`

**Deprecated subcomponents:**
```tsx
// Deprecated
<Body />
<Caption />
<Cell />
<Head />
<HeaderCell />
<Row />

// New syntax
<Table.Body />
<Table.Caption />
<Table.Cell />
<Table.Head />
<Table.HeaderCell />
<Table.Row />
```

### Tabs (`@zendeskgarden/react-tabs`)

**Deprecated subcomponents:**
```tsx
// Deprecated
<Tab />
<TabList />
<TabPanel />

// New syntax
<Tabs.Tab />
<Tabs.TabList />
<Tabs.TabPanel />
```

### Tooltips (`@zendeskgarden/react-tooltips`)

**Removed:**
- `eventsEnabled` prop
- `popperModifiers` prop

**Deprecated subcomponents:**
```tsx
// Deprecated
<Paragraph />
<Title />

// New syntax
<Tooltip.Paragraph />
<Tooltip.Title />
```

### Typography (`@zendeskgarden/react-typography`)

**Changed:**
- `CodeBlock` language support reduced from 32 to 13 languages
- Icon component types use `SVGAttributes<SVGElement>`

**Supported languages (v9):**
- bash, css, diff, html, javascript, json, jsx, markdown, python, ruby, tsx, typescript, xml

### Utilities (`@zendeskgarden/react-utilities`)

**Package removed entirely**

**Migration paths:**
- Container hooks → `@zendeskgarden/container-utilities`
- Theming utilities → `@zendeskgarden/react-theming`

## Timeline Component Changes

### Accordions (`@zendeskgarden/react-accordions`)

**Removed:**
- `IItem` type export (use `ITimelineItemProps`)

**Changed:**
- Icon props now require `ReactElement` (not `ReactNode`)

## Migration Strategy

### Phase 1: Preparation
1. Audit your codebase for Garden component usage
2. Update all v8 packages to v8.75.0+
3. Review breaking changes relevant to your usage

### Phase 2: Theming
1. Upgrade `@zendeskgarden/react-theming` to v9
2. Update `getColor()` calls to new signature
3. Replace removed theme values
4. Test theme-dependent components

### Phase 3: Component Updates
1. Upgrade components by package
2. Address breaking changes per package
3. Update deprecated subcomponent imports (can defer)
4. Test each package upgrade

### Phase 4: Validation
1. Run type checking (TypeScript)
2. Test keyboard navigation and accessibility
3. Visual regression testing
4. Browser compatibility testing

## Common Patterns

### Detecting Version
```typescript
// Check package.json
const gardenVersion = require('@zendeskgarden/react-buttons/package.json').version;
const isV9 = gardenVersion.startsWith('9.');
```

### Conditional Imports
```typescript
// For gradual migration
const Button = isV9
  ? require('@zendeskgarden/react-buttons').Button
  : require('@zendeskgarden/react-buttons/dist/v8').Button;
```

### Theme Migration Helper
```typescript
import { getColor, getColorV8 } from '@zendeskgarden/react-theming';

// Wrapper for gradual migration
const getColorSafe = (theme, hue, shade) => {
  try {
    return getColor({ theme, hue, shade });
  } catch {
    return getColorV8(hue, shade, theme);
  }
};
```

## Resources

- [Official v9 Migration Guide](https://github.com/zendeskgarden/react-components/blob/main/docs/migration.md)
- [Garden Documentation](https://garden.zendesk.com)
- [GitHub Issues](https://github.com/zendeskgarden/react-components/issues)