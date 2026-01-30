# Garden Theming Patterns

Comprehensive guide to theming Garden components.

## Core Concepts

### ThemeProvider

The `ThemeProvider` is required at the root of your application to provide theme context to all Garden components.

```tsx
import { ThemeProvider } from '@zendeskgarden/react-theming';

function App() {
  return (
    <ThemeProvider>
      <YourApp />
    </ThemeProvider>
  );
}
```

### DEFAULT_THEME

Garden provides a default theme object with all necessary properties:

```tsx
import { ThemeProvider, DEFAULT_THEME } from '@zendeskgarden/react-theming';

console.log(DEFAULT_THEME);
// {
//   space: {...},
//   colors: {...},
//   fonts: {...},
//   fontSizes: {...},
//   lineHeights: {...},
//   borders: {...},
//   borderRadius: {...},
//   shadows: {...},
//   rtl: false,
// }
```

### PALETTE

The palette system provides a comprehensive color system with support for light and dark modes:

```tsx
import { PALETTE } from '@zendeskgarden/react-theming';

console.log(PALETTE);
// {
//   black: '#000',
//   white: '#fff',
//   blue: { 100: '...', 200: '...', ..., 900: '...' },
//   red: { ... },
//   yellow: { ... },
//   green: { ... },
//   // ... and more
// }
```

## Basic Theming Setup

### Minimal Setup

```tsx
import React from 'react';
import ReactDOM from 'react-dom/client';
import { ThemeProvider } from '@zendeskgarden/react-theming';
import App from './App';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ThemeProvider>
      <App />
    </ThemeProvider>
  </React.StrictMode>
);
```

### With Custom Theme

```tsx
import { ThemeProvider, DEFAULT_THEME } from '@zendeskgarden/react-theming';

const customTheme = {
  ...DEFAULT_THEME,
  colors: {
    ...DEFAULT_THEME.colors,
    primaryHue: 'teal', // Change primary color
  },
};

function App() {
  return (
    <ThemeProvider theme={customTheme}>
      <YourApp />
    </ThemeProvider>
  );
}
```

## Dark Mode Support

### Using ColorSchemeProvider

```tsx
import { ColorSchemeProvider, ThemeProvider } from '@zendeskgarden/react-theming';

function App() {
  return (
    <ColorSchemeProvider colorScheme="dark">
      <ThemeProvider>
        <YourApp />
      </ThemeProvider>
    </ColorSchemeProvider>
  );
}
```

### User-Controlled Dark Mode

```tsx
import { useState } from 'react';
import { ColorSchemeProvider, ThemeProvider } from '@zendeskgarden/react-theming';
import { Toggle, Field } from '@zendeskgarden/react-forms';

function App() {
  const [isDark, setIsDark] = useState(false);

  return (
    <ColorSchemeProvider colorScheme={isDark ? 'dark' : 'light'}>
      <ThemeProvider>
        <Field>
          <Toggle checked={isDark} onChange={(e) => setIsDark(e.target.checked)}>
            <Field.Label>Dark Mode</Field.Label>
          </Toggle>
        </Field>
        <YourApp />
      </ThemeProvider>
    </ColorSchemeProvider>
  );
}
```

### System Preference Detection

```tsx
import { useEffect, useState } from 'react';
import { ColorSchemeProvider, ThemeProvider } from '@zendeskgarden/react-theming';

function App() {
  const [colorScheme, setColorScheme] = useState<'light' | 'dark'>('light');

  useEffect(() => {
    // Check system preference
    const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    setColorScheme(isDark ? 'dark' : 'light');

    // Listen for changes
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (e: MediaQueryListEvent) => {
      setColorScheme(e.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  return (
    <ColorSchemeProvider colorScheme={colorScheme}>
      <ThemeProvider>
        <YourApp />
      </ThemeProvider>
    </ColorSchemeProvider>
  );
}
```

### Persistent Dark Mode (localStorage)

```tsx
import { useEffect, useState } from 'react';
import { ColorSchemeProvider, ThemeProvider } from '@zendeskgarden/react-theming';

const STORAGE_KEY = 'garden-color-scheme';

function App() {
  const [colorScheme, setColorScheme] = useState<'light' | 'dark'>(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'dark' || stored === 'light') return stored;

    // Default to system preference
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  });

  const toggleColorScheme = () => {
    setColorScheme(prev => {
      const next = prev === 'dark' ? 'light' : 'dark';
      localStorage.setItem(STORAGE_KEY, next);
      return next;
    });
  };

  return (
    <ColorSchemeProvider colorScheme={colorScheme}>
      <ThemeProvider>
        <button onClick={toggleColorScheme}>
          Toggle {colorScheme === 'dark' ? 'Light' : 'Dark'} Mode
        </button>
        <YourApp />
      </ThemeProvider>
    </ColorSchemeProvider>
  );
}
```

## Custom Colors

### Using getColor Utility

```tsx
import { useTheme, getColor } from '@zendeskgarden/react-theming';
import styled from 'styled-components';

const CustomButton = styled.button`
  background-color: ${props => getColor({ theme: props.theme, hue: 'blue', shade: 600 })};
  color: ${props => getColor({ theme: props.theme, variable: 'background.default' })};

  &:hover {
    background-color: ${props => getColor({ theme: props.theme, hue: 'blue', shade: 700 })};
  }
`;

export const MyComponent = () => {
  return <CustomButton>Click me</CustomButton>;
};
```

### Using Theme in Components

```tsx
import { useTheme } from '@zendeskgarden/react-theming';

export const ThemedComponent = () => {
  const theme = useTheme();

  return (
    <div style={{ padding: theme.space.md, color: theme.colors.foreground }}>
      Themed content
    </div>
  );
};
```

### Custom Color Palette

```tsx
import { ThemeProvider, DEFAULT_THEME, PALETTE } from '@zendeskgarden/react-theming';

const customTheme = {
  ...DEFAULT_THEME,
  colors: {
    ...DEFAULT_THEME.colors,
    // Override primary color
    primaryHue: 'purple',

    // Add custom colors
    brand: {
      primary: '#6C47FF',
      secondary: '#FF6C47',
    },
  },
  // Extend palette
  palette: {
    ...PALETTE,
    purple: {
      100: '#F3EDFF',
      200: '#E6DBFF',
      300: '#D1BBFF',
      400: '#BB9AFF',
      500: '#A478FF',
      600: '#6C47FF', // Primary
      700: '#5430CC',
      800: '#3D2399',
      900: '#271666',
    },
  },
};

function App() {
  return (
    <ThemeProvider theme={customTheme}>
      <YourApp />
    </ThemeProvider>
  );
}
```

## RTL (Right-to-Left) Support

### Enabling RTL

```tsx
import { ThemeProvider, DEFAULT_THEME } from '@zendeskgarden/react-theming';

const rtlTheme = {
  ...DEFAULT_THEME,
  rtl: true,
};

function App() {
  return (
    <ThemeProvider theme={rtlTheme}>
      <YourApp />
    </ThemeProvider>
  );
}
```

### RTL-Aware Styles

```tsx
import styled from 'styled-components';
import { getColor, focusStyles } from '@zendeskgarden/react-theming';

const Container = styled.div`
  /* Use logical properties for automatic RTL support */
  padding-inline-start: ${props => props.theme.space.md};
  padding-inline-end: ${props => props.theme.space.sm};

  /* These automatically flip in RTL */
  margin-inline-start: auto;
  border-inline-start: 1px solid ${props => getColor({ theme: props.theme, hue: 'grey', shade: 300 })};
`;
```

## Focus Styles

### Using focusStyles Utility

```tsx
import styled from 'styled-components';
import { focusStyles } from '@zendeskgarden/react-theming';

const FocusableElement = styled.button`
  /* Apply Garden's focus styles */
  ${props => focusStyles({
    theme: props.theme,
    inset: false, // Focus ring outside element
    condition: true, // Always apply (or use :focus-visible)
  })}

  &:focus {
    outline: none; // Remove default outline
  }
`;
```

### Custom Focus Styles

```tsx
import styled from 'styled-components';
import { getFocusBoxShadow } from '@zendeskgarden/react-theming';

const CustomFocusButton = styled.button`
  &:focus-visible {
    outline: none;
    box-shadow: ${props => getFocusBoxShadow({
      theme: props.theme,
      hue: 'blue',
      shade: 600,
      inset: false,
    })};
  }
`;
```

## Spacing System

```tsx
import { useTheme } from '@zendeskgarden/react-theming';

export const SpacingExample = () => {
  const theme = useTheme();

  return (
    <div>
      {/* Available spacing values */}
      <div style={{ padding: theme.space.xxs }}>XXS: 4px</div>
      <div style={{ padding: theme.space.xs }}>XS: 8px</div>
      <div style={{ padding: theme.space.sm }}>SM: 12px</div>
      <div style={{ padding: theme.space.md }}>MD: 16px</div>
      <div style={{ padding: theme.space.lg }}>LG: 20px</div>
      <div style={{ padding: theme.space.xl }}>XL: 24px</div>
      <div style={{ padding: theme.space.xxl }}>XXL: 32px</div>
    </div>
  );
};
```

## Typography System

```tsx
import styled from 'styled-components';

const Heading = styled.h1`
  font-family: ${props => props.theme.fonts.system};
  font-size: ${props => props.theme.fontSizes.xl};
  line-height: ${props => props.theme.lineHeights.xl};
  font-weight: ${props => props.theme.fontWeights.semibold};
`;

const Body = styled.p`
  font-family: ${props => props.theme.fonts.system};
  font-size: ${props => props.theme.fontSizes.md};
  line-height: ${props => props.theme.lineHeights.md};
  font-weight: ${props => props.theme.fontWeights.regular};
`;

const Code = styled.code`
  font-family: ${props => props.theme.fonts.mono};
  font-size: ${props => props.theme.fontSizes.sm};
`;
```

## Border Radius System

```tsx
import styled from 'styled-components';

const RoundedBox = styled.div`
  /* Available border radius values */
  border-radius: ${props => props.theme.borderRadius.sm}; // 2px
  border-radius: ${props => props.theme.borderRadius.md}; // 4px
  border-radius: ${props => props.theme.borderRadius.lg}; // 8px
  border-radius: ${props => props.theme.borderRadius.xl}; // 16px

  /* Full rounded (pill shape) */
  border-radius: ${props => props.theme.borderRadius.full}; // 9999px
`;
```

## Shadow System

```tsx
import styled from 'styled-components';

const ShadowCard = styled.div`
  /* Available shadow values */
  box-shadow: ${props => props.theme.shadows.sm}; // Small shadow
  box-shadow: ${props => props.theme.shadows.md}; // Medium shadow
  box-shadow: ${props => props.theme.shadows.lg}; // Large shadow
`;
```

## Styled Components Integration

### Complete Example

```tsx
import styled from 'styled-components';
import { getColor, focusStyles } from '@zendeskgarden/react-theming';
import { Button } from '@zendeskgarden/react-buttons';

// Custom styled component using theme
const Card = styled.div`
  background-color: ${props => getColor({ theme: props.theme, variable: 'background.default' })};
  border: ${props => props.theme.borders.sm} ${props => getColor({ theme: props.theme, hue: 'grey', shade: 300 })};
  border-radius: ${props => props.theme.borderRadius.md};
  padding: ${props => props.theme.space.lg};
  box-shadow: ${props => props.theme.shadows.md};

  /* RTL support */
  margin-inline-start: ${props => props.theme.space.md};
`;

const CardTitle = styled.h2`
  font-family: ${props => props.theme.fonts.system};
  font-size: ${props => props.theme.fontSizes.xl};
  font-weight: ${props => props.theme.fontWeights.semibold};
  line-height: ${props => props.theme.lineHeights.xl};
  color: ${props => getColor({ theme: props.theme, variable: 'foreground.default' })};
  margin-bottom: ${props => props.theme.space.md};
`;

const CardBody = styled.div`
  font-size: ${props => props.theme.fontSizes.md};
  line-height: ${props => props.theme.lineHeights.md};
  color: ${props => getColor({ theme: props.theme, variable: 'foreground.subtle' })};
  margin-bottom: ${props => props.theme.space.lg};
`;

const CardLink = styled.a`
  color: ${props => getColor({ theme: props.theme, hue: 'blue', shade: 600 })};
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }

  ${props => focusStyles({ theme: props.theme })}
`;

export const ThemedCard = () => {
  return (
    <Card>
      <CardTitle>Themed Card</CardTitle>
      <CardBody>
        This card uses Garden's theming system for colors, spacing, typography, and more.
      </CardBody>
      <Button isPrimary>Learn More</Button>
      <CardLink href="#" style={{ marginLeft: '12px' }}>
        View Details
      </CardLink>
    </Card>
  );
};
```

## Theme Context Hook

### useTheme Hook

```tsx
import { useTheme } from '@zendeskgarden/react-theming';

export const ComponentWithTheme = () => {
  const theme = useTheme();

  return (
    <div
      style={{
        padding: theme.space.md,
        borderRadius: theme.borderRadius.md,
        backgroundColor: theme.palette.blue[100],
      }}
    >
      Themed content
    </div>
  );
};
```

## Migration from v8 Theming

### getColor Migration

```tsx
// v8
import { getColorV8 } from '@zendeskgarden/react-theming';
const color = getColorV8('blue', 600, theme);

// v9
import { getColor } from '@zendeskgarden/react-theming';
const color = getColor({ theme, hue: 'blue', shade: 600 });

// v9 with color variable
const bgColor = getColor({ theme, variable: 'background.default' });
```

### Color Variables (v9)

```tsx
// Available color variables in v9
getColor({ theme, variable: 'background.default' })
getColor({ theme, variable: 'background.subtle' })
getColor({ theme, variable: 'background.emphasis' })
getColor({ theme, variable: 'foreground.default' })
getColor({ theme, variable: 'foreground.subtle' })
getColor({ theme, variable: 'foreground.emphasis' })
getColor({ theme, variable: 'border.default' })
getColor({ theme, variable: 'border.subtle' })
getColor({ theme, variable: 'border.emphasis' })
```

## Best Practices

### ✅ Do's

1. **Always wrap app in ThemeProvider**
   ```tsx
   <ThemeProvider>
     <App />
   </ThemeProvider>
   ```

2. **Use theme utilities for consistency**
   ```tsx
   const color = getColor({ theme, hue: 'blue', shade: 600 });
   ```

3. **Leverage spacing system**
   ```tsx
   padding: ${props => props.theme.space.md}
   ```

4. **Use color variables when possible**
   ```tsx
   getColor({ theme, variable: 'background.default' })
   ```

5. **Support RTL with logical properties**
   ```tsx
   margin-inline-start: ${props => props.theme.space.md}
   ```

### ❌ Don'ts

1. **Don't hardcode colors**
   ```tsx
   // NO
   background-color: #0F3554;

   // YES
   background-color: ${props => getColor({ theme: props.theme, hue: 'blue', shade: 600 })}
   ```

2. **Don't hardcode spacing**
   ```tsx
   // NO
   padding: 16px;

   // YES
   padding: ${props => props.theme.space.md}
   ```

3. **Don't skip ThemeProvider**
   ```tsx
   // NO - Components need theme context
   <Button>Click me</Button>

   // YES
   <ThemeProvider>
     <Button>Click me</Button>
   </ThemeProvider>
   ```

4. **Don't use v8 utilities in v9**
   ```tsx
   // NO - v8 utility
   getColorV8('blue', 600, theme)

   // YES - v9 utility
   getColor({ theme, hue: 'blue', shade: 600 })
   ```

## Complete Theming Example

```tsx
import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom/client';
import styled from 'styled-components';
import {
  ThemeProvider,
  ColorSchemeProvider,
  DEFAULT_THEME,
  PALETTE,
  getColor,
  focusStyles,
} from '@zendeskgarden/react-theming';
import { Button } from '@zendeskgarden/react-buttons';
import { Toggle, Field } from '@zendeskgarden/react-forms';

// Custom theme with branding
const customTheme = {
  ...DEFAULT_THEME,
  colors: {
    ...DEFAULT_THEME.colors,
    primaryHue: 'purple',
  },
  palette: {
    ...PALETTE,
    purple: {
      100: '#F3EDFF',
      600: '#6C47FF',
      700: '#5430CC',
    },
  },
};

// Styled components using theme
const Container = styled.div`
  min-height: 100vh;
  padding: ${props => props.theme.space.xl};
  background-color: ${props => getColor({ theme: props.theme, variable: 'background.default' })};
  color: ${props => getColor({ theme: props.theme, variable: 'foreground.default' })};
  transition: background-color 0.3s, color 0.3s;
`;

const Title = styled.h1`
  font-size: ${props => props.theme.fontSizes.xxl};
  margin-bottom: ${props => props.theme.space.lg};
`;

function App() {
  const [colorScheme, setColorScheme] = useState<'light' | 'dark'>('light');

  useEffect(() => {
    const stored = localStorage.getItem('color-scheme');
    if (stored === 'dark' || stored === 'light') {
      setColorScheme(stored);
    }
  }, []);

  const toggleColorScheme = (checked: boolean) => {
    const newScheme = checked ? 'dark' : 'light';
    setColorScheme(newScheme);
    localStorage.setItem('color-scheme', newScheme);
  };

  return (
    <ColorSchemeProvider colorScheme={colorScheme}>
      <ThemeProvider theme={customTheme}>
        <Container>
          <Title>Garden Themed App</Title>

          <Field>
            <Toggle
              checked={colorScheme === 'dark'}
              onChange={(e) => toggleColorScheme(e.target.checked)}
            >
              <Field.Label>Dark Mode</Field.Label>
            </Toggle>
          </Field>

          <Button isPrimary style={{ marginTop: '20px' }}>
            Themed Button
          </Button>
        </Container>
      </ThemeProvider>
    </ColorSchemeProvider>
  );
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
```

## Resources

- [Garden Theming Documentation](https://garden.zendesk.com/components/theme-provider)
- [Design Tokens](https://garden.zendesk.com/design/color)
- [Color Palette](https://garden.zendesk.com/components/palette)