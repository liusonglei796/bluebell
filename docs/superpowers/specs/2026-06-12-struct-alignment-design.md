# Struct Alignment Design Spec

This specification describes the plan for checking all structs in the `bluebell` project, optimizing their memory alignment, and writing an educational tutorial to help developers understand, detect, and fix unaligned structures.

## Objectives

1. **Optimize Memory Usage**: Reorder struct fields in the codebase to minimize padding, reducing memory consumption and improving cache efficiency.
2. **Ensure Code Quality & Correctness**: Verify that changes do not break any existing code (such as positional struct literals) or tests.
3. **Preserve Code Readability**: Clean up formatting and comments in modified files.
4. **Create Educational Content**: Write a markdown tutorial at `docs/struct_alignment_tutorial.md` explaining Go memory alignment.

## Current Status (Unaligned Structs)

Based on the `fieldalignment` tool check, several structs across multiple packages have suboptimal memory layout. Examples include:
- `appConfig`, `mysqlConfig`, `redisConfig`, `Config` in `internal/config/config.go`
- Domain entities: `Bookmark`, `Community`, `Post`, `Remark`, `SearchPostDoc`, `Social`, `User`
- Database models: `Bookmark`, `Post`, `Remark`, `Social`, `User`
- API DTOs and other internal utilities.

## Strategy

1. **Automated Rearrangement**:
   Use `fieldalignment -fix ./...` to automatically sort fields in all Go files to minimize sizes.
2. **Manual Formatting & Adjustments**:
   - Re-format the modified structs to match the project's formatting standard (via `go fmt`).
   - Fix comment positions and alignments that the automated tool might have shifted.
3. **Verification**:
   - Compile the code to verify that no positional struct initializers were broken.
   - Run the test suite (`go test ./...`) to ensure all behaviors remain correct.
4. **Tutorial Writing**:
   Write `docs/struct_alignment_tutorial.md` with:
   - Theory of CPU word size, alignment boundaries, and padding.
   - Diagrammatic explanation of memory layouts.
   - Walkthrough of how to check alignment manually using `unsafe` and tools (`fieldalignment`).
