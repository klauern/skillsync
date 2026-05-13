# Garden Design Tokens Integration

Guide to integrating custom design tokens with Garden's theming system.

## What Are Design Tokens?

Design tokens are the visual design atoms of the design system — specifically, they are named entities that store visual design attributes. They maintain a scalable and consistent visual system for UI development.

**Examples:**
- Colors: `color.brand.primary`, `color.background.default`
- Spacing: `space.sm`, `space.md`, `space.lg`
- Typography: `font.size.body`, `font.family.heading`
- Borders: `border.radius.md`, `border.width.default`

## Garden's Token System

Garden uses a token-based theming system accessible through the theme object:

```tsx
import { useTheme } from '@zendeskgarden/react-theming';

const theme = useTheme();

// Access tokens
theme.space.md          // 16px
theme.colors.primaryHue // 'blue'
theme.fontSizes.lg      // 18px
theme.borderRadius.md   // 4px
```

## Integrating Custom Design Tokens

### Method 1: Extend Garden's Theme

The simplest approach - extend Garden's default theme with your tokens:

```tsx
import { ThemeProvider, DEFAULT_THEME, PALETTE } from '@zendeskgarden/react-theming';

// Your design tokens
const designTokens = {
  colors: {
    brand: {
      primary: '#6C47FF',
      secondary: '#FF6C47',
      tertiary: '#47D7FF',
    },
    semantic: {
      success: '#00C9A7',
      warning: '#FFC845',
      error: '#FF5C51',
      info: '#5393FF',
    },
  },
  spacing: {
    xs: '4px',
    sm: '8px',
    md: '16px',
    lg: '24px',
    xl: '32px',
    xxl: '48px',
  },
  typography: {
    fontFamily: {
      primary: "'Inter', -apple-system, BlinkMacSystemFont, sans-serif",
      mono: "'Fira Code', 'Consolas', monospace",
    },
    fontSize: {
      xs: '12px',
      sm: '14px',
      md: '16px',
      lg: '18px',
      xl: '24px',
      xxl: '32px',
    },
  },
  borderRadius: {
    sm: '2px',
    md: '4px',
    lg: '8px',
    xl: '12px',
    full: '9999px',
  },
};

// Merge with Garden theme
const customTheme = {
  ...DEFAULT_THEME,
  // Map your tokens to Garden's structure
  colors: {
    ...DEFAULT_THEME.colors,
    primaryHue: 'purple', // Use custom primary
  },
  space: {
    ...DEFAULT_THEME.space,
    // Garden uses base-4 spacing, adjust if needed
  },
  fonts: {
    ...DEFAULT_THEME.fonts,
    system: designTokens.typography.fontFamily.primary,
    mono: designTokens.typography.fontFamily.mono,
  },
  // Add custom token namespace
  customTokens: designTokens,
};

function App() {
  return (
    <ThemeProvider theme={customTheme}>
      <YourApp />
    </ThemeProvider>
  );
}
```

### Method 2: Style Dictionary Integration

For larger design systems, use [Style Dictionary](https://amzn.github.io/style-dictionary/) to generate tokens:

**1. Install Style Dictionary:**
```bash
bun add -D style-dictionary
```

**2. Create token definitions** (`tokens/colors.json`):
```json
{
  "color": {
    "brand": {
      "primary": { "value": "#6C47FF" },
      "secondary": { "value": "#FF6C47" }
    },
    "semantic": {
      "success": { "value": "#00C9A7" },
      "error": { "value": "#FF5C51" }
    }
  }
}
```

**3. Configure Style Dictionary** (`style-dictionary.config.js`):
```javascript
module.exports = {
  source: ['tokens/**/*.json'],
  platforms: {
    ts: {
      transformGroup: 'js',
      buildPath: 'src/tokens/',
      files: [{
        destination: 'tokens.ts',
        format: 'javascript/es6',
      }]
    }
  }
};
```

**4. Generate tokens:**
```bash
npx style-dictionary build
```

**5. Integrate with Garden:**
```tsx
import { ThemeProvider, DEFAULT_THEME } from '@zendeskgarden/react-theming';
import { tokens } from './tokens/tokens';

const theme = {
  ...DEFAULT_THEME,
  customTokens: tokens,
};

function App() {
  return (
    <ThemeProvider theme={theme}>
      <YourApp />
    </ThemeProvider>
  );
}
```

### Method 3: CSS Custom Properties

Use CSS variables for maximum flexibility:

**1. Define CSS custom properties:**
```css
/* tokens.css */
:root {
  /* Colors */
  --color-brand-primary: #6C47FF;
  --color-brand-secondary: #FF6C47;
  --color-semantic-success: #00C9A7;
  --color-semantic-error: #FF5C51;

  /* Spacing */
  --space-xs: 4px;
  --space-sm: 8px;
  --space-md: 16px;
  --space-lg: 24px;

  /* Typography */
  --font-family-primary: 'Inter', sans-serif;
  --font-size-sm: 14px;
  --font-size-md: 16px;
  --font-size-lg: 18px;

  /* Border Radius */
  --border-radius-sm: 2px;
  --border-radius-md: 4px;
  --border-radius-lg: 8px;
}

[data-color-scheme="dark"] {
  --color-brand-primary: #8B6FFF;
  --color-brand-secondary: #FF8B6F;
}
```

**2. Create token accessor:**
```tsx
// tokens.ts
export const tokens = {
  color: {
    brand: {
      primary: 'var(--color-brand-primary)',
      secondary: 'var(--color-brand-secondary)',
    },
    semantic: {
      success: 'var(--color-semantic-success)',
      error: 'var(--color-semantic-error)',
    },
  },
  space: {
    xs: 'var(--space-xs)',
    sm: 'var(--space-sm)',
    md: 'var(--space-md)',
    lg: 'var(--space-lg)',
  },
};
```

**3. Use in styled components:**
```tsx
import styled from 'styled-components';
import { tokens } from './tokens';

const BrandButton = styled.button`
  background: ${tokens.color.brand.primary};
  padding: ${tokens.space.md};
  border-radius: ${tokens.borderRadius.md};
  font-size: ${tokens.typography.fontSize.md};

  &:hover {
    background: ${tokens.color.brand.secondary};
  }
`;
```

## Token Categories

### Color Tokens

```tsx
const colorTokens = {
  // Brand colors
  brand: {
    primary: '#6C47FF',
    secondary: '#FF6C47',
    tertiary: '#47D7FF',
  },

  // Semantic colors
  semantic: {
    success: {
      default: '#00C9A7',
      hover: '#00B396',
      active: '#009D85',
    },
    error: {
      default: '#FF5C51',
      hover: '#E64C42',
      active: '#CC3D33',
    },
    warning: {
      default: '#FFC845',
      hover: '#E6B33E',
      active: '#CC9F37',
    },
    info: {
      default: '#5393FF',
      hover: '#4A84E6',
      active: '#4176CC',
    },
  },

  // Neutral colors
  neutral: {
    0: '#FFFFFF',
    100: '#F8F9FA',
    200: '#E9ECEF',
    300: '#DEE2E6',
    400: '#CED4DA',
    500: '#ADB5BD',
    600: '#6C757D',
    700: '#495057',
    800: '#343A40',
    900: '#212529',
  },
};

// Map to Garden theme
const theme = {
  ...DEFAULT_THEME,
  customTokens: { colors: colorTokens },
};
```

### Spacing Tokens

```tsx
const spacingTokens = {
  // Base spacing scale (4px base)
  0: '0',
  1: '4px',
  2: '8px',
  3: '12px',
  4: '16px',
  5: '20px',
  6: '24px',
  8: '32px',
  10: '40px',
  12: '48px',
  16: '64px',
  20: '80px',
  24: '96px',

  // Semantic spacing
  xs: '4px',
  sm: '8px',
  md: '16px',
  lg: '24px',
  xl: '32px',
  xxl: '48px',

  // Component-specific
  buttonPadding: '12px 16px',
  inputPadding: '8px 12px',
  cardPadding: '24px',
};
```

### Typography Tokens

```tsx
const typographyTokens = {
  fontFamily: {
    primary: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
    heading: "'Inter', sans-serif",
    mono: "'Fira Code', 'Consolas', 'Monaco', monospace",
  },

  fontSize: {
    xs: '12px',
    sm: '14px',
    md: '16px',
    lg: '18px',
    xl: '20px',
    '2xl': '24px',
    '3xl': '30px',
    '4xl': '36px',
  },

  fontWeight: {
    light: 300,
    regular: 400,
    medium: 500,
    semibold: 600,
    bold: 700,
  },

  lineHeight: {
    tight: 1.25,
    normal: 1.5,
    relaxed: 1.75,
    loose: 2,
  },

  letterSpacing: {
    tighter: '-0.05em',
    tight: '-0.025em',
    normal: '0',
    wide: '0.025em',
    wider: '0.05em',
  },
};
```

### Border & Shadow Tokens

```tsx
const borderTokens = {
  width: {
    none: '0',
    thin: '1px',
    medium: '2px',
    thick: '4px',
  },

  radius: {
    none: '0',
    sm: '2px',
    md: '4px',
    lg: '8px',
    xl: '12px',
    '2xl': '16px',
    full: '9999px',
  },

  style: {
    solid: 'solid',
    dashed: 'dashed',
    dotted: 'dotted',
  },
};

const shadowTokens = {
  sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
  md: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
  lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1)',
  xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1)',
  '2xl': '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
  inner: 'inset 0 2px 4px 0 rgba(0, 0, 0, 0.06)',
};
```

## Using Custom Tokens in Components

### With styled-components

```tsx
import styled from 'styled-components';
import { useTheme } from '@zendeskgarden/react-theming';

const StyledCard = styled.div`
  background: white;
  border-radius: ${props => props.theme.customTokens.borderRadius.lg};
  padding: ${props => props.theme.customTokens.spacing.lg};
  box-shadow: ${props => props.theme.customTokens.shadows.md};
  border: ${props => props.theme.customTokens.border.width.thin} solid
          ${props => props.theme.customTokens.colors.neutral[200]};
`;

const StyledTitle = styled.h2`
  font-family: ${props => props.theme.customTokens.typography.fontFamily.heading};
  font-size: ${props => props.theme.customTokens.typography.fontSize['2xl']};
  font-weight: ${props => props.theme.customTokens.typography.fontWeight.semibold};
  color: ${props => props.theme.customTokens.colors.neutral[900]};
  margin-bottom: ${props => props.theme.customTokens.spacing.md};
`;

export const Card = ({ title, children }) => {
  return (
    <StyledCard>
      <StyledTitle>{title}</StyledTitle>
      {children}
    </StyledCard>
  );
};
```

### With useTheme Hook

```tsx
import { useTheme } from '@zendeskgarden/react-theming';

export const TokenDisplay = () => {
  const theme = useTheme();
  const tokens = theme.customTokens;

  return (
    <div
      style={{
        padding: tokens.spacing.lg,
        backgroundColor: tokens.colors.brand.primary,
        borderRadius: tokens.borderRadius.md,
        color: 'white',
      }}
    >
      <h1 style={{ fontSize: tokens.typography.fontSize.xl }}>
        Using Custom Tokens
      </h1>
    </div>
  );
};
```

## Token Organization Best Practices

### 1. Consistent Naming Convention

```tsx
// Good - clear hierarchy
tokens.color.brand.primary
tokens.spacing.component.button.padding
tokens.typography.fontSize.heading.large

// Bad - inconsistent
tokens.brandColorPrimary
tokens.btnPadding
tokens.h1Size
```

### 2. Semantic vs. Literal Tokens

```tsx
// Literal tokens (base values)
const baseTokens = {
  color: {
    purple: {
      500: '#6C47FF',
      600: '#5430CC',
    },
  },
};

// Semantic tokens (mapped meanings)
const semanticTokens = {
  color: {
    brand: {
      primary: baseTokens.color.purple[500],
      primaryHover: baseTokens.color.purple[600],
    },
  },
};
```

### 3. Responsive Tokens

```tsx
const responsiveTokens = {
  spacing: {
    container: {
      mobile: '16px',
      tablet: '24px',
      desktop: '32px',
    },
  },
  typography: {
    fontSize: {
      heading: {
        mobile: '24px',
        tablet: '32px',
        desktop: '40px',
      },
    },
  },
};

// Usage in styled component
const ResponsiveHeading = styled.h1`
  font-size: ${props => props.theme.customTokens.typography.fontSize.heading.mobile};

  @media (min-width: 768px) {
    font-size: ${props => props.theme.customTokens.typography.fontSize.heading.tablet};
  }

  @media (min-width: 1024px) {
    font-size: ${props => props.theme.customTokens.typography.fontSize.heading.desktop};
  }
`;
```

## Dark Mode with Custom Tokens

```tsx
import { ColorSchemeProvider } from '@zendeskgarden/react-theming';

const tokens = {
  light: {
    background: {
      default: '#FFFFFF',
      subtle: '#F8F9FA',
      emphasis: '#E9ECEF',
    },
    foreground: {
      default: '#212529',
      subtle: '#6C757D',
      emphasis: '#000000',
    },
  },
  dark: {
    background: {
      default: '#1A1D1F',
      subtle: '#272B30',
      emphasis: '#33383D',
    },
    foreground: {
      default: '#F8F9FA',
      subtle: '#ADB5BD',
      emphasis: '#FFFFFF',
    },
  },
};

// Create theme with color scheme support
const createTheme = (colorScheme: 'light' | 'dark') => ({
  ...DEFAULT_THEME,
  customTokens: {
    colors: tokens[colorScheme],
  },
});

function App() {
  const [colorScheme, setColorScheme] = useState<'light' | 'dark'>('light');

  return (
    <ColorSchemeProvider colorScheme={colorScheme}>
      <ThemeProvider theme={createTheme(colorScheme)}>
        <YourApp />
      </ThemeProvider>
    </ColorSchemeProvider>
  );
}
```

## TypeScript Support

### Token Type Definitions

```tsx
// tokens.types.ts
export interface DesignTokens {
  colors: {
    brand: {
      primary: string;
      secondary: string;
      tertiary: string;
    };
    semantic: {
      success: string;
      error: string;
      warning: string;
      info: string;
    };
  };
  spacing: {
    xs: string;
    sm: string;
    md: string;
    lg: string;
    xl: string;
  };
  typography: {
    fontFamily: {
      primary: string;
      heading: string;
      mono: string;
    };
    fontSize: {
      xs: string;
      sm: string;
      md: string;
      lg: string;
      xl: string;
    };
  };
}

// Extend Garden's theme type
declare module 'styled-components' {
  export interface DefaultTheme {
    customTokens: DesignTokens;
  }
}
```

### Usage with Type Safety

```tsx
import styled from 'styled-components';
import { DefaultTheme } from 'styled-components';

const StyledButton = styled.button<{ variant: 'primary' | 'secondary' }>`
  background: ${({ theme, variant }) =>
    variant === 'primary'
      ? theme.customTokens.colors.brand.primary
      : theme.customTokens.colors.brand.secondary
  };
  padding: ${({ theme }) => theme.customTokens.spacing.md};
  font-size: ${({ theme }) => theme.customTokens.typography.fontSize.md};
`;
```

## Token Documentation

Generate documentation from your tokens:

```tsx
// TokenDocumentation.tsx
import { useTheme } from '@zendeskgarden/react-theming';

export const TokenDocumentation = () => {
  const theme = useTheme();
  const { customTokens } = theme;

  return (
    <div>
      <h1>Design Tokens</h1>

      <section>
        <h2>Colors</h2>
        <div>
          {Object.entries(customTokens.colors.brand).map(([name, value]) => (
            <div key={name} style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <div
                style={{
                  width: '48px',
                  height: '48px',
                  background: value,
                  borderRadius: '4px',
                }}
              />
              <div>
                <strong>{name}</strong>
                <br />
                <code>{value}</code>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section>
        <h2>Spacing</h2>
        <div>
          {Object.entries(customTokens.spacing).map(([name, value]) => (
            <div key={name}>
              <strong>{name}</strong>: <code>{value}</code>
              <div style={{ width: value, height: '16px', background: '#ddd' }} />
            </div>
          ))}
        </div>
      </section>
    </div>
  );
};
```

## Resources

- [Style Dictionary](https://amzn.github.io/style-dictionary/)
- [Design Tokens W3C Community Group](https://www.w3.org/community/design-tokens/)
- [Theo (Salesforce)](https://github.com/salesforce-ux/theo)
- [Tokens Studio (Figma Plugin)](https://tokens.studio/)