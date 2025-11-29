# Frontend Architecture

This document describes the modular architecture of the CI Dashboard frontend.

## Directory Structure

```text
frontend/
├── index.html          # Main HTML entry point
├── ARCHITECTURE.md     # This file
├── css/                # Modular CSS stylesheets
│   ├── main.css        # Main CSS that imports all modules
│   ├── variables.css   # CSS custom properties (theme)
│   ├── base.css        # Reset and base styles
│   ├── header.css      # Header and navigation
│   ├── buttons.css     # Button components
│   ├── panels.css      # Panel containers
│   ├── controls.css    # Control bar and stats
│   ├── pipeline.css    # Pipeline visualization
│   ├── events.css      # Event log
│   ├── summary.css     # Run summary
│   ├── composer.css    # Plan composer
│   ├── plans.css       # Saved plans
│   └── modal.css       # Modal dialogs
├── html/               # HTML component templates (reference)
│   ├── header.html     # Header component template
│   ├── composer.html   # Plan composer tab template
│   ├── plans.html      # Saved plans section template
│   ├── controls.html   # Controls panel template
│   ├── pipeline.html   # Pipeline graph template
│   ├── summary.html    # Run summary template
│   ├── modal.html      # Save plan modal template
│   └── footer.html     # Footer component template
└── js/                 # Modular JavaScript modules
    ├── main.js         # Application entry point
    ├── api.js          # API communication layer
    ├── utils.js        # Utility functions
    ├── dashboard.js    # CIDashboard class
    ├── composer.js     # PlanComposer class
    └── runner.js       # PlanRunner class
```

## JavaScript Modules

### `main.js`

- Application entry point
- Initializes all components
- Sets up global event handlers
- Exposes necessary functions to window for HTML onclick handlers

### `api.js`

- Centralized API communication layer
- All HTTP requests to the backend
- Exports functions for each API endpoint

### `utils.js`

- Shared utility functions
- `escapeHtml()` - XSS prevention
- `formatDuration()` - Time formatting
- `getStatusIcon()` - Status SVG icons

### `dashboard.js`

- `CIDashboard` class
- SSE connection management
- Event handling (started, completed, skipped, summary)
- Pipeline and event log rendering

### `composer.js`

- `PlanComposer` class
- Target selection and dependency auto-selection
- Plan preview
- Save/run custom plans

### `runner.js`

- `PlanRunner` class
- Saved plans management
- Run mode selection (all, target, stage)
- Fail target simulation

## CSS Modules

### `variables.css`

CSS custom properties for theming:

- Colors (background, text, accents, status)
- Shadows, radii, transitions
- Aliases for compatibility

### `base.css`

- CSS reset
- Base typography
- App layout

### Component CSS

Each component has its own stylesheet for isolated styling:

- `header.css` - Header, logo, tabs, connection status
- `buttons.css` - All button variants
- `panels.css` - Panel containers, badges, empty states
- `controls.css` - Control bar, stats display
- `pipeline.css` - Stage visualization, target cards
- `events.css` - Event list items
- `summary.css` - Run summary panel
- `composer.css` - Plan composer UI
- `plans.css` - Saved plans cards and runner
- `modal.css` - Modal dialogs and forms

## HTML Component Templates

The `html/` directory contains reference templates for each major UI component. These serve as:

1. **Documentation** - Clear, isolated view of each component's structure
2. **Development Aid** - Easy to copy and modify individual components
3. **Reference** - Shows the expected HTML structure for styling

**Note:** The actual application uses `index.html` as a single entry point. The templates are for reference and are marked with clear comments indicating their purpose and location in the main file.

### Component Files

| Template | Description |
|----------|-------------|
| `header.html` | Logo, navigation tabs, connection status |
| `composer.html` | Target selection grid, preview, run config |
| `plans.html` | Saved plans list and details panel |
| `controls.html` | Run/cancel buttons, real-time stats |
| `pipeline.html` | Pipeline graph and event log panels |
| `summary.html` | Run summary with results |
| `modal.html` | Save plan dialog |
| `footer.html` | Application footer |

## Usage

The `index.html` loads:

1. `/frontend/css/main.css` - Which imports all CSS modules
2. `/frontend/js/main.js` - ES module entry point

The JavaScript uses ES modules (`import`/`export`) for clean dependency management.

The HTML contains clear section comments referencing the corresponding template files for easier navigation.
