# Multi-Agent / PDF Splitting and Google NotebookLM Upload Design

**Date**: 2026-06-12
**Author**: Antigravity (Advanced Agentic Coding Team)

---

## 1. Overview
The goal of this task is to split six PDF files located at `C:\Users\pc\Desktop\` by chapter and upload each chapter as a separate source into its own Notebook on Google NotebookLM using the `nlm` command-line utility.

### Books to Process
1. `Designing Multi-Agent Systems (Victor Dibia)` (Abbreviation: `DMAS`)
2. `Domain-Driven Design Tips and Tricks (Jim LEWIS)` (Abbreviation: `DDD`)
3. `Efficient Go (Bartlomiej Plotka)` (Abbreviation: `EG`)
4. `Go 语言设计与实现 (左书祺)` (Abbreviation: `GoDesign`)
5. `深入剖析Kubernetes (张磊)` (Abbreviation: `K8s`)
6. `AI Agents in Action (Micheal Lanham)` (Abbreviation: `AIA`)

---

## 2. Notebook Naming & Chapter Range Specifications

The notebook titles will be formatted as `[BookAbbrev] - [Chapter prefix]: [Title]`.

### Book 1: Designing Multi-Agent Systems (DMAS)
* **Strategy**: Extracted dynamically from Level 2 TOC entries under Parts I to IV.
* **Notebook Prefix**: `DMAS - Chapter [N]: [Title]`
* **Expected Chapters (15)**:
  1. `Understanding Multi-Agent Systems` (Pages 26–54)
  2. `Multi-Agent Patterns` (Pages 55–73)
  3. `UX Principles for Multi-Agent Systems` (Pages 74–90)
  4. `Building Your First Agent` (Pages 92–149)
  5. `Building Computer Use Agents` (Pages 150–171)
  6. `Building Multi-Agent Workflows` (Pages 172–195)
  7. `Building Autonomous Multi-Agent Orchestration` (Pages 196–220)
  8. `Building Modern Web Experiences for Agent Applications` (Pages 221–245)
  9. `But What About Multi-Agent Frameworks?` (Pages 246–259)
  10. `Evaluating Multi-Agent Systems` (Pages 261–284)
  11. `Optimizing Multi-Agent Systems` (Pages 285–301)
  12. `Protocols for Distributed Agents` (Pages 302–324)
  13. `Ethics and Responsible AI for Multi-Agent Systems` (Pages 325–343)
  14. `Answering Business Questions from Unstructured Data` (Pages 345–364)
  15. `Building a Software Engineering Agent` (Pages 365–377)

### Book 2: Domain-Driven Design Tips and Tricks (DDD)
* **Strategy**: Static mapping based on search matches.
* **Notebook Prefix**: `DDD - Chapter [N]: [Title]` (Conclusion: `DDD - Conclusion`)
* **Chapters (17)**:
  1. `Chapter One: Introduction to Domain-driven Design (DDD)` (Pages 13–23)
  2. `Chapter Two: The Business Value of DDD` (Pages 24–33)
  3. `Chapter Three: Importance of Strategic Thinking` (Pages 34–40)
  4. `Chapter Four: Challenges of DDD` (Pages 41–54)
  5. `Chapter Five: Domains, Subdomains, and Bounded Contexts` (Pages 55–62)
  6. `Chapter Six: Introduction to Context Maps` (Pages 63–70)
  7. `Chapter Seven: Strategic DDD Using Context Maps` (Pages 71–85)
  8. `Chapter Eight: Entities in DDD` (Pages 86–101)
  9. `Chapter Nine: Value Objects` (Pages 102–109)
  10. `Chapter Ten: Differences Between Entities and Value Objects` (Pages 110–118)
  11. `Chapter Eleven: Services in DDD` (Pages 119–129)
  12. `Chapter Twelve: Domain Events - Design and Implementation` (Pages 130–151)
  13. `Chapter Thirteen: Modules in DDD` (Pages 152–159)
  14. `Chapter Fourteen: Aggregates in Domains` (Pages 160–166)
  15. `Chapter Fifteen: Factories` (Pages 167–178)
  16. `Chapter Sixteen: Repository Patterns` (Pages 179–185)
  17. `Conclusion` (Pages 186–189)

### Book 3: Efficient Go (EG)
* **Strategy**: Extracted dynamically from Level 1 TOC entries starting with digits.
* **Notebook Prefix**: `EG - Chapter [N]: [Title]`
* **Expected Chapters (11)**:
  * Chapter 1 to 11 (Pages starting from 17, 65, 109, 164, 214, 273, 339, 372, 441, 512, 555, ending at page 614).

### Book 4: Go 语言设计与实现 (GoDesign)
* **Strategy**: Static mapping based on search matches.
* **Notebook Prefix**: `GoDesign - 第[N]章: [Title]`
* **Chapters (9)**:
  1. `第1章: 调试源代码` (Pages 12–15)
  2. `第2章: 编译原理` (Pages 16–61)
  3. `第3章: 数据结构` (Pages 62–106)
  4. `第4章: 语言特性` (Pages 107–146)
  5. `第5章: 常用关键字` (Pages 147–195)
  6. `第6章: 并发编程` (Pages 196–307)
  7. `第7章: 内存管理` (Pages 308–381)
  8. `第8章: 元编程` (Pages 382–393)
  9. `第9章: 标准库` (Pages 394–422)

### Book 5: 深入剖析Kubernetes (K8s)
* **Strategy**: Static mapping based on search matches.
* **Notebook Prefix**: `K8s - 第[N]章: [Title]`
* **Chapters (12)**:
  1. `第1章: 背景回顾:云原生大事记` (Pages 10–24)
  2. `第2章: 容器技术基础` (Pages 25–54)
  3. `第3章: Kubernetes设计与架构` (Pages 55–63)
  4. `第4章: Kubernetes集群搭建与配置` (Pages 64–89)
  5. `第5章: Kubernetes编排原理` (Pages 90–224)
  6. `第6章: Kubernetes存储原理` (Pages 225–260)
  7. `第7章: Kubernetes网络原理` (Pages 261–319)
  8. `第8章: Kubernetes调度与资源管理` (Pages 320–342)
  9. `第9章: 容器运行时` (Pages 343–356)
  10. `第10章: Kubernetes监控与日志` (Pages 357–373)
  11. `第11章: Kubernetes应用管理进阶` (Pages 374–385)
  12. `第12章: Kubernetes开源社区` (Pages 386–391)

### Book 6: AI Agents in Action (AIA)
* **Strategy**: Extracted dynamically from Level 1 TOC entries starting with digits.
* **Notebook Prefix**: `AIA - Chapter [N]: [Title]`
* **Expected Chapters (11)**:
  * Chapters 1 to 11 (Pages starting from 25, 38, 63, 92, 122, 153, 184, 204, 236, 268, 296, ending at page 322).

### Book 7: MongoDB权威指南第3版 (MongoDB)
* **Strategy**: Static mapping based on search matches.
* **Notebook Prefix**: `MongoDB - 第[N]章: [Title]`
* **Chapters (11)**:
  1. `第1章: 简介` (Pages 21–24)
  2. `第2章: 入门` (Pages 25–42)
  3. `第3章: 创建、更新及删除文档` (Pages 43–64)
  4. `第4章: 查询` (Pages 65–84)
  5. `第5章: 索引` (Pages 85–98)
  6. `第6章: 聚合` (Pages 99–110)
  7. `第7章: 进阶指南` (Pages 111–126)
  8. `第8章: 管理` (Pages 127–140)
  9. `第9章: 复制` (Pages 141–154)
  10. `第10章: 分片` (Pages 155–164)
  11. `第11章: 应用举例` (Pages 165–182)

### Book 8: Concurrency in Go (CG)
* **Strategy**: Static mapping based on search matches.
* **Notebook Prefix**: `CG - Chapter [N]: [Title]`
* **Chapters (6)**:
  1. `Chapter 1: An Introduction to Concurrency` (Pages 15–36)
  2. `Chapter 2: Modeling Your Code: Communicating Sequential Processes` (Pages 37–50)
  3. `Chapter 3: Go’s Concurrency Building Blocks` (Pages 51–98)
  4. `Chapter 4: Concurrency Patterns in Go` (Pages 99–160)
  5. `Chapter 5: Concurrency at Scale` (Pages 161–210)
  6. `Chapter 6: Goroutines and the Go Runtime` (Pages 211–238)

### Book 9: Go Design Patterns (GDP)
* **Strategy**: Static mapping based on search matches.
* **Notebook Prefix**: `GDP - Chapter [N]: [Title]`
* **Chapters (10)**:
  1. `Chapter 1: Ready... Steady... Go!` (Pages 28–69)
  2. `Chapter 2: Creational Patterns - Singleton, Builder, Factory, Prototype, and Abstract Factory Design Patterns` (Pages 70–106)
  3. `Chapter 3: Structural Patterns - Composite, Adapter, and Bridge Design Patterns` (Pages 107–132)
  4. `Chapter 4: Structural Patterns - Proxy, Facade, Decorator, and Flyweight Design Patterns` (Pages 133–171)
  5. `Chapter 5: Behavioral Patterns - Strategy, Chain of Responsibility, and Command Design Patterns` (Pages 172–202)
  6. `Chapter 6: Behavioral Patterns - Template, Memento, and Interpreter Design Patterns` (Pages 203–236)
  7. `Chapter 7: Behavioral Patterns - Visitor, State, Mediator, and Observer Design Patterns` (Pages 237–269)
  8. `Chapter 8: Introduction to Gos Concurrency` (Pages 270–302)
  9. `Chapter 9: Concurrency Patterns - Barrier, Future, and Pipeline Design Patterns` (Pages 303–333)
  10. `Chapter 10: Concurrency Patterns - Workers Pool and Publish/Subscriber Design Patterns` (Pages 334–361)

### Book 10: Hands-On Software Architecture with Golang (GoArch)
* **Strategy**: Static mapping based on search matches.
* **Notebook Prefix**: `GoArch - Chapter [N]: [Title]`
* **Chapters (12)**:
  1. `Chapter 1: Building Big with Go` (Pages 22–52)
  2. `Chapter 2: Packaging Code` (Pages 53–80)
  3. `Chapter 3: Design Patterns` (Pages 81–117)
  4. `Chapter 4: Scaling Applications` (Pages 118–150)
  5. `Chapter 5: Going Distributed` (Pages 151–198)
  6. `Chapter 6: Messaging` (Pages 199–239)
  7. `Chapter 7: Building APIs` (Pages 240–281)
  8. `Chapter 8: Modeling Data` (Pages 282–337)
  9. `Chapter 9: Anti-Fragile Systems` (Pages 338–383)
  10. `Chapter 10: Case Study – Travel Website` (Pages 384–413)
  11. `Chapter 11: Planning for Deployment` (Pages 414–458)
  12. `Chapter 12: Migrating Applications` (Pages 459–491)

---

## 3. Automation Implementation Details

A Python script `split_and_upload.py` will be created to orchestrate the process.

### Python Script Pipeline
1. **List Existing Notebooks**: 
   Call `nlm list notebooks --json` to build a lookup cache of already-created notebooks.
2. **Determine Chapter Definitions**:
   * Dynamically parse TOC for Books 1, 3, and 6.
   * Load static ranges for Books 2, 4, and 5.
3. **Iterate Chapters**:
   * Formulate target notebook title.
   * If notebook title already exists in the cache, retrieve its `notebook_id`. Otherwise, call `nlm notebook create "[Title]" --json` and parse the new ID.
   * Verify if the source file is already uploaded to avoid duplication.
   * Slice the source PDF range into a temporary PDF in `D:\download\project\bluebell\split_chapters\`.
   * Call `nlm source add <notebook_id> --file <temp_pdf_path> --wait` to upload.
   * Delete the temporary sliced PDF file.
4. **Log Progress**: Write logs to console.
