# Garden Accessibility Patterns

Comprehensive guide to building accessible interfaces with Garden React components.

## Core Principles

Garden components are built with accessibility as a priority, but developers must still:

1. **Provide meaningful labels** - All interactive elements need accessible names
2. **Maintain keyboard navigation** - All interactions must work without a mouse
3. **Use semantic HTML** - Leverage native accessibility features
4. **Manage focus appropriately** - Guide users through the interface
5. **Provide feedback** - Communicate state changes to assistive technologies
6. **Ensure color contrast** - Meet WCAG AA standards (4.5:1 for text)
7. **Support screen readers** - Test with actual screen reader software
8. **Handle errors gracefully** - Clearly identify and describe errors

## Buttons & Interactive Elements

### Button Labels

✅ **Always provide accessible text:**
```tsx
import { Button } from '@zendeskgarden/react-buttons';

// Text button - inherently accessible
<Button>Save Changes</Button>

// Icon button - needs aria-label
<Button aria-label="Close dialog">
  <XIcon />
</Button>

// Icon with text - text provides the label
<Button>
  <Button.StartIcon>
    <SaveIcon />
  </Button.StartIcon>
  Save
</Button>
```

❌ **Never create icon-only buttons without labels:**
```tsx
// BAD - no accessible name
<Button>
  <DeleteIcon />
</Button>

// GOOD - aria-label provides name
<Button aria-label="Delete item">
  <DeleteIcon />
</Button>
```

### Button States

```tsx
// Disabled button - automatically handled by Garden
<Button disabled>Can't Click</Button>

// Loading state - communicate to screen readers
<Button aria-busy={isLoading} disabled={isLoading}>
  {isLoading ? 'Saving...' : 'Save'}
</Button>

// Toggle button - indicate pressed state
<Button
  isPill
  isBasic={!isActive}
  isPrimary={isActive}
  aria-pressed={isActive}
  onClick={() => setIsActive(!isActive)}
>
  {isActive ? 'Active' : 'Inactive'}
</Button>
```

### Link vs Button

Use the right element for the job:

```tsx
// Navigation - use Anchor or Button isLink
<Anchor href="/dashboard">Go to Dashboard</Anchor>
<Button isLink href="/dashboard">Go to Dashboard</Button>

// Actions - use Button
<Button onClick={handleSave}>Save</Button>
<Button type="submit">Submit Form</Button>
```

## Forms & Inputs

### Labels Are Required

✅ **Every form control needs a label:**
```tsx
import { Field, Input } from '@zendeskgarden/react-forms';

<Field>
  <Field.Label>Email Address</Field.Label>
  <Input type="email" />
</Field>
```

❌ **Never rely on placeholders alone:**
```tsx
// BAD - placeholder is not a label
<Input placeholder="Enter your email" />

// GOOD - proper label provided
<Field>
  <Field.Label>Email Address</Field.Label>
  <Input type="email" placeholder="example@company.com" />
</Field>
```

### Required Fields

```tsx
<Field>
  <Field.Label>
    Password
    <Field.Label.Required /> {/* Visual asterisk */}
  </Field.Label>
  <Input
    type="password"
    required
    aria-required="true"
  />
</Field>
```

### Error Messaging

```tsx
const [email, setEmail] = useState('');
const [error, setError] = useState<string | null>(null);

<Field>
  <Field.Label>Email Address</Field.Label>
  <Input
    type="email"
    value={email}
    validation={error ? 'error' : undefined}
    aria-invalid={!!error}
    aria-describedby={error ? 'email-error' : undefined}
    onChange={(e) => {
      setEmail(e.target.value);
      // Validate...
    }}
  />
  {error && (
    <Field.Message id="email-error" validation="error">
      {error}
    </Field.Message>
  )}
</Field>
```

### Fieldset & Legend

Group related controls:

```tsx
<fieldset>
  <legend>Contact Preferences</legend>

  <Field>
    <Checkbox>
      <Field.Label>Email notifications</Field.Label>
    </Checkbox>
  </Field>

  <Field>
    <Checkbox>
      <Field.Label>SMS notifications</Field.Label>
    </Checkbox>
  </Field>
</fieldset>
```

### Radio Groups

```tsx
<fieldset>
  <legend>Select Plan</legend>

  <Field>
    <Radio name="plan" value="basic" checked={plan === 'basic'}>
      <Field.Label>Basic Plan</Field.Label>
      <Field.Hint>$10/month - Perfect for individuals</Field.Hint>
    </Radio>
  </Field>

  <Field>
    <Radio name="plan" value="pro" checked={plan === 'pro'}>
      <Field.Label>Pro Plan</Field.Label>
      <Field.Hint>$50/month - Best for teams</Field.Hint>
    </Radio>
  </Field>
</fieldset>
```

### Form Validation

```tsx
const handleSubmit = (e: FormEvent) => {
  e.preventDefault();

  const errors = validate(formData);

  if (Object.keys(errors).length > 0) {
    // Announce errors to screen readers
    const errorCount = Object.keys(errors).length;
    const announcement = `Form has ${errorCount} ${errorCount === 1 ? 'error' : 'errors'}`;

    // Create live region announcement
    const liveRegion = document.createElement('div');
    liveRegion.setAttribute('role', 'alert');
    liveRegion.setAttribute('aria-live', 'assertive');
    liveRegion.textContent = announcement;
    document.body.appendChild(liveRegion);

    setTimeout(() => document.body.removeChild(liveRegion), 1000);

    // Move focus to first error
    const firstErrorField = document.querySelector('[aria-invalid="true"]');
    if (firstErrorField instanceof HTMLElement) {
      firstErrorField.focus();
    }

    setErrors(errors);
  } else {
    // Submit form
  }
};
```

## Keyboard Navigation

### Tab Order

Ensure logical tab order:

```tsx
// Good - follows visual order
<form>
  <Field>
    <Field.Label>Name</Field.Label>
    <Input tabIndex={0} /> {/* First */}
  </Field>

  <Field>
    <Field.Label>Email</Field.Label>
    <Input tabIndex={0} /> {/* Second */}
  </Field>

  <Button type="submit" tabIndex={0}> {/* Third */}
    Submit
  </Button>
</form>

// Avoid tabIndex > 0 - it creates unpredictable order
```

### Skip Links

Provide navigation shortcuts:

```tsx
const SkipLink = styled.a`
  position: absolute;
  left: -9999px;
  top: 0;

  &:focus {
    left: 0;
    padding: ${props => props.theme.space.md};
    background: ${props => getColor({ theme: props.theme, hue: 'blue', shade: 600 })};
    color: white;
    z-index: 1000;
  }
`;

function App() {
  return (
    <>
      <SkipLink href="#main-content">
        Skip to main content
      </SkipLink>

      <Header />
      <Sidebar />

      <main id="main-content" tabIndex={-1}>
        <Content />
      </main>
    </>
  );
}
```

### Keyboard Shortcuts

```tsx
const MyComponent = () => {
  useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      // Cmd/Ctrl + S to save
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault();
        handleSave();
      }

      // Escape to close
      if (e.key === 'Escape') {
        handleClose();
      }
    };

    document.addEventListener('keydown', handleKeyPress);
    return () => document.removeEventListener('keydown', handleKeyPress);
  }, []);

  return (
    <div>
      <p>Press <Kbd>⌘S</Kbd> or <Kbd>Ctrl+S</Kbd> to save</p>
      <p>Press <Kbd>Esc</Kbd> to close</p>
    </div>
  );
};
```

## Modals & Dialogs

### Modal Accessibility

```tsx
import { Modal } from '@zendeskgarden/react-modals';
import { Button } from '@zendeskgarden/react-buttons';

const AccessibleModal = () => {
  const [isOpen, setIsOpen] = useState(false);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    // Focus close button when modal opens
    if (isOpen && closeButtonRef.current) {
      closeButtonRef.current.focus();
    }
  }, [isOpen]);

  return (
    <>
      <Button onClick={() => setIsOpen(true)}>
        Open Modal
      </Button>

      <Modal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        // Garden handles focus trap automatically
      >
        <Modal.Header>
          Accessible Modal
        </Modal.Header>

        <Modal.Body>
          <p>Content goes here. Focus is trapped within the modal.</p>
          <p>Press Escape or click the close button to dismiss.</p>
        </Modal.Body>

        <Modal.Footer>
          <Modal.FooterItem>
            <Button
              isBasic
              onClick={() => setIsOpen(false)}
              ref={closeButtonRef}
            >
              Cancel
            </Button>
          </Modal.FooterItem>
          <Modal.FooterItem>
            <Button isPrimary onClick={handleConfirm}>
              Confirm
            </Button>
          </Modal.FooterItem>
        </Modal.Footer>

        <Modal.Close aria-label="Close modal" />
      </Modal>
    </>
  );
};
```

### Alert Dialogs

For destructive actions:

```tsx
<Modal
  isOpen={isOpen}
  onClose={() => setIsOpen(false)}
  // Indicates this is an alert dialog
  role="alertdialog"
  aria-describedby="delete-description"
>
  <Modal.Header>Delete Account?</Modal.Header>

  <Modal.Body>
    <p id="delete-description">
      This action cannot be undone. Your account and all data will be permanently deleted.
    </p>
  </Modal.Body>

  <Modal.Footer>
    <Modal.FooterItem>
      <Button isBasic onClick={() => setIsOpen(false)}>
        Cancel
      </Button>
    </Modal.FooterItem>
    <Modal.FooterItem>
      <Button isPrimary isDanger onClick={handleDelete}>
        Delete Account
      </Button>
    </Modal.FooterItem>
  </Modal.Footer>
</Modal>
```

## Tooltips & Popovers

### Tooltip Accessibility

```tsx
import { Tooltip } from '@zendeskgarden/react-tooltips';
import { Button } from '@zendeskgarden/react-buttons';

// For supplementary info
<Tooltip content="Additional information about this feature">
  <Button>Hover for Info</Button>
</Tooltip>

// For icon buttons, use aria-label instead
<Button aria-label="Delete item">
  <DeleteIcon />
</Button>

// NOT a tooltip - essential information should be visible
```

### TooltipDialog for Interactive Content

```tsx
import { TooltipDialog } from '@zendeskgarden/react-modals';

<TooltipDialog
  isOpen={isOpen}
  onClose={() => setIsOpen(false)}
>
  <TooltipDialog.Title>User Settings</TooltipDialog.Title>
  <TooltipDialog.Body>
    <Field>
      <Toggle>
        <Field.Label>Enable notifications</Field.Label>
      </Toggle>
    </Field>
  </TooltipDialog.Body>
  <TooltipDialog.Close aria-label="Close dialog" />
</TooltipDialog>
```

## Notifications

### Alert Notifications

```tsx
import { Alert } from '@zendeskgarden/react-notifications';

<Alert type="success" role="status" aria-live="polite">
  <Alert.Title>Success!</Alert.Title>
  Your changes have been saved.
  <Alert.Close aria-label="Dismiss success notification" />
</Alert>

<Alert type="error" role="alert" aria-live="assertive">
  <Alert.Title>Error</Alert.Title>
  Failed to save changes. Please try again.
  <Alert.Close aria-label="Dismiss error notification" />
</Alert>
```

### Toast Notifications

```tsx
import { useToast, ToastProvider } from '@zendeskgarden/react-notifications';

function MyComponent() {
  const { addToast } = useToast();

  const showNotification = () => {
    addToast(
      ({ close }) => (
        <Notification type="success">
          <Notification.Title>Success!</Notification.Title>
          Operation completed successfully.
          <Notification.Close aria-label="Close notification" onClick={close} />
        </Notification>
      ),
      {
        placement: 'top',
        // Automatically dismissed after 5 seconds
        autoDismiss: 5000,
      }
    );
  };

  return <Button onClick={showNotification}>Show Toast</Button>;
}
```

## Tables

### Accessible Tables

```tsx
import { Table } from '@zendeskgarden/react-tables';

<Table>
  <Table.Caption>User List</Table.Caption>
  <Table.Head>
    <Table.HeaderRow>
      <Table.HeaderCell scope="col">Name</Table.HeaderCell>
      <Table.HeaderCell scope="col">Email</Table.HeaderCell>
      <Table.HeaderCell scope="col">Role</Table.HeaderCell>
      <Table.HeaderCell scope="col">
        <span className="visually-hidden">Actions</span>
      </Table.HeaderCell>
    </Table.HeaderRow>
  </Table.Head>
  <Table.Body>
    {users.map(user => (
      <Table.Row key={user.id}>
        <Table.Cell>{user.name}</Table.Cell>
        <Table.Cell>{user.email}</Table.Cell>
        <Table.Cell>{user.role}</Table.Cell>
        <Table.Cell>
          <Button size="small" aria-label={`Edit ${user.name}`}>
            Edit
          </Button>
        </Table.Cell>
      </Table.Row>
    ))}
  </Table.Body>
</Table>
```

### Sortable Tables

```tsx
<Table>
  <Table.Head>
    <Table.HeaderRow>
      <Table.SortableCell
        onClick={() => handleSort('name')}
        sort={sortColumn === 'name' ? sortDirection : undefined}
      >
        Name
      </Table.SortableCell>
      <Table.SortableCell
        onClick={() => handleSort('email')}
        sort={sortColumn === 'email' ? sortDirection : undefined}
      >
        Email
      </Table.SortableCell>
    </Table.HeaderRow>
  </Table.Head>
  <Table.Body>
    {/* Table rows */}
  </Table.Body>
</Table>
```

## Navigation

### Breadcrumbs

```tsx
import { Breadcrumb } from '@zendeskgarden/react-breadcrumbs';

<nav aria-label="Breadcrumb">
  <Breadcrumb>
    <Anchor href="/">Home</Anchor>
    <Anchor href="/products">Products</Anchor>
    <Anchor href="/products/widgets">Widgets</Anchor>
    <span aria-current="page">Blue Widget</span>
  </Breadcrumb>
</nav>
```

### Tabs

```tsx
import { Tabs } from '@zendeskgarden/react-tabs';

<Tabs selectedItem={selectedTab} onChange={setSelectedTab}>
  <Tabs.TabList>
    <Tabs.Tab item="overview">Overview</Tabs.Tab>
    <Tabs.Tab item="settings">Settings</Tabs.Tab>
    <Tabs.Tab item="billing">Billing</Tabs.Tab>
  </Tabs.TabList>

  <Tabs.TabPanel item="overview">
    <h2 id="overview-heading">Overview</h2>
    {/* Content */}
  </Tabs.TabPanel>

  <Tabs.TabPanel item="settings">
    <h2 id="settings-heading">Settings</h2>
    {/* Content */}
  </Tabs.TabPanel>

  <Tabs.TabPanel item="billing">
    <h2 id="billing-heading">Billing</h2>
    {/* Content */}
  </Tabs.TabPanel>
</Tabs>
```

## Live Regions

### Announcing Dynamic Content

```tsx
const LiveRegion = styled.div`
  position: absolute;
  left: -9999px;
  width: 1px;
  height: 1px;
  overflow: hidden;
`;

function SearchResults() {
  const [results, setResults] = useState<Result[]>([]);
  const [announcement, setAnnouncement] = useState('');

  useEffect(() => {
    if (results.length > 0) {
      setAnnouncement(`${results.length} results found`);
    } else {
      setAnnouncement('No results found');
    }
  }, [results]);

  return (
    <>
      <LiveRegion role="status" aria-live="polite" aria-atomic="true">
        {announcement}
      </LiveRegion>

      <div>
        {results.map(result => (
          <ResultCard key={result.id} result={result} />
        ))}
      </div>
    </>
  );
}
```

### Loading States

```tsx
import { Spinner, Inline } from '@zendeskgarden/react-loaders';

<div>
  {isLoading ? (
    <div role="status" aria-live="polite">
      <Inline>
        <Spinner size="medium" />
        <span>Loading data...</span>
      </Inline>
    </div>
  ) : (
    <DataDisplay data={data} />
  )}
</div>
```

## Focus Management

### Focus Styles

```tsx
import styled from 'styled-components';
import { focusStyles } from '@zendeskgarden/react-theming';

// Garden's built-in focus styles
const FocusableDiv = styled.div`
  &:focus {
    outline: none;
    ${props => focusStyles({
      theme: props.theme,
      inset: false,
    })}
  }
`;

// Usage
<FocusableDiv tabIndex={0} role="button" onClick={handleClick}>
  Clickable div with proper focus styles
</FocusableDiv>
```

### Managing Focus Programmatically

```tsx
function FormWithValidation() {
  const firstErrorRef = useRef<HTMLInputElement>(null);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();

    const errors = validate(formData);

    if (Object.keys(errors).length > 0) {
      // Move focus to first error
      firstErrorRef.current?.focus();
      setErrors(errors);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <Field>
        <Field.Label>Email</Field.Label>
        <Input
          ref={errors.email ? firstErrorRef : null}
          type="email"
          validation={errors.email ? 'error' : undefined}
        />
        {errors.email && (
          <Field.Message validation="error">{errors.email}</Field.Message>
        )}
      </Field>
      <Button type="submit">Submit</Button>
    </form>
  );
}
```

## Color Contrast

### Ensuring Sufficient Contrast

```tsx
import { getColor } from '@zendeskgarden/react-theming';
import styled from 'styled-components';

// Garden's palette has good contrast by default
const GoodContrast = styled.div`
  /* Blue 600 on white - meets WCAG AA (4.5:1) */
  background: ${props => getColor({ theme: props.theme, variable: 'background.default' })};
  color: ${props => getColor({ theme: props.theme, hue: 'blue', shade: 600 })};
`;

// For custom colors, verify contrast
const CustomText = styled.span<{ isImportant?: boolean }>`
  color: ${props => props.isImportant
    ? getColor({ theme: props.theme, hue: 'red', shade: 700 }) // Higher contrast
    : getColor({ theme: props.theme, hue: 'grey', shade: 600 })
  };
`;
```

## Screen Reader Only Content

```tsx
import styled from 'styled-components';

const VisuallyHidden = styled.span`
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
`;

// Usage
<Button>
  <DeleteIcon />
  <VisuallyHidden>Delete item</VisuallyHidden>
</Button>

// Or use aria-label
<Button aria-label="Delete item">
  <DeleteIcon />
</Button>
```

## Testing Accessibility

### Automated Testing with jest-axe

```tsx
import { render } from '@testing-library/react';
import { axe, toHaveNoViolations } from 'jest-axe';
import { ThemeProvider } from '@zendeskgarden/react-theming';
import { MyComponent } from './MyComponent';

expect.extend(toHaveNoViolations);

describe('MyComponent accessibility', () => {
  it('has no accessibility violations', async () => {
    const { container } = render(
      <ThemeProvider>
        <MyComponent />
      </ThemeProvider>
    );

    const results = await axe(container);
    expect(results).toHaveNoViolations();
  });
});
```

### Manual Testing Checklist

- [ ] Keyboard navigation works (Tab, Shift+Tab, Enter, Space, Arrow keys, Escape)
- [ ] All interactive elements are focusable
- [ ] Focus is visible (focus indicator)
- [ ] All images have alt text
- [ ] All form inputs have labels
- [ ] Color is not the only means of conveying information
- [ ] Text has sufficient contrast (4.5:1 minimum)
- [ ] Content is readable when zoomed to 200%
- [ ] Screen reader announces content correctly (test with NVDA, JAWS, VoiceOver)
- [ ] No keyboard traps
- [ ] Skip links work
- [ ] Error messages are clear and associated with fields
- [ ] Form validation is announced
- [ ] Dynamic content changes are announced (live regions)
- [ ] Modals trap focus and return focus on close
- [ ] Page has proper heading hierarchy (h1 → h2 → h3)

## Common Accessibility Mistakes

### ❌ Missing Labels

```tsx
// BAD
<Input placeholder="Search" />

// GOOD
<Field>
  <Field.Label>Search</Field.Label>
  <Input placeholder="e.g., product name" />
</Field>
```

### ❌ Div Buttons

```tsx
// BAD - not keyboard accessible
<div onClick={handleClick}>Click me</div>

// GOOD - proper button
<Button onClick={handleClick}>Click me</Button>
```

### ❌ Empty Links/Buttons

```tsx
// BAD
<button onClick={handleDelete}>
  <TrashIcon />
</button>

// GOOD
<Button onClick={handleDelete} aria-label="Delete item">
  <TrashIcon />
</Button>
```

### ❌ Poor Color Contrast

```tsx
// BAD - light grey on white
const BadText = styled.span`
  color: #ccc;
  background: #fff;
`;

// GOOD - sufficient contrast
const GoodText = styled.span`
  color: ${props => getColor({ theme: props.theme, hue: 'grey', shade: 700 })};
  background: ${props => getColor({ theme: props.theme, variable: 'background.default' })};
`;
```

## Resources

- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [Garden Accessibility](https://garden.zendesk.com/design/accessibility)
- [WebAIM](https://webaim.org/)
- [axe DevTools](https://www.deque.com/axe/devtools/)
- [ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/)