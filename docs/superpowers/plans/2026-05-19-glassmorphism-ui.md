# System Optimization Implementation Plan - Phase 3: Glassmorphism UI

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Modernize the Bluebell frontend with a Glassmorphism aesthetic, featuring translucent backgrounds, blur effects, and vibrant gradients.

**Architecture:** Tailwind CSS v4 styling with custom design tokens for glass effects. Component-level refactoring of Home, PostCard, and Navbar.

**Tech Stack:** Vue 3, Tailwind CSS, Lucide Icons.

---

### Task 1: Setup Glassmorphism Design Tokens

**Files:**
- Modify: `frontend/src/style.css`
- Modify: `frontend/tailwind.config.js` (if applicable for v4)

- [ ] **Step 1: Define CSS variables for glass effects**

```css
:root {
  --glass-bg: rgba(255, 255, 255, 0.4);
  --glass-border: rgba(255, 255, 255, 0.2);
  --blur-amount: 12px;
  --accent-gradient: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
}

.glass {
  background: var(--glass-bg);
  backdrop-filter: blur(var(--blur-amount));
  -webkit-backdrop-filter: blur(var(--blur-amount));
  border: 1px solid var(--glass-border);
}
```

- [ ] **Step 2: Update global body background**

- [ ] **Step 3: Commit**

```bash
git add frontend/src/style.css
git commit -m "style: add glassmorphism design tokens"
```

---

### Task 2: Refactor Navbar and Layout

**Files:**
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/components/Navbar.vue`

- [ ] **Step 1: Apply glass style to Navbar**
- [ ] **Step 2: Update App.vue background gradients**
- [ ] **Step 3: Commit**

```bash
git add frontend/src/App.vue frontend/src/components/Navbar.vue
git commit -m "style: apply glassmorphism to navbar and layout"
```

---

### Task 3: Modernize PostCard and Home Page

**Files:**
- Modify: `frontend/src/components/PostCard.vue`
- Modify: `frontend/src/pages/Home.vue`

- [ ] **Step 1: Refactor PostCard with glass cards and rounded corners**
- [ ] **Step 2: Update voting icons and score display with accent colors**
- [ ] **Step 3: Update Home.vue layout with glass sidebars**
- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/PostCard.vue frontend/src/pages/Home.vue
git commit -m "style: modernize home page and post cards with glassmorphism"
```
