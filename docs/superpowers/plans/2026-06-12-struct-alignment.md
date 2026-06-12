# Struct Alignment Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rearrange and optimize the memory alignment of all structs identified by `fieldalignment` in the project, verify correctness, and create a tutorial teaching how to check for struct alignment.

**Architecture:** We will use the official Go `fieldalignment` tool with `-fix` to automatically reorder struct fields. Then, we will verify compiling/tests, adjust any comment alignment manually, and write a detailed tutorial at `docs/struct_alignment_tutorial.md`.

**Tech Stack:** Go 1.26, golang.org/x/tools/go/analysis/passes/fieldalignment

---

### Task 1: Automated Struct Realignment

**Files:**
- Modify: Multiple Go files across packages as reported by `fieldalignment`.

- [ ] **Step 1: Run fieldalignment -fix on the workspace**
  Run: `go run golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest -fix ./...`
  Expected: The command runs, applies structural fixes to various files, and exits.

- [ ] **Step 2: Format the code**
  Run: `go fmt ./...`
  Expected: Re-formats files modified by the tool.

- [ ] **Step 3: Commit initial automated changes**
  Run:
  ```bash
  git add -u
  git commit -m "refactor: auto align Go structures using fieldalignment"
  ```

---

### Task 2: Compilation & Test Verification

**Files:**
- Test: All package tests in the workspace.

- [ ] **Step 1: Verify compilation**
  Run: `go build ./...`
  Expected: Clean compile with no errors (e.g. from broken positional struct literals).

- [ ] **Step 2: Run test suite**
  Run: `go test ./...`
  Expected: PASS

- [ ] **Step 3: If compilation or tests fail due to positional literals**
  Review the compiler error output, identify the broken literal, and rewrite it to use keyed fields:
  ```go
  // Example broken literal:
  u := User{"alice", 20}
  // Change to:
  u := User{Name: "alice", Age: 20}
  ```
  Run: `go test ./...` to verify passing.
  Commit any manual fixes:
  ```bash
  git add -u
  git commit -m "fix: fix positional struct literals after field reordering"
  ```

---

### Task 3: Manual Formatting & Comment Polish

**Files:**
- Modify: `internal/config/config.go`, `internal/domain/entity/post.go` (and other files as needed to restore comment style and alignment).

- [ ] **Step 1: Review git diff of modified files**
  Run: `git diff HEAD~1` (or check IDE git status) to review the structural changes.
  Ensure that any block comments or inline comments on struct fields are still aligned and correctly associated with their fields.

- [ ] **Step 2: Manually adjust formatting if needed**
  Ensure struct tag alignment and formatting look clean and professional.
  Run: `go fmt ./...`
  Expected: Clean code.

- [ ] **Step 3: Commit manual formatting polish**
  Run:
  ```bash
  git add -u
  git commit -m "style: polish struct comments and formatting after alignment"
  ```

---

### Task 4: Write the Educational Tutorial

**Files:**
- Create: `docs/struct_alignment_tutorial.md`

- [ ] **Step 1: Write struct_alignment_tutorial.md**
  Write a comprehensive document explaining:
  - What memory alignment is.
  - What struct alignment is and why it matters in Go (CPU word boundaries, padding, memory footprint).
  - Concrete examples comparing unaligned and aligned struct layouts.
  - Rules of thumb for manually ordering fields (largest fields to smallest fields).
  - Special considerations (e.g., zero-sized fields like `struct{}` at the end of structs).
  - How to programmatically detect and measure struct size and alignment using `unsafe.Sizeof`, `unsafe.Alignof`, and `unsafe.Offsetof`.
  - How to use the `fieldalignment` tool for automated checking and fixing.

- [ ] **Step 2: Review tutorial file**
  Open the file `docs/struct_alignment_tutorial.md` and read it to ensure it is clear, concise, and technically accurate.

- [ ] **Step 3: Commit the tutorial**
  Run:
  ```bash
  git add docs/struct_alignment_tutorial.md
  git commit -m "docs: add Go struct alignment tutorial"
  ```
