# PDF Splitting and Google NotebookLM Upload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split six PDF files on the desktop into chapter slices and upload them to separate notebooks in Google NotebookLM using the `nlm` CLI.

**Architecture:** Use a modular Python script structure. One script handles TOC/metadata extraction (both dynamic and static), another script interfaces with the `nlm` CLI using the `subprocess` module, and a main orchestration script runs the pipeline sequentially, creating temp slices, creating/finding notebooks, uploading files, and cleaning up.

**Tech Stack:** Python 3.14, PyMuPDF (fitz), Google NotebookLM CLI (nlm).

---

## File Structure
- `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\extract_chapters.py`: Handles getting chapter titles and page ranges.
- `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\upload_to_notebooklm.py`: Manages the `nlm` CLI subprocess interface.
- `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\split_and_upload.py`: The orchestrator script.

---

### Task 1: Create Temporary Storage Directory

**Files:**
- Create: `D:\download\project\bluebell\split_chapters`

- [ ] **Step 1: Create the directory**
  Create the folder `D:\download\project\bluebell\split_chapters` using a terminal command.

  Run: `mkdir D:\download\project\bluebell\split_chapters`
  Expected: Directory is created.

---

### Task 2: Create Chapter Extraction Module

**Files:**
- Create: `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\extract_chapters.py`
- Test: `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\test_extract_chapters.py`

- [ ] **Step 1: Write the extraction module**
  Create `extract_chapters.py` with dynamic parsing logic for Books 1, 3, and 6, and static ranges for Books 2, 4, and 5.

```python
# C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\extract_chapters.py
import fitz
import re

# Static mappings for books with broken outlines
STATIC_BOOK_MAPPINGS = {
    "DDD": {
        "path": r"C:\Users\pc\Desktop\Domain-Driven Design Tips and Tricks to Learn the Best Theories and Principles of Domain-Driven Design (2) (Jim LEWIS) (z-library.sk, 1lib.sk, z-lib.sk).pdf",
        "chapters": [
            ("Chapter 1: Introduction to Domain-driven Design (DDD)", 13, 23),
            ("Chapter 2: The Business Value of DDD", 24, 33),
            ("Chapter 3: Importance of Strategic Thinking", 34, 40),
            ("Chapter 4: Challenges of DDD", 41, 54),
            ("Chapter 5: Domains, Subdomains, and Bounded Contexts", 55, 62),
            ("Chapter 6: Introduction to Context Maps", 63, 70),
            ("Chapter 7: Strategic DDD Using Context Maps", 71, 85),
            ("Chapter 8: Entities in DDD", 86, 101),
            ("Chapter 9: Value Objects", 102, 109),
            ("Chapter 10: Differences Between Entities and Value Objects", 110, 118),
            ("Chapter 11: Services in DDD", 119, 129),
            ("Chapter 12: Domain Events - Design and Implementation", 130, 151),
            ("Chapter 13: Modules in DDD", 152, 159),
            ("Chapter 14: Aggregates in Domains", 160, 166),
            ("Chapter 15: Factories", 167, 178),
            ("Chapter 16: Repository Patterns", 179, 185),
            ("Conclusion", 186, 189),
        ]
    },
    "GoDesign": {
        "path": r"C:\Users\pc\Desktop\Go 语言设计与实现 (左书祺) (z-library.sk, 1lib.sk, z-lib.sk).pdf",
        "chapters": [
            ("第1章: 调试源代码", 12, 15),
            ("第2章: 编译原理", 16, 61),
            ("第3章: 数据结构", 62, 106),
            ("第4章: 语言特性", 107, 146),
            ("第5章: 常用关键字", 147, 195),
            ("第6章: 并发编程", 196, 307),
            ("第7章: 内存管理", 308, 381),
            ("第8章: 元编程", 382, 393),
            ("第9章: 标准库", 394, 422),
        ]
    },
    "K8s": {
        "path": r"C:\Users\pc\Desktop\深入剖析Kubernetes (张磊 (自动化技术)) (z-library.sk, 1lib.sk, z-lib.sk).pdf",
        "chapters": [
            ("第1章: 背景回顾:云原生大事记", 10, 24),
            ("第2章: 容器技术基础", 25, 54),
            ("第3章: Kubernetes设计与架构", 55, 63),
            ("第4章: Kubernetes集群搭建与配置", 64, 89),
            ("第5章: Kubernetes编排原理", 90, 224),
            ("第6章: Kubernetes存储原理", 225, 260),
            ("第7章: Kubernetes网络原理", 261, 319),
            ("第8章: Kubernetes调度与资源管理", 320, 342),
            ("第9章: 容器运行时", 343, 356),
            ("第10章: Kubernetes监控与日志", 357, 373),
            ("第11章: Kubernetes应用管理进阶", 374, 385),
            ("第12章: Kubernetes开源社区", 386, 391),
        ]
    }
}

DYNAMIC_BOOKS = {
    "DMAS": r"C:\Users\pc\Desktop\Designing Multi-Agent Systems (Victor Dibia) (z-library.sk, 1lib.sk, z-lib.sk).pdf",
    "EG": r"C:\Users\pc\Desktop\Efficient Go (Bartlomiej Plotka) (z-library.sk, 1lib.sk, z-lib.sk).pdf",
    "AIA": r"C:\Users\pc\Desktop\AI Agents in Action (Micheal Lanham) (z-library.sk, 1lib.sk, z-lib.sk).pdf"
}

def extract_dmas_chapters(path):
    doc = fitz.open(path)
    toc = doc.get_toc()
    # Extract Level 2 items under parts I to IV
    chapters = []
    level2_entries = []
    for entry in toc:
        level, title, page = entry
        if level == 2:
            # Check if this is preface/glossary or actual chapters. Preface starts at 15.
            if page >= 26 and page <= 377:
                level2_entries.append((title, page))
    
    # Calculate page ranges
    for idx, (title, start_page) in enumerate(level2_entries):
        end_page = doc.page_count
        if idx + 1 < len(level2_entries):
            end_page = level2_entries[idx + 1][1] - 1
        
        # Format title
        full_title = f"Chapter {idx + 1}: {title}"
        chapters.append((full_title, start_page, end_page))
    return chapters

def extract_numbered_chapters(path):
    doc = fitz.open(path)
    toc = doc.get_toc()
    # Extract Level 1 entries starting with digits
    level1_entries = []
    for entry in toc:
        level, title, page = entry
        if level == 1:
            # match pattern "1. Software Efficiency" or "1 Introduction to agents"
            if re.match(r'^\d+', title.strip()):
                level1_entries.append((title.strip(), page))
                
    chapters = []
    for idx, (title, start_page) in enumerate(level1_entries):
        end_page = doc.page_count
        # Find next chapter page or index page
        # Let's find index/appendix start pages
        appendix_page = doc.page_count
        for entry in toc:
            l, t, p = entry
            if l == 1 and p > start_page:
                if re.match(r'^(appendix|index)', t.strip(), re.IGNORECASE) or (idx + 1 < len(level1_entries) and p >= level1_entries[idx+1][1]):
                    appendix_page = min(appendix_page, p)
        
        if idx + 1 < len(level1_entries):
            end_page = level1_entries[idx + 1][1] - 1
        else:
            end_page = appendix_page - 1
            
        # Standardize the format: e.g. "Chapter 1: Software Efficiency Matters"
        # strip any existing digit prefix
        clean_title = re.sub(r'^\d+[\.\s]*', '', title)
        full_title = f"Chapter {idx + 1}: {clean_title}"
        chapters.append((full_title, start_page, end_page))
    return chapters

def get_all_book_chapters():
    results = {}
    
    # Load dynamic books
    results["DMAS"] = {
        "path": DYNAMIC_BOOKS["DMAS"],
        "chapters": extract_dmas_chapters(DYNAMIC_BOOKS["DMAS"])
    }
    results["EG"] = {
        "path": DYNAMIC_BOOKS["EG"],
        "chapters": extract_numbered_chapters(DYNAMIC_BOOKS["EG"])
    }
    results["AIA"] = {
        "path": DYNAMIC_BOOKS["AIA"],
        "chapters": extract_numbered_chapters(DYNAMIC_BOOKS["AIA"])
    }
    
    # Load static books
    for k, v in STATIC_BOOK_MAPPINGS.items():
        results[k] = v
        
    return results
```

- [ ] **Step 2: Write tests for extraction module**
  Create `test_extract_chapters.py` to verify all books have non-empty chapters and sensible page numbers.

```python
# C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\test_extract_chapters.py
from extract_chapters import get_all_book_chapters

def test_extraction():
    all_books = get_all_book_chapters()
    for prefix, data in all_books.items():
        print(f"Book: {prefix}, Path: {data['path']}")
        chapters = data["chapters"]
        assert len(chapters) > 0, f"No chapters found for {prefix}"
        print(f"  Total chapters: {len(chapters)}")
        for title, start, end in chapters:
            assert start <= end, f"Invalid range in {prefix}: {title} ({start}-{end})"
            assert start > 0, f"Invalid start page in {prefix}: {title} ({start})"
        print(f"  First: {chapters[0]}")
        print(f"  Last: {chapters[-1]}")

if __name__ == '__main__':
    test_extraction()
    print("ALL EXTRACTION TESTS PASSED!")
```

- [ ] **Step 3: Run the test to verify correctness**
  Run: `python "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\test_extract_chapters.py"`
  Expected: Outputs summary details for all 6 books and prints "ALL EXTRACTION TESTS PASSED!".

- [ ] **Step 4: Commit**
  Run:
  ```powershell
  git add "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\extract_chapters.py"; git commit -m "feat: add extract_chapters module"
  ```

---

### Task 3: Create NotebookLM CLI Subprocess Interface Module

**Files:**
- Create: `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\upload_to_notebooklm.py`

- [ ] **Step 1: Write the upload interface module**
  Create `upload_to_notebooklm.py` to wrap the `nlm` commands using Python's `subprocess`.

```python
# C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\upload_to_notebooklm.py
import subprocess
import json
import os
import time

def list_notebooks():
    """List all notebooks in NotebookLM."""
    try:
        # Run nlm list notebooks --json
        res = subprocess.run(["nlm", "list", "notebooks", "--json"], capture_output=True, text=True, check=True)
        return json.loads(res.stdout)
    except Exception as e:
        print(f"Error listing notebooks: {e}")
        return []

def create_notebook(title):
    """Create a new notebook with title and return its ID."""
    try:
        print(f"Creating notebook: {title}")
        res = subprocess.run(["nlm", "notebook", "create", title, "--json"], capture_output=True, text=True, check=True)
        data = json.loads(res.stdout)
        return data.get("notebook_id")
    except Exception as e:
        print(f"Error creating notebook {title}: {e}")
        return None

def upload_source(notebook_id, file_path):
    """Upload a source PDF to a notebook and wait for completion."""
    try:
        print(f"Uploading source {file_path} to notebook {notebook_id}...")
        # nlm source add <notebook_id> --file <file_path> --wait
        res = subprocess.run(["nlm", "source", "add", notebook_id, "--file", file_path, "--wait"], capture_output=True, text=True, check=True)
        print("Upload completed successfully.")
        return True
    except Exception as e:
        print(f"Error uploading source: {e}")
        return False
```

- [ ] **Step 2: Commit**
  Run:
  ```powershell
  git add "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\upload_to_notebooklm.py"; git commit -m "feat: add upload_to_notebooklm module"
  ```

---

### Task 4: Create Orchestration Script

**Files:**
- Create: `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\split_and_upload.py`

- [ ] **Step 1: Write the orchestrator script**
  Create `split_and_upload.py` to tie TOC extraction, PDF slicing, and NotebookLM uploading together.

```python
# C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\split_and_upload.py
import os
import fitz
from extract_chapters import get_all_book_chapters
from upload_to_notebooklm import list_notebooks, create_notebook, upload_source

TEMP_DIR = r"D:\download\project\bluebell\split_chapters"

def get_existing_notebooks_map():
    notebooks = list_notebooks()
    mapping = {}
    for nb in notebooks:
        title = nb.get("title")
        nb_id = nb.get("id")
        if title and nb_id:
            mapping[title] = nb_id
    return mapping

def slice_pdf(pdf_path, start_page, end_page, output_path):
    """Slice original PDF from start_page to end_page (1-indexed, inclusive) using fitz."""
    doc = fitz.open(pdf_path)
    # fitz uses 0-indexed pages
    start_idx = start_page - 1
    end_idx = end_page - 1
    
    new_doc = fitz.open()
    new_doc.insert_pdf(doc, from_page=start_idx, to_page=end_idx)
    new_doc.save(output_path)
    new_doc.close()
    doc.close()

def main():
    if not os.path.exists(TEMP_DIR):
        os.makedirs(TEMP_DIR)
        
    print("Fetching existing notebooks from NotebookLM...")
    existing_notebooks = get_existing_notebooks_map()
    print(f"Found {len(existing_notebooks)} existing notebooks.")
    
    print("Loading chapter outlines for all books...")
    all_books = get_all_book_chapters()
    
    for prefix, data in all_books.items():
        pdf_path = data["path"]
        chapters = data["chapters"]
        print(f"\n=========================================")
        print(f"Processing Book: {prefix} ({len(chapters)} chapters)")
        print(f"File: {pdf_path}")
        
        for idx, (ch_title, start_page, end_page) in enumerate(chapters):
            notebook_title = f"{prefix} - {ch_title}"
            print(f"\n---> Chapter {idx+1}/{len(chapters)}: {notebook_title} (Pages {start_page} to {end_page})")
            
            # Step 1: Resolve notebook ID (Create if doesn't exist)
            if notebook_title in existing_notebooks:
                nb_id = existing_notebooks[notebook_title]
                print(f"Notebook '{notebook_title}' already exists (ID: {nb_id}). Skipping creation.")
            else:
                nb_id = create_notebook(notebook_title)
                if not nb_id:
                    print(f"FAILED to create notebook for {notebook_title}. Skipping.")
                    continue
                # Add to local cache map to avoid duplicate attempts
                existing_notebooks[notebook_title] = nb_id
            
            # Step 2: Slice the PDF
            temp_pdf_name = f"{prefix}_Ch_{idx+1}.pdf"
            temp_pdf_path = os.path.join(TEMP_DIR, temp_pdf_name)
            
            try:
                slice_pdf(pdf_path, start_page, end_page, temp_pdf_path)
                print(f"Created slice: {temp_pdf_path} (Size: {os.path.getsize(temp_pdf_path)} bytes)")
                
                # Step 3: Upload the sliced PDF to the notebook
                success = upload_source(nb_id, temp_pdf_path)
                if success:
                    print(f"Successfully processed and uploaded {notebook_title}.")
                else:
                    print(f"Failed to upload source for {notebook_title}.")
            except Exception as e:
                print(f"Error processing chapter: {e}")
            finally:
                # Step 4: Clean up temp file
                if os.path.exists(temp_pdf_path):
                    os.remove(temp_pdf_path)
                    print(f"Cleaned up temp file: {temp_pdf_path}")
                    
    print("\nALL BOOKS PROCESSED!")

if __name__ == '__main__':
    main()
```

- [ ] **Step 2: Commit**
  Run:
  ```powershell
  git add "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\split_and_upload.py"; git commit -m "feat: add orchestrator script split_and_upload"
  ```

---
 
### Task 5: Execute Script and Monitor (First 6 Books)
 
- [x] **Step 1: Execute the script in the background**
  Run: `python "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\split_and_upload.py"`
  Expected: Script runs and uploads the 75 chapter files sequentially.

---

### Task 6: Add MongoDB Book and Execute Second Pass

**Files:**
- Modify: `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\extract_chapters.py`

- [x] **Step 1: Add MongoDB to static book mappings**
  Add the static chapter metadata mapping for the book `C:\Users\pc\Desktop\MongoDB权威指南第3版.pdf` with the `MongoDB` prefix.
 
- [x] **Step 2: Run verification test**
  Run: `python "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\test_extract_chapters.py"`
  Expected: Verification passes and prints MongoDB chapter count (11).
 
- [x] **Step 3: Run the orchestrator script for MongoDB**
  Run: `python "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\split_and_upload.py"`
  Expected: Orchestrator reads active notebooks cache, identifies existing 75 notebooks, skips them, and creates & uploads the 11 new MongoDB notebooks.

---

### Task 7: Add Concurrency in Go, Go Design Patterns, and Hands-On Software Architecture Books and Execute Third Pass

**Files:**
- Modify: `C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\extract_chapters.py`

- [x] **Step 1: Add new books to static mappings**
  Add static mappings for `Concurrency in Go` (CG), `Go Design Patterns` (GDP), and `Hands-On Software Architecture with Golang` (GoArch) to `STATIC_BOOK_MAPPINGS` in `extract_chapters.py`.

- [x] **Step 2: Run verification test**
  Run: `python "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\test_extract_chapters.py"`
  Expected: All 10 books verification passes.

- [ ] **Step 3: Run the orchestrator script for the third pass**
  Run: `python -u "C:\Users\pc\.gemini\antigravity-cli\brain\8f929dd9-8c42-434f-940f-b979ab20fa48\scratch\split_and_upload.py"`
  Expected: Orchestrator runs, skips the 86 existing notebooks, and uploads the 28 new chapters.
