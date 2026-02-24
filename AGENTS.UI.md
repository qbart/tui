# AGENTS.UI.md

## Purpose
This repository is a collection of reusable TUI components for Bubble Tea.

Current primary component:
- `Pipeline` component (`tui.PipelineModel`), which renders and controls a dependency graph of pipeline steps.

## Package
- Single public package: `tui` (import path: `tui/tui`).
- Domain types and UI model live in the same package.

## Pipeline Component

### Public Types
- `type StepID string`
- `type StepVisualStatus string`
- `type StepSpec struct { ID StepID; Status StepVisualStatus; JobName string; DependsOn []StepID }`
- `type PipelineSpec struct { ID string; Steps []StepSpec }`
- `type PipelineModel struct` (Bubble Tea model)

### Status Values
- `StatusBlack`
- `StatusGray`
- `StatusGreen`
- `StatusRed`
- `StatusYellow`
- `StatusBlue`
- `StatusOrange`
- `StatusPurple`

### Build Spec
- `NewPipelineSpec(id string, steps []StepSpec) PipelineSpec`

### Build UI Model
- `NewPipelineModel(spec PipelineSpec) PipelineModel`

### Runtime Control API
- `SetStepStatus(stepID StepID, status StepVisualStatus) error`
- `SetStepSpinner(stepID StepID, spinning bool) error`
- `SetStepSelected(stepID StepID) error` (`""` clears selection)

### Runtime Message API (for `Program.Send`)
- `SetStepStatusMsg`
- `SetStepSpinnerMsg`
- `SetStepSelectedMsg`

## Pipeline Rendering Behavior

### Layout
- Full-screen Bubble Tea-friendly rendering.
- Graph is padded by 1 line top and bottom.
- Left/right padding is 1 char when width allows.
- Horizontal/vertical scrolling supported via viewport offsets.

### Step Bricks
- Label format: `"  <job-name>  "` (2 spaces on each side).
- Spinner replaces last rune in the label when enabled.
- Width = rune count of `job-name` + 4.
- No border.

### Selection
- Selected step uses `SelectedBg`/`SelectedFg` theme colors.
- Selection also highlights upstream and downstream edges.

### Connectors
- Connectors are drawn directly with box-drawing glyphs on a canvas.
- Highlighted edges use `ArrowSelectedColor`, non-highlighted use `ArrowColor`.

### Keyboard
- `q` / `ctrl+c`: quit.
- `h`,`j`,`k`,`l` or arrows: scroll.

## Theme Tokens (Pipeline-Relevant)
- `ContentBackground`, `ContentForeground`
- `StatusBlackBg/Fg`
- `StatusGrayBg/Fg`
- `StatusGreenBg/Fg`
- `StatusRedBg/Fg`
- `StatusYellowBg/Fg`
- `StatusBlueBg/Fg`
- `StatusOrangeBg/Fg`
- `StatusPurpleBg/Fg`
- `SelectedBg`, `SelectedFg`
- `ArrowColor`, `ArrowSelectedColor`
