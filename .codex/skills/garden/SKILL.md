---
description: Work with Zendesk's Garden design system - generate React components, setup theming, migrate between versions, create tests and stories, and apply best practices.
name: garden
---

# Garden Design System Skill

You are a Garden design system expert. Help developers work efficiently with Zendesk's Garden React components by generating code, providing guidance, handling migrations, and ensuring accessibility.

## When to Use This Skill

Trigger this skill when the user mentions:
- "create/generate a Garden component"
- "add Garden [component] to [file]"
- "setup Garden theme/theming"
- "migrate to Garden v9" or "upgrade Garden"
- "install Garden [component]"
- "Garden best practices"
- "fix Garden accessibility"
- "Garden [component] example"
- Any reference to `@zendeskgarden/react-*` packages

## Quick Reference

**Component Index:** See `references/components.md` for complete component list, packages, and imports.

**Key Packages:**
- `@zendeskgarden/react-theming` - Required for all components
- `@zendeskgarden/react-buttons` - Buttons
- `@zendeskgarden/react-forms` - Form inputs
- `@zendeskgarden/react-modals` - Modals and dialogs
- `@zendeskgarden/react-dropdowns` - Combobox, Menu
- `@zendeskgarden/react-notification` - Alerts, notifications

**Version Detection:**
```bash
cat package.json | grep "@zendeskgarden/react-" | head -1
```

**v9 Requirements:**
- React 16.8+ (hooks support)
- styled-components ^5.3.1
- All packages must be v9 (no mixing v8/v9)

## Core Capabilities

1. **Component Generation** - TypeScript, accessibility, best practices, tests, stories
2. **Version Detection** - Auto-detect v8/v9 and generate appropriate code
3. **Package Management** - Check/install dependencies (ask for confirmation)
4. **Migration Assistance** - v8 → v9 migration with breaking change detection
5. **Theming Setup** - ThemeProvider, dark mode, custom tokens
6. **Testing & Stories** - Jest/RTL tests with accessibility checks, Storybook stories

## Workflows

### Workflow 1: Generate New Component

1. **Detect Garden version:**
   ```bash
   cat package.json | grep "@zendeskgarden"
   ```

2. **Determine requirements:**
   - Which Garden component(s)? (see `references/components.md`)
   - TypeScript or JavaScript?
   - Need tests? Storybook?
   - Styling approach?

3. **Check/install packages:**
   - Check if required packages installed
   - Ask user for confirmation before installing
   - Use `bun add` per user preferences
   - Required: `@zendeskgarden/react-theming`, `styled-components`, component package(s)

4. **Generate component:**
   - Use `templates/component.tsx.tmpl` as base
   - Import required components (see `references/components.md`)
   - TypeScript interface for props
   - Implement with Garden components using dot-notation (v9)
   - Add accessibility attributes (see `patterns/accessibility.md`)
   - Export component

5. **Generate test (if requested):**
   - Use `templates/component.test.tsx.tmpl` as base
   - Render tests with ThemeProvider wrapper
   - Accessibility tests with jest-axe
   - Interaction tests with userEvent

6. **Generate story (if requested):**
   - Use `templates/component.stories.tsx.tmpl` as base
   - Default story with controls
   - Multiple variants

7. **Provide usage example and brief insights**

### Workflow 2: Setup Theming

1. **Check existing setup:**
   ```bash
   rg "ThemeProvider" --type tsx --type ts
   ```

2. **Install theming package:**
   ```bash
   bun add @zendeskgarden/react-theming styled-components
   ```

3. **Ask about customizations:**
   - Custom colors/branding?
   - Dark mode support?
   - RTL support?
   - Design tokens?

4. **Generate theme config:**
   - Use `templates/theme.ts.tmpl` as base
   - Custom theme object extending DEFAULT_THEME
   - Design token integration (see `patterns/design-tokens.md`)

5. **Generate provider:**
   - Use `templates/provider.tsx.tmpl` as base
   - Wrap app with ThemeProvider
   - Add ColorSchemeProvider if dark mode (see `patterns/theming.md`)

6. **Provide usage examples**

### Workflow 3: Migrate v8 → v9

1. **Identify migration scope:**
   ```bash
   rg "@zendeskgarden/react-" --type tsx --type ts
   ```

2. **Detect version:**
   ```bash
   cat package.json | grep "@zendeskgarden/react-" | head -1
   ```

3. **Read component file(s)**

4. **Identify v8 patterns** (see `migration-guide.md`):
   - `ButtonGroup` → individual Buttons with flex layout
   - `getColor('blue', 600, theme)` → `getColor({ theme, hue: 'blue', shade: 600 })`
   - Individual imports → dot-notation subcomponents
   - `Colorpicker` → `ColorPicker` (capitalization)
   - `popperModifiers` → removed (Floating UI)

5. **Generate migration plan:**
   - List breaking changes found
   - Show before/after examples
   - Explain reasoning

6. **Generate migrated code:**
   - Update imports
   - Replace deprecated patterns
   - Update prop usage
   - Fix type references

7. **Update package.json:**
   - Upgrade to v9 packages
   - Update styled-components to ^5.3.1

8. **Provide testing checklist**

### Workflow 4: Fix Accessibility Issues

1. **Read component code**

2. **Identify issues** (see `patterns/accessibility.md`):
   - Missing ARIA labels (especially icon buttons)
   - Incorrect ARIA attributes
   - Missing keyboard navigation
   - Poor focus management
   - Insufficient color contrast
   - Missing labels on form inputs

3. **Apply Garden patterns:**
   - Proper ARIA attributes
   - Keyboard event handlers
   - Focus trap for modals
   - Screen reader text
   - Semantic HTML

4. **Generate fixed code**

5. **Suggest testing:**
   - jest-axe for automated checks
   - Manual keyboard navigation
   - Screen reader testing

## Tool Usage

### Read Tool
- Check existing Garden usage
- Read component files for migration
- Inspect package.json for versions
- Read pattern files when needed

### Write Tool
- Create new component files (use templates/)
- Create test files (use templates/)
- Create story files (use templates/)
- Create theme configuration (use templates/)

### Edit Tool
- Update existing components
- Fix accessibility issues
- Migrate deprecated patterns

### Bash Tool
- Check package installations
- Install missing packages (with user confirmation)
- Run tests
- Check Garden versions

### Glob Tool
- Find all files using Garden components
- Locate theme configuration
- Find test files

### Grep Tool
- Search for specific Garden patterns
- Find deprecated usage
- Locate ThemeProvider usage

### AskUserQuestion Tool
Use when:
- Multiple valid approaches exist
- Need clarification on requirements
- Confirming package installations
- Choosing between patterns

### WebFetch Tool
Use for latest documentation:
- When component not in embedded knowledge
- User requests latest docs
- Version-specific information needed

### Context7 Integration
Use `/zendeskgarden/react-components` library ID:
```typescript
mcp__context7__get-library-docs({
  context7CompatibleLibraryID: '/zendeskgarden/react-components',
  topic: 'button component usage examples'
});
```

## File References

**Component Reference:**
- `references/components.md` - Complete component index with packages, props, v8/v9 differences

**Pattern References:**
- `patterns/forms.md` - Form patterns, validation, Field component usage
- `patterns/theming.md` - Theming setup, dark mode, custom tokens, RTL support
- `patterns/accessibility.md` - Accessibility best practices, ARIA, keyboard navigation
- `patterns/design-tokens.md` - Design token integration, Style Dictionary, CSS variables

**Templates:**
- `templates/component.tsx.tmpl` - Component template
- `templates/component.test.tsx.tmpl` - Test template
- `templates/component.stories.tsx.tmpl` - Storybook template
- `templates/theme.ts.tmpl` - Theme configuration template
- `templates/provider.tsx.tmpl` - Provider setup template

**Migration:**
- `migration-guide.md` - Complete v8 → v9 migration reference

## Best Practices Checklist

### Accessibility
- ✅ ARIA labels for icon buttons
- ✅ Semantic HTML elements
- ✅ Keyboard navigation
- ✅ Sufficient color contrast (WCAG AA)
- ✅ Focus indicators
- ✅ Screen reader support

### Performance
- ✅ React.memo for expensive components
- ✅ Avoid inline functions in render
- ✅ Lazy load modals and heavy components

### Theming
- ✅ Always wrap app in ThemeProvider
- ✅ Use getColor() for custom colors
- ✅ Support light and dark modes
- ✅ Use focusStyles() for focus states

### Testing
- ✅ Test accessibility with jest-axe
- ✅ Test keyboard navigation
- ✅ Test ARIA attributes
- ✅ Test responsive behavior

### Code Style
- ✅ TypeScript for type safety
- ✅ Document props with JSDoc
- ✅ Use dot-notation for subcomponents (v9)
- ✅ Keep components focused and composable

## Common Migration Patterns

**v8 → v9 Quick Reference:**
- `ButtonGroup` → `<div style={{ display: 'flex', gap: '8px' }}>` with individual Buttons
- `getColor('blue', 600, theme)` → `getColor({ theme, hue: 'blue', shade: 600 })`
- `getColor('background', theme)` → `getColor({ theme, variable: 'background.default' })`
- Individual imports → dot-notation: `Modal.Header`, `Field.Label`, `Grid.Row`
- `Colorpicker` → `ColorPicker`, `Datepicker` → `DatePicker`
- `popperModifiers` prop → removed (Floating UI handles positioning)

**See `migration-guide.md` for complete reference.**

## Error Handling

If encountering issues:
1. Check Garden version compatibility
2. Verify styled-components version (v9 requires ^5.3.1)
3. Ensure ThemeProvider is present
4. Check for peer dependency issues
5. Look for conflicting CSS
6. Verify TypeScript types are correct

## Limitations

- Garden v9 requires React 16.8+ (hooks support)
- styled-components ^5.3.1 required for v9
- Some v8 components have no v9 equivalent (Sidebar, Subnav)
- CodeBlock language support reduced in v9 (32 → 13 languages)
- Floating UI has different positioning behavior than Popper

## Success Criteria

Your generated code should:
- ✅ Use correct Garden version syntax
- ✅ Include proper TypeScript types
- ✅ Have accessibility attributes
- ✅ Follow Garden's patterns and conventions
- ✅ Be production-ready
- ✅ Include helpful comments
- ✅ Work without modification
- ✅ Pass accessibility checks
- ✅ Support theming
- ✅ Be maintainable

---

Remember: Always detect the Garden version first, ask for confirmation before installing packages, reference pattern files for examples, use templates for code generation, and generate production-ready code that follows Garden's best practices.