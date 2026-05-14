---
name: Obsidian Signal
colors:
  surface: '#131313'
  surface-dim: '#131313'
  surface-bright: '#393939'
  surface-container-lowest: '#0e0e0e'
  surface-container-low: '#1c1b1b'
  surface-container: '#201f1f'
  surface-container-high: '#2a2a2a'
  surface-container-highest: '#353534'
  on-surface: '#e5e2e1'
  on-surface-variant: '#ebbbb4'
  inverse-surface: '#e5e2e1'
  inverse-on-surface: '#313030'
  outline: '#b18780'
  outline-variant: '#603e39'
  surface-tint: '#ffb4a8'
  primary: '#ffb4a8'
  on-primary: '#690100'
  primary-container: '#ff5540'
  on-primary-container: '#5c0000'
  inverse-primary: '#c00100'
  secondary: '#c3c6d2'
  on-secondary: '#2c3039'
  secondary-container: '#454953'
  on-secondary-container: '#b5b8c4'
  tertiary: '#acc7ff'
  on-tertiary: '#002f67'
  tertiary-container: '#488fff'
  on-tertiary-container: '#00285b'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#ffdad4'
  primary-fixed-dim: '#ffb4a8'
  on-primary-fixed: '#410000'
  on-primary-fixed-variant: '#930100'
  secondary-fixed: '#dfe2ee'
  secondary-fixed-dim: '#c3c6d2'
  on-secondary-fixed: '#181c24'
  on-secondary-fixed-variant: '#434750'
  tertiary-fixed: '#d7e2ff'
  tertiary-fixed-dim: '#acc7ff'
  on-tertiary-fixed: '#001a40'
  on-tertiary-fixed-variant: '#004491'
  background: '#131313'
  on-background: '#e5e2e1'
  surface-variant: '#353534'
typography:
  display:
    fontFamily: Geist
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Geist
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
    letterSpacing: -0.01em
  body-lg:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
    letterSpacing: 0em
  body-md:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
    letterSpacing: 0em
  label-sm:
    fontFamily: JetBrains Mono
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.02em
  label-xs:
    fontFamily: JetBrains Mono
    fontSize: 10px
    fontWeight: '500'
    lineHeight: 12px
    letterSpacing: 0.04em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  unit: 4px
  container-padding: 1rem
  gutter: 0.5rem
  component-gap: 0.25rem
  stack-spacing: 0.75rem
---

## Brand & Style

This design system is engineered for professional creators and community managers who require high-velocity interaction with large volumes of data. The brand personality is **utilitarian, precise, and authoritative**, moving away from the entertainment-centric nature of the source platform toward a "mission control" aesthetic.

The visual style is **High-Density Minimalism**. It prioritizes information throughput over decorative breathing room. By utilizing a dark, low-fatigue palette and a structured, "modular" layout, the system minimizes cognitive load while maximizing the number of actionable items on screen. The interface should feel like a high-performance terminal—fast, responsive, and strictly functional.

## Colors

The color strategy uses a **neutral-first approach** to ensure the user's focus remains on the content (the comments). 
- **Primary Background:** A deep, near-black charcoal (`#0A0A0A`) serves as the canvas to reduce eye strain during long sessions.
- **Surface Tiers:** We use subtle shifts in gray to define hierarchy. Secondary surfaces (`#141414`) and tertiary containers (`#1F1F1F`) separate content modules without the need for heavy borders.
- **Accent:** YouTube Red (`#FF0000`) is reserved exclusively for high-priority states: destructive actions (Delete/Ban), critical alerts, or "Live" status indicators. It should never be used for secondary buttons or decorative elements.
- **Text:** High-contrast off-white for primary labels, with stepped-down grays for timestamps and metadata.

## Typography

The typography system is built for **scannability and density**.
- **Geist** is used for headers to provide a modern, technical feel with its geometric precision.
- **Inter** handles the bulk of comment text for maximum readability at smaller sizes. Tracking is tightened slightly across the board to conserve horizontal space.
- **JetBrains Mono** is utilized for metadata, tags, and timestamps. Its monospaced nature helps users quickly align information when scanning down columns of data (like view counts or timestamps).
- **Hierarchy:** Use font weight rather than size to distinguish between user handles and comment body text to maintain a consistent baseline across the dense UI.

## Layout & Spacing

This design system employs a **compact fluid grid** designed to fill wide-screen monitors typical of developer setups. 
- **Density:** We use a 4px base unit. Component internal padding should rarely exceed 8px (2 units) to ensure more data is visible above the fold.
- **Grid:** A 12-column layout is standard, but in the "Comment Management" view, a 3-pane layout is preferred: 
    - *Left Sidebar (narrow):* Navigation and filters.
    - *Center (wide):* Scrollable comment feed.
    - *Right Sidebar (medium):* User details and history.
- **Breakpoints:** 
    - Desktop (1280px+): Full 3-pane view.
    - Tablet (768px - 1279px): Collapsed sidebars, focus on feed.
    - Mobile: Single column, bottom-sheet actions for comment management.

## Elevation & Depth

To maintain a "flat/utility" feel, the system avoids traditional drop shadows. Depth is communicated through:
- **Tonal Layering:** The further "forward" an element is, the lighter its gray value. A modal sits on `base-300`, while the background is `base-100`.
- **Low-Contrast Outlines:** Surfaces are defined by 1px solid borders (`#2A2E37`). This creates a sharp, technical look that avoids the "fuzziness" of shadows.
- **Focus States:** Active elements (like a selected comment) should use a subtle 1px primary border or a distinct background shift rather than a shadow glow.

## Shapes

The shape language is **precise and rectangular**. 
- **Base Radius:** 4px is the standard for cards, buttons, and input fields. This provides just enough softness to be professional without appearing playful.
- **Full Rounding:** Only used for user avatars and specific status "pills" (like 'New' or 'Moderated') to provide a visual contrast against the otherwise rigid grid.

## Components

The components follow the **DaisyUI/Tailwind** convention but are customized for high density:

- **Buttons:** Use the `btn-sm` or `btn-xs` scale by default. The `primary` button (YouTube Red) is reserved for 'Submit', 'Broadcast', or 'Delete'. Secondary actions use `ghost` or `outline` styles.
- **Input Fields:** Minimalist boxes with a subtle border. On focus, the border shifts to a light gray, never the primary accent unless there is an error.
- **Cards (Comment Rows):** These are the core atoms. They should have minimal vertical padding. Actions (Like, Delete, Reply) should appear on hover or be tucked into a compact menu to reduce visual noise in the default view.
- **Chips/Badges:** Small, monospaced labels for 'Spam Score', 'Subscriber Status', or 'Member Level'. Use subtle background tints (e.g., a dark green tint for 'Member').
- **Status Indicators:** Use small, 6px circles for status. Green for 'Approved', Red for 'Flagged', and Yellow for 'Pending'.
- **Lists:** Comment threads should use a vertical line "thread" indicator (1px wide, subtle gray) to show nesting without excessive indentation.