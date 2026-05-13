# Garden Forms Patterns

Comprehensive patterns for working with forms in Garden React components.

## Core Components

### Field Component (`@zendeskgarden/react-forms`)

The `Field` component is the semantic wrapper for form inputs. It provides context for labels, hints, and validation messages.

```tsx
import { Field, Input } from '@zendeskgarden/react-forms';

<Field>
  <Field.Label>Email Address</Field.Label>
  <Input type="email" />
</Field>
```

## Common Form Patterns

### Basic Input with Label

```tsx
<Field>
  <Field.Label>Username</Field.Label>
  <Input placeholder="Enter username" />
</Field>
```

### Input with Hint

```tsx
<Field>
  <Field.Label>Password</Field.Label>
  <Field.Hint>Must be at least 8 characters</Field.Hint>
  <Input type="password" />
</Field>
```

### Input with Validation

```tsx
const [email, setEmail] = useState('');
const [validation, setValidation] = useState<'success' | 'error' | undefined>();

<Field>
  <Field.Label>Email</Field.Label>
  <Input
    type="email"
    value={email}
    validation={validation}
    onChange={(e) => {
      const value = e.target.value;
      setEmail(value);

      if (value) {
        const isValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
        setValidation(isValid ? 'success' : 'error');
      } else {
        setValidation(undefined);
      }
    }}
  />
  {validation === 'error' && (
    <Field.Message validation="error">
      Please enter a valid email address
    </Field.Message>
  )}
  {validation === 'success' && (
    <Field.Message validation="success">
      Email is valid
    </Field.Message>
  )}
</Field>
```

### Required Field

```tsx
<Field>
  <Field.Label>
    Full Name
    <Field.Label.Required />
  </Field.Label>
  <Input required aria-required="true" />
</Field>
```

### Disabled Field

```tsx
<Field>
  <Field.Label>Disabled Input</Field.Label>
  <Input disabled value="Cannot edit this" />
</Field>
```

### Textarea

```tsx
<Field>
  <Field.Label>Description</Field.Label>
  <Field.Hint>Provide a detailed description (max 500 characters)</Field.Hint>
  <Textarea
    rows={4}
    maxLength={500}
    placeholder="Enter description..."
  />
</Field>
```

## Input Groups

### Start Icon

```tsx
import { InputGroup } from '@zendeskgarden/react-forms';
import { ReactComponent as SearchIcon } from '@zendeskgarden/svg-icons/src/16/search-stroke.svg';

<Field>
  <Field.Label>Search</Field.Label>
  <InputGroup>
    <InputGroup.StartIcon>
      <SearchIcon />
    </InputGroup.StartIcon>
    <Input placeholder="Search..." />
  </InputGroup>
</Field>
```

### End Icon (Clear Button)

```tsx
import { ReactComponent as XIcon } from '@zendeskgarden/svg-icons/src/16/x-stroke.svg';

const [value, setValue] = useState('');

<Field>
  <Field.Label>Search</Field.Label>
  <InputGroup>
    <Input
      value={value}
      onChange={(e) => setValue(e.target.value)}
      placeholder="Type to search..."
    />
    {value && (
      <InputGroup.EndIcon isButton>
        <Button
          isBasic
          isPill
          size="small"
          onClick={() => setValue('')}
          aria-label="Clear search"
        >
          <XIcon />
        </Button>
      </InputGroup.EndIcon>
    )}
  </InputGroup>
</Field>
```

### With Button

```tsx
<Field>
  <Field.Label>Invite User</Field.Label>
  <InputGroup>
    <Input type="email" placeholder="email@example.com" />
    <Button isPrimary>Send Invite</Button>
  </InputGroup>
</Field>
```

## Select & Combobox

### Basic Select

```tsx
import { Field, Select } from '@zendeskgarden/react-forms';

<Field>
  <Field.Label>Country</Field.Label>
  <Select>
    <option value="">Select a country</option>
    <option value="us">United States</option>
    <option value="ca">Canada</option>
    <option value="mx">Mexico</option>
  </Select>
</Field>
```

### Combobox (Autocomplete)

```tsx
import { Combobox, Field } from '@zendeskgarden/react-dropdowns';

const options = ['Apple', 'Banana', 'Cherry', 'Date', 'Elderberry'];
const [selectedItem, setSelectedItem] = useState('');
const [inputValue, setInputValue] = useState('');

const filteredOptions = options.filter(option =>
  option.toLowerCase().includes(inputValue.toLowerCase())
);

<Field>
  <Field.Label>Favorite Fruit</Field.Label>
  <Combobox
    inputValue={inputValue}
    selectionValue={selectedItem}
    onSelect={setSelectedItem}
    onInputValueChange={setInputValue}
  >
    <Combobox.Input />
    <Combobox.Listbox>
      {filteredOptions.length === 0 ? (
        <Combobox.Message>No matches found</Combobox.Message>
      ) : (
        filteredOptions.map(option => (
          <Combobox.Option key={option} value={option}>
            {option}
          </Combobox.Option>
        ))
      )}
    </Combobox.Listbox>
  </Combobox>
</Field>
```

### Combobox with Groups

```tsx
<Combobox>
  <Combobox.Input />
  <Combobox.Listbox>
    <Combobox.OptGroup legend="Fruits">
      <Combobox.Option value="apple">Apple</Combobox.Option>
      <Combobox.Option value="banana">Banana</Combobox.Option>
    </Combobox.OptGroup>
    <Combobox.OptGroup legend="Vegetables">
      <Combobox.Option value="carrot">Carrot</Combobox.Option>
      <Combobox.Option value="broccoli">Broccoli</Combobox.Option>
    </Combobox.OptGroup>
  </Combobox.Listbox>
</Combobox>
```

## Checkboxes & Radios

### Basic Checkbox

```tsx
import { Checkbox, Field } from '@zendeskgarden/react-forms';

const [checked, setChecked] = useState(false);

<Field>
  <Checkbox checked={checked} onChange={(e) => setChecked(e.target.checked)}>
    <Field.Label>I agree to the terms and conditions</Field.Label>
  </Checkbox>
</Field>
```

### Checkbox Group

```tsx
const [selectedOptions, setSelectedOptions] = useState<string[]>([]);

const handleCheckboxChange = (value: string, checked: boolean) => {
  setSelectedOptions(prev =>
    checked ? [...prev, value] : prev.filter(v => v !== value)
  );
};

<fieldset>
  <legend>Select Features</legend>
  <Field>
    <Checkbox
      checked={selectedOptions.includes('feature1')}
      onChange={(e) => handleCheckboxChange('feature1', e.target.checked)}
    >
      <Field.Label>Feature 1</Field.Label>
    </Checkbox>
  </Field>
  <Field>
    <Checkbox
      checked={selectedOptions.includes('feature2')}
      onChange={(e) => handleCheckboxChange('feature2', e.target.checked)}
    >
      <Field.Label>Feature 2</Field.Label>
    </Checkbox>
  </Field>
</fieldset>
```

### Radio Group

```tsx
import { Radio, Field } from '@zendeskgarden/react-forms';

const [selected, setSelected] = useState('option1');

<fieldset>
  <legend>Choose an Option</legend>
  <Field>
    <Radio
      name="options"
      value="option1"
      checked={selected === 'option1'}
      onChange={(e) => setSelected(e.target.value)}
    >
      <Field.Label>Option 1</Field.Label>
    </Radio>
  </Field>
  <Field>
    <Radio
      name="options"
      value="option2"
      checked={selected === 'option2'}
      onChange={(e) => setSelected(e.target.value)}
    >
      <Field.Label>Option 2</Field.Label>
    </Radio>
  </Field>
</fieldset>
```

### Radio with Hints

```tsx
<Field>
  <Radio name="plan" value="basic" checked={plan === 'basic'}>
    <Field.Label>Basic Plan</Field.Label>
    <Field.Hint>Perfect for individuals - $10/month</Field.Hint>
  </Radio>
</Field>
<Field>
  <Radio name="plan" value="pro" checked={plan === 'pro'}>
    <Field.Label>Pro Plan</Field.Label>
    <Field.Hint>Best for teams - $50/month</Field.Hint>
  </Radio>
</Field>
```

## Toggle Switches

### Basic Toggle

```tsx
import { Toggle, Field } from '@zendeskgarden/react-forms';

const [enabled, setEnabled] = useState(false);

<Field>
  <Toggle checked={enabled} onChange={(e) => setEnabled(e.target.checked)}>
    <Field.Label>Enable notifications</Field.Label>
  </Toggle>
</Field>
```

### Toggle with Hint

```tsx
<Field>
  <Toggle checked={darkMode} onChange={(e) => setDarkMode(e.target.checked)}>
    <Field.Label>Dark Mode</Field.Label>
    <Field.Hint>Easier on the eyes in low light</Field.Hint>
  </Toggle>
</Field>
```

## File Upload

```tsx
import { FileUpload, Field } from '@zendeskgarden/react-forms';

const [files, setFiles] = useState<File[]>([]);

<Field>
  <Field.Label>Upload Documents</Field.Label>
  <FileUpload
    isDraggable
    onChange={(e) => {
      if (e.target.files) {
        setFiles(Array.from(e.target.files));
      }
    }}
  >
    <FileUpload.Label>
      Drag and drop files here or click to browse
    </FileUpload.Label>
    <FileUpload.Hint>Maximum file size: 10MB</FileUpload.Hint>
  </FileUpload>
  {files.length > 0 && (
    <div>
      <p>Selected files:</p>
      <ul>
        {files.map((file, index) => (
          <li key={index}>{file.name}</li>
        ))}
      </ul>
    </div>
  )}
</Field>
```

## Range Slider

```tsx
import { Range, Field } from '@zendeskgarden/react-forms';

const [value, setValue] = useState(50);

<Field>
  <Field.Label>Volume: {value}%</Field.Label>
  <Range
    min={0}
    max={100}
    step={1}
    value={value}
    onChange={(e) => setValue(Number(e.target.value))}
  />
</Field>
```

## Complete Form Example

```tsx
import React, { useState, FormEvent } from 'react';
import { Field, Input, Textarea, Select, Checkbox, Toggle } from '@zendeskgarden/react-forms';
import { Button } from '@zendeskgarden/react-buttons';
import { Grid } from '@zendeskgarden/react-grid';

interface FormData {
  name: string;
  email: string;
  phone: string;
  country: string;
  message: string;
  subscribe: boolean;
  notifications: boolean;
}

export const ContactForm: React.FC = () => {
  const [formData, setFormData] = useState<FormData>({
    name: '',
    email: '',
    phone: '',
    country: '',
    message: '',
    subscribe: false,
    notifications: false,
  });

  const [validation, setValidation] = useState<{
    [key: string]: 'success' | 'error' | undefined;
  }>({});

  const validateEmail = (email: string) => {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  };

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();

    const newValidation: typeof validation = {};

    if (!formData.name) newValidation.name = 'error';
    if (!validateEmail(formData.email)) newValidation.email = 'error';
    if (!formData.message) newValidation.message = 'error';

    if (Object.keys(newValidation).length === 0) {
      console.log('Form submitted:', formData);
      // Handle form submission
    } else {
      setValidation(newValidation);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <Grid.Row>
        <Grid.Col md={6}>
          <Field>
            <Field.Label>
              Name
              <Field.Label.Required />
            </Field.Label>
            <Input
              value={formData.name}
              validation={validation.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
            {validation.name === 'error' && (
              <Field.Message validation="error">Name is required</Field.Message>
            )}
          </Field>
        </Grid.Col>

        <Grid.Col md={6}>
          <Field>
            <Field.Label>
              Email
              <Field.Label.Required />
            </Field.Label>
            <Input
              type="email"
              value={formData.email}
              validation={validation.email}
              onChange={(e) => setFormData({ ...formData, email: e.target.value })}
              required
            />
            {validation.email === 'error' && (
              <Field.Message validation="error">Valid email is required</Field.Message>
            )}
          </Field>
        </Grid.Col>
      </Grid.Row>

      <Field>
        <Field.Label>Phone</Field.Label>
        <Input
          type="tel"
          value={formData.phone}
          onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
        />
      </Field>

      <Field>
        <Field.Label>Country</Field.Label>
        <Select
          value={formData.country}
          onChange={(e) => setFormData({ ...formData, country: e.target.value })}
        >
          <option value="">Select a country</option>
          <option value="us">United States</option>
          <option value="ca">Canada</option>
          <option value="uk">United Kingdom</option>
        </Select>
      </Field>

      <Field>
        <Field.Label>
          Message
          <Field.Label.Required />
        </Field.Label>
        <Textarea
          value={formData.message}
          validation={validation.message}
          onChange={(e) => setFormData({ ...formData, message: e.target.value })}
          rows={4}
          required
        />
        {validation.message === 'error' && (
          <Field.Message validation="error">Message is required</Field.Message>
        )}
      </Field>

      <Field>
        <Checkbox
          checked={formData.subscribe}
          onChange={(e) => setFormData({ ...formData, subscribe: e.target.checked })}
        >
          <Field.Label>Subscribe to newsletter</Field.Label>
        </Checkbox>
      </Field>

      <Field>
        <Toggle
          checked={formData.notifications}
          onChange={(e) => setFormData({ ...formData, notifications: e.target.checked })}
        >
          <Field.Label>Enable email notifications</Field.Label>
        </Toggle>
      </Field>

      <Button type="submit" isPrimary>
        Submit
      </Button>
    </form>
  );
};
```

## Form Validation Patterns

### Real-time Validation

```tsx
const [value, setValue] = useState('');
const [validation, setValidation] = useState<'success' | 'error' | undefined>();

const validate = (val: string) => {
  if (!val) return undefined;
  return val.length >= 8 ? 'success' : 'error';
};

<Input
  value={value}
  validation={validation}
  onChange={(e) => {
    const newValue = e.target.value;
    setValue(newValue);
    setValidation(validate(newValue));
  }}
/>
```

### On-Blur Validation

```tsx
const [value, setValue] = useState('');
const [validation, setValidation] = useState<'success' | 'error' | undefined>();
const [touched, setTouched] = useState(false);

<Input
  value={value}
  validation={touched ? validation : undefined}
  onChange={(e) => setValue(e.target.value)}
  onBlur={() => {
    setTouched(true);
    setValidation(validateValue(value));
  }}
/>
```

### Submit Validation

```tsx
const handleSubmit = (e: FormEvent) => {
  e.preventDefault();

  const errors = validateForm(formData);

  if (Object.keys(errors).length === 0) {
    // Submit form
  } else {
    setValidation(errors);
  }
};
```

## Accessibility Best Practices

1. **Always use Field.Label**: Screen readers need labels
2. **Use required attribute**: Mark required fields
3. **Provide validation messages**: Explain what's wrong
4. **Use appropriate input types**: email, tel, url, etc.
5. **Group related fields**: Use fieldset and legend
6. **Provide hints**: Help users understand requirements
7. **Error identification**: Clearly mark and describe errors
8. **Focus management**: Move focus to first error on submit
9. **Keyboard navigation**: Ensure all controls are keyboard accessible
10. **ARIA attributes**: Use aria-required, aria-invalid, aria-describedby

## Common Mistakes to Avoid

❌ **Missing labels:**
```tsx
<Input placeholder="Email" /> // NO!
```

✅ **Always include labels:**
```tsx
<Field>
  <Field.Label>Email</Field.Label>
  <Input placeholder="e.g., user@example.com" />
</Field>
```

❌ **Placeholder as label:**
```tsx
<Input placeholder="Enter your password" /> // NO!
```

✅ **Use actual labels:**
```tsx
<Field>
  <Field.Label>Password</Field.Label>
  <Input type="password" placeholder="At least 8 characters" />
</Field>
```

❌ **No validation feedback:**
```tsx
<Input validation="error" /> // NO! Users don't know why
```

✅ **Provide clear error messages:**
```tsx
<Input validation="error" />
<Field.Message validation="error">
  Password must be at least 8 characters long
</Field.Message>
```