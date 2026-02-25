package main

import (
	"fmt"
	"os"
	"time"
	"github.com/qbart/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	spec := tui.NewPipelineSpec("sample-cicd", []tui.StepSpec{
		{ID: "checkout", JobName: "checkout", Status: tui.StatusGreen},
		{ID: "lint", JobName: "lint", DependsOn: []tui.StepID{"checkout"}},
		{ID: "unit-core", JobName: "unit core", DependsOn: []tui.StepID{"checkout"}},
		{ID: "unit-api", JobName: "unit api", DependsOn: []tui.StepID{"checkout"}},
		{ID: "build-ui-assets", JobName: "build ui assets", DependsOn: []tui.StepID{"checkout"}, Status: tui.StatusBlue},
		{ID: "build-api-image", JobName: "build api image", DependsOn: []tui.StepID{"checkout"}, Status: tui.StatusOrange},
		{ID: "build-worker-image", JobName: "build worker image", DependsOn: []tui.StepID{"checkout"}},
		{ID: "policy-scan", JobName: "policy scan", DependsOn: []tui.StepID{"lint"}},
		{ID: "int-postgres", JobName: "int postgres", DependsOn: []tui.StepID{"unit-core", "unit-api"}},
		{ID: "int-sqlite", JobName: "int sqlite", DependsOn: []tui.StepID{"unit-core"}, Status: tui.StatusGray},
		{ID: "int-duckdb", JobName: "int duckdb", DependsOn: []tui.StepID{"unit-api"}},
		{ID: "e2e-web", JobName: "e2e web", DependsOn: []tui.StepID{"build-ui-assets", "build-api-image"}, Status: tui.StatusPurple},
		{ID: "e2e-mobile", JobName: "e2e mobile", DependsOn: []tui.StepID{"build-ui-assets", "build-api-image"}, Status: tui.StatusRed},
		{ID: "worker-smoke", JobName: "worker smoke", DependsOn: []tui.StepID{"build-worker-image"}},
		{ID: "quality-gate", JobName: "quality gate", DependsOn: []tui.StepID{"policy-scan", "int-postgres", "int-sqlite", "int-duckdb", "e2e-web", "e2e-mobile", "worker-smoke"}},
		{ID: "package-api", JobName: "package api", DependsOn: []tui.StepID{"quality-gate"}, Status: tui.StatusYellow},
		{ID: "package-worker", JobName: "package worker", DependsOn: []tui.StepID{"quality-gate"}},
		{ID: "package-ui", JobName: "package ui", DependsOn: []tui.StepID{"quality-gate"}},
		{ID: "deploy-staging", JobName: "deploy staging", DependsOn: []tui.StepID{"package-api", "package-worker", "package-ui"}},
		{ID: "smoke-staging", JobName: "smoke staging", DependsOn: []tui.StepID{"deploy-staging"}},
		{ID: "perf-staging", JobName: "perf staging", DependsOn: []tui.StepID{"deploy-staging"}},
		{ID: "approve-prod", JobName: "approve prod", DependsOn: []tui.StepID{"smoke-staging", "perf-staging"}},
		{ID: "deploy-prod", JobName: "deploy prod", DependsOn: []tui.StepID{"approve-prod"}},
		{ID: "verify-prod", JobName: "verify prod", DependsOn: []tui.StepID{"deploy-prod"}},
		{ID: "notify-success", JobName: "notify success", DependsOn: []tui.StepID{"verify-prod"}},
		{ID: "sbom-generate", JobName: "sbom generate", DependsOn: []tui.StepID{"package-api", "package-worker", "package-ui"}},
		{ID: "security-sign", JobName: "security sign", DependsOn: []tui.StepID{"sbom-generate"}},
		{ID: "release-notes", JobName: "release notes", DependsOn: []tui.StepID{"quality-gate"}},
		{ID: "publish-release", JobName: "publish release", DependsOn: []tui.StepID{"security-sign", "release-notes"}},
		{ID: "chaos-staging", JobName: "chaos staging", DependsOn: []tui.StepID{"deploy-staging"}},
		{ID: "rollback-drill", JobName: "rollback drill", DependsOn: []tui.StepID{"chaos-staging"}},
		{ID: "audit-prod", JobName: "audit prod", DependsOn: []tui.StepID{"deploy-prod"}},
		{ID: "notify-security", JobName: "notify security", DependsOn: []tui.StepID{"audit-prod"}},
		// Disconnected test data (not connected to existing pipeline nodes).
		{ID: "sandbox-a-prepare", JobName: "sandbox a prepare"},
		{ID: "sandbox-a-run", JobName: "sandbox a run", DependsOn: []tui.StepID{"sandbox-a-prepare"}},
		{ID: "sandbox-a-report", JobName: "sandbox a report", DependsOn: []tui.StepID{"sandbox-a-run"}},
		{ID: "sandbox-b-prepare", JobName: "sandbox b prepare"},
		{ID: "sandbox-b-run", JobName: "sandbox b run", DependsOn: []tui.StepID{"sandbox-b-prepare"}},
		{ID: "sandbox-b-cleanup", JobName: "sandbox b cleanup", DependsOn: []tui.StepID{"sandbox-b-run"}},
		{ID: "orphan-healthcheck", JobName: "orphan healthcheck"},
		{ID: "orphan-metrics", JobName: "orphan metrics"},
		{ID: "perf-a-setup", JobName: "perf a setup"},
		{ID: "perf-a-run", JobName: "perf a run", DependsOn: []tui.StepID{"perf-a-setup"}},
		{ID: "perf-a-report", JobName: "perf a report", DependsOn: []tui.StepID{"perf-a-run"}},
		{ID: "perf-b-setup", JobName: "perf b setup"},
		{ID: "perf-b-baseline", JobName: "perf b baseline", DependsOn: []tui.StepID{"perf-b-setup"}},
		{ID: "perf-b-loadgen", JobName: "perf b loadgen", DependsOn: []tui.StepID{"perf-b-setup"}},
		{ID: "perf-b-stress", JobName: "perf b stress", DependsOn: []tui.StepID{"perf-b-loadgen"}},
		{ID: "perf-b-validate", JobName: "perf b validate", DependsOn: []tui.StepID{"perf-b-baseline", "perf-b-stress"}},
		{ID: "perf-b-profile", JobName: "perf b profile", DependsOn: []tui.StepID{"perf-b-stress"}},
		{ID: "perf-b-compare", JobName: "perf b compare", DependsOn: []tui.StepID{"perf-b-validate", "perf-b-profile"}},
		{ID: "lab-seed", JobName: "lab seed"},
		{ID: "lab-simulate", JobName: "lab simulate", DependsOn: []tui.StepID{"lab-seed"}},
		{ID: "lab-analyze", JobName: "lab analyze", DependsOn: []tui.StepID{"lab-simulate"}},
		{ID: "lab-archive", JobName: "lab archive", DependsOn: []tui.StepID{"lab-analyze"}},
		{ID: "isolated-alpha", JobName: "isolated alpha"},
		{ID: "isolated-beta", JobName: "isolated beta"},
	})

	pipelineModel := tui.NewPipelineModel(spec)
	p := tea.NewProgram(pipelineModel, tea.WithAltScreen())

	go func() {
		time.Sleep(1 * time.Second)
		p.Send(tui.SetStepStatusMsg{StepID: "perf-a-setup", Status: tui.StatusGray})
		time.Sleep(150 * time.Millisecond)
		p.Send(tui.SetStepSpinnerMsg{StepID: "perf-a-setup", Spinner: true})
		p.Send(tui.SetStepStatusMsg{StepID: "perf-a-setup", Status: tui.StatusYellow})
		time.Sleep(2 * time.Second)
		p.Send(tui.SetStepSpinnerMsg{StepID: "perf-a-setup", Spinner: false})
		p.Send(tui.SetStepStatusMsg{StepID: "perf-a-setup", Status: tui.StatusGreen})
		p.Send(tui.SetStepSelectedMsg{StepID: "perf-b-stress"})
		time.Sleep(500 * time.Millisecond)
		p.Send(tui.SetStepSelectedMsg{StepID: ""})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running program: %v\n", err)
		os.Exit(1)
	}
}
