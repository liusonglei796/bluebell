# Frontend UI Refactoring: Sleek Glassmorphism Specification

This document outlines the design decisions, component layouts, and implementation details for refactoring the Bluebell front-end interface.

## 1. Core Visual Vibe & Aesthetics
We are adopting a **Pure Monochrome Light Mode** glassmorphism aesthetic. It focuses on depth, high contrast black-on-white details, frosted overlays, and smooth transitions.

### 1.1 Typography (Outfit & Inter)
We will import and pair two Google Fonts:
- **Outfit:** For headings, buttons, and big numbers (giving a bold, geometric, tech feel).
- **Inter:** For readable body texts, metadata, labels, and form fields.

### 1.2 Design Tokens
In `frontend/src/style.css`, we will define the following CSS custom properties:
```css
:root {
  --glass-bg: rgba(255, 255, 255, 0.45);
  --glass-border: rgba(255, 255, 255, 0.25);
  --glass-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.05);
  --glass-shadow-hover: 0 12px 40px 0 rgba(31, 38, 135, 0.1);
  --accent: #000000;
  --transition-speed: 0.3s;
  --transition-bounce: cubic-bezier(0.34, 1.56, 0.64, 1);
}
```

To prevent the initial-hover "white flash" (flickering due to lazy GPU promotion on transition), we will build hardware-acceleration directly into the `.glass` class:
```css
.glass {
  background: var(--glass-bg);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid var(--glass-border);
  box-shadow: var(--glass-shadow);
  
  /* Force GPU rendering from the start to prevent white flashes */
  will-change: transform, backdrop-filter;
  transform: translateZ(0);
  backface-visibility: hidden;
}
```

---

## 2. Component Specifications

### 2.1 Global App Shell (`App.vue` & `style.css`)
- **Background:** Soft radial/linear light gray background gradient (`linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%)`).
- **Main Container:** Add appropriate padding and alignment for layout elements.

### 2.2 Navigation (`Navbar.vue`)
- **Structure:** A floating pill navigation bar at the top with a high-contrast logo `BLUEBELL` (Outfit black font).
- **Features:** 
  - Rounded search input with transition expanding on focus.
  - Action buttons styled with solid black background and hover/active states.
  - Hover links with simple, clean underlines.

### 2.3 Post Feed Card (`PostCard.vue`)
- **Layout:** Flex row with voting controls on the left, post content on the right.
- **Styling:**
  - Entire card uses `.glass` class.
  - Elevates on hover using `transform: translateY(-4px);` and a richer shadow.
  - The voting sidebar has a subtle dark transparent background (`rgba(0, 0, 0, 0.03)`), matching custom arrow icon sizes.
  - The header uses Outfit font, and the body text uses Inter font.
  - Community tags are styled as pill badges (`bg-black/5 text-black font-bold`).

### 2.4 User Profile (`Profile.vue`)
- **Left Column (User Card):**
  - Large black-to-gray gradient avatar box with custom user icon.
  - Outfit typography for user's reputation and post stats (split into a grid of 4 clean sub-cards).
- **Right Column (Activity Feed):**
  - Activities are grouped into glass list blocks.
  - Contains descriptive metadata and a blockquote styling for user comments/text.

### 2.5 Post Details & Comments (`PostDetail.vue`)
- **Layout:** Expanded layout for reading post content.
- **Comment Section:**
  - Clean textarea with subtle borders and shadow on focus.
  - The comments list utilizes glass container items with custom timestamps.

### 2.6 Form Pages (Login, Signup, CreatePost, CreateCommunity)
- **Inputs:** Re-styled with frosted inputs, clear focus states, and bold Outfit buttons.

---

## 3. Verification Plan
1. **Visual Check:** Verify the fonts (Outfit, Inter) load correctly.
2. **Animation check:** Verify hover effects on `PostCard` and sidebar communities are smooth and have no white flickering/flashing.
3. **Responsive check:** Ensure layouts remain responsive on mobile screens.
