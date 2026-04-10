# Requirement Model Description PPT - Canonical Plan

## P0: Interview & Normalize Requirements
- P0.01 [TODO] Assemble interview questions via structured UI
- P0.02 [TODO] Wait for user input (WAIT_USER)
- P0.03 [TODO] Write results to interview-qa.txt
- P0.04 [TODO] Normalize to requirements-interview.txt
- **Gate**: `contract_validator interview` + `contract_validator requirements-interview`

## P1: Input Recognition & Branch Selection
- P1.01 [TODO] Recognize user materials and context
- P1.02 [TODO] Confirm branch (research vs. non-research) with user (WAIT_USER)
- P1.03 [TODO] Update branch field in requirements-interview.txt

## P2: Data Synthesis (Branch Dependent)
- P2.XX [TODO] Execute Step 2A (Research) OR Step 2B (Source Synth)
- **Gate**: `contract_validator search-brief` OR `contract_validator source-brief`

## P3: Outline Generation
- P3.01 [TODO] Harness generation for Phase 1 & 2
- P3.02 [TODO] Create Outline subagent and execute
- P3.03 [TODO] Gate check outline.txt
- **Gate**: `contract_validator outline`

## P3.5: Style Definition
- P3.5.01 [TODO] Harness generation for Style phase
- P3.5.02 [TODO] Create Style subagent and execute
- P3.5.03 [TODO] Gate check style.json
- **Gate**: `contract_validator style`

## P4: Page Production (Parallel)
- P4.XX [TODO] For each page: Harness -> orchestrator -> PageAgent -> Verify
- **Gate**: `planning_validator` + visual confirmation

## P5: Export & Delivery
- P5.01 [TODO] Generate preview.html
- P5.02 [TODO] PNG export -> presentation-png.pptx
- P5.03 [TODO] SVG export -> presentation-svg.pptx
- P5.04 [TODO] Finalize delivery-manifest.json
- **Gate**: `contract_validator delivery-manifest`
