# AGENTS.UI.md

## Purpose
This document defines the current visual behavior of the terminal UI.

## Layout
- Full-screen Bubble Tea app in alt-screen mode.
- Two vertical regions:
  - Content area: takes all rows except the last row.
  - Footer: single-line bar pinned to the bottom.

## Content Area
- Background: black (`theme.ContentBackground`).
- Foreground: white (`theme.ContentForeground`).
- Padding:
  - Top: 1 line.
  - Left/Right: 1 character each side.
  - Bottom: 0.
- No pipeline header text is rendered in content.
- Content displays only the pipeline graph.

## Footer
- Exactly one line.
- Background: dark gray (`theme.FooterBackground`).
- Foreground: white (`theme.FooterForeground`).
- Shows run status text (e.g. `run:running | q to quit`).

## Step Visuals
- Step text format: `" <icon?> <job-name> "` with one leading and one trailing space.
- If icon is empty, only job name is shown with side spaces.
- Step has no border.
- Step colors:
  - Background: black (`theme.StepBackground`, `#000000`).
  - Foreground: white (`theme.StepForeground`, `#ffffff`).
- Step width rule: text rune count + 2 padding characters.

## Pipeline Columns and Positioning
- Steps are placed by dependency levels (left to right columns).
- Dependency-free steps appear in the first column.
- Dependent steps appear in subsequent columns.
- Vertical placement is computed from dependency layout logic.
- Steps in the same column are visually separated by one extra blank line.

## Connector Visuals
- No extra spaces around connector segments.
- Connector component is configurable:
  - Width.
  - Type: `Solid` or `Dashed`.
  - Color (passed from top-level renderer).
- Current top-level config:
  - Type: `Solid`.
  - Width: `5`.
  - Color: `theme.ArrowColor`.
  - Background: `theme.ContentBackground`.

## Connector Rules (Current)
- Single dependency (step -> one target in next column):
  - Horizontal line only.
- Multiple dependencies (step -> multiple targets in next column):
  - Source row uses split marker with center `┳` and horizontal continuation.
  - Intermediate target rows use `┣` with horizontal continuation.
  - Last target row uses `┗` with horizontal continuation.
  - Spacer rows between source/targets use centered vertical continuation.
- All connector glyphs are box-drawing heavy style for clean joins.

## Keyboard
- `q` or `ctrl+c` quits the app.
