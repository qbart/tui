package main

import (
	"fmt"
	"os"

	"tui/core"
	"tui/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	spec := core.NewPipelineSpec("sample-cicd", []core.StepSpec{
		{ID: "checkout", JobName: "checkout", Status: core.StatusGreen},
		{ID: "lint", JobName: "lint", DependsOn: []core.StepID{"checkout"}},
		{ID: "unit-core", JobName: "unit core", DependsOn: []core.StepID{"checkout"}},
		{ID: "unit-api", JobName: "unit api", DependsOn: []core.StepID{"checkout"}},
		{ID: "build-ui-assets", JobName: "build ui assets", DependsOn: []core.StepID{"checkout"}, Status: core.StatusBlue},
		{ID: "build-api-image", JobName: "build api image", DependsOn: []core.StepID{"checkout"}, Status: core.StatusOrange},
		{ID: "build-worker-image", JobName: "build worker image", DependsOn: []core.StepID{"checkout"}},
		{ID: "policy-scan", JobName: "policy scan", DependsOn: []core.StepID{"lint"}},
		{ID: "int-postgres", JobName: "int postgres", DependsOn: []core.StepID{"unit-core", "unit-api"}},
		{ID: "int-sqlite", JobName: "int sqlite", DependsOn: []core.StepID{"unit-core"}, Status: core.StatusGray},
		{ID: "int-duckdb", JobName: "int duckdb", DependsOn: []core.StepID{"unit-api"}},
		{ID: "e2e-web", JobName: "e2e web", DependsOn: []core.StepID{"build-ui-assets", "build-api-image"}, Status: core.StatusPurple},
		{ID: "e2e-mobile", JobName: "e2e mobile", DependsOn: []core.StepID{"build-ui-assets", "build-api-image"}, Status: core.StatusRed},
		{ID: "worker-smoke", JobName: "worker smoke", DependsOn: []core.StepID{"build-worker-image"}},
		{ID: "quality-gate", JobName: "quality gate", DependsOn: []core.StepID{"policy-scan", "int-postgres", "int-sqlite", "int-duckdb", "e2e-web", "e2e-mobile", "worker-smoke"}},
		{ID: "package-api", JobName: "package api", DependsOn: []core.StepID{"quality-gate"}, Status: core.StatusYellow},
		{ID: "package-worker", JobName: "package worker", DependsOn: []core.StepID{"quality-gate"}},
		{ID: "package-ui", JobName: "package ui", DependsOn: []core.StepID{"quality-gate"}},
		{ID: "deploy-staging", JobName: "deploy staging", DependsOn: []core.StepID{"package-api", "package-worker", "package-ui"}},
		{ID: "smoke-staging", JobName: "smoke staging", DependsOn: []core.StepID{"deploy-staging"}},
		{ID: "perf-staging", JobName: "perf staging", DependsOn: []core.StepID{"deploy-staging"}},
		{ID: "approve-prod", JobName: "approve prod", DependsOn: []core.StepID{"smoke-staging", "perf-staging"}},
		{ID: "deploy-prod", JobName: "deploy prod", DependsOn: []core.StepID{"approve-prod"}},
		{ID: "verify-prod", JobName: "verify prod", DependsOn: []core.StepID{"deploy-prod"}},
		{ID: "notify-success", JobName: "notify success", DependsOn: []core.StepID{"verify-prod"}},
		{ID: "sbom-generate", JobName: "sbom generate", DependsOn: []core.StepID{"package-api", "package-worker", "package-ui"}},
		{ID: "security-sign", JobName: "security sign", DependsOn: []core.StepID{"sbom-generate"}},
		{ID: "release-notes", JobName: "release notes", DependsOn: []core.StepID{"quality-gate"}},
		{ID: "publish-release", JobName: "publish release", DependsOn: []core.StepID{"security-sign", "release-notes"}},
		{ID: "chaos-staging", JobName: "chaos staging", DependsOn: []core.StepID{"deploy-staging"}},
		{ID: "rollback-drill", JobName: "rollback drill", DependsOn: []core.StepID{"chaos-staging"}},
		{ID: "audit-prod", JobName: "audit prod", DependsOn: []core.StepID{"deploy-prod"}},
		{ID: "notify-security", JobName: "notify security", DependsOn: []core.StepID{"audit-prod"}},
		// Disconnected test data (not connected to existing pipeline nodes).
		{ID: "sandbox-a-prepare", JobName: "sandbox a prepare"},
		{ID: "sandbox-a-run", JobName: "sandbox a run", DependsOn: []core.StepID{"sandbox-a-prepare"}},
		{ID: "sandbox-a-report", JobName: "sandbox a report", DependsOn: []core.StepID{"sandbox-a-run"}},
		{ID: "sandbox-b-prepare", JobName: "sandbox b prepare"},
		{ID: "sandbox-b-run", JobName: "sandbox b run", DependsOn: []core.StepID{"sandbox-b-prepare"}},
		{ID: "sandbox-b-cleanup", JobName: "sandbox b cleanup", DependsOn: []core.StepID{"sandbox-b-run"}},
		{ID: "orphan-healthcheck", JobName: "orphan healthcheck"},
		{ID: "orphan-metrics", JobName: "orphan metrics"},
		{ID: "perf-a-setup", JobName: "perf a setup"},
		{ID: "perf-a-run", JobName: "perf a run", DependsOn: []core.StepID{"perf-a-setup"}},
		{ID: "perf-a-report", JobName: "perf a report", DependsOn: []core.StepID{"perf-a-run"}},
		{ID: "perf-b-setup", JobName: "perf b setup"},
		{ID: "perf-b-baseline", JobName: "perf b baseline", DependsOn: []core.StepID{"perf-b-setup"}},
		{ID: "perf-b-loadgen", JobName: "perf b loadgen", DependsOn: []core.StepID{"perf-b-setup"}},
		{ID: "perf-b-stress", JobName: "perf b stress", DependsOn: []core.StepID{"perf-b-loadgen"}},
		{ID: "perf-b-validate", JobName: "perf b validate", DependsOn: []core.StepID{"perf-b-baseline", "perf-b-stress"}},
		{ID: "perf-b-profile", JobName: "perf b profile", DependsOn: []core.StepID{"perf-b-stress"}},
		{ID: "perf-b-compare", JobName: "perf b compare", DependsOn: []core.StepID{"perf-b-validate", "perf-b-profile"}},
		{ID: "lab-seed", JobName: "lab seed"},
		{ID: "lab-simulate", JobName: "lab simulate", DependsOn: []core.StepID{"lab-seed"}},
		{ID: "lab-analyze", JobName: "lab analyze", DependsOn: []core.StepID{"lab-simulate"}},
		{ID: "lab-archive", JobName: "lab archive", DependsOn: []core.StepID{"lab-analyze"}},
		{ID: "isolated-alpha", JobName: "isolated alpha"},
		{ID: "isolated-beta", JobName: "isolated beta"},
	})

	p := tea.NewProgram(tui.NewPipelineModel(spec, "example"), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running program: %v\n", err)
		os.Exit(1)
	}
}
