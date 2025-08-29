package cmd

import "fmt"

// Step-related concrete Command Structs (Verbs)

type RunStepCmd struct {
	Target string `arg:"" help:"Step name to run, or 'all'"`
	Force  bool   `help:"Force the step to run, ignoring state." short:"f"`
	From   string `help:"Start execution from this step (inclusive). Requires 'all' target."`
	To     string `help:"End execution at this step (inclusive). Requires 'all' target."`
}

type GetStepCmd struct {
	Target string `arg:"" help:"Step name to get configuration for, or 'all'"`
}
type DescribeStepCmd struct {
	Target string `arg:"" help:"Step name to describe, or 'all'"`
}
type ValidateStepCmd struct {
	Target string `arg:"" help:"Step name to validate, or 'all'"`
}

// Step-related command groups (objects)

// StepCmd holds subcommands for operating on steps.
type StepCmd struct {
	Run      RunStepCmd      `cmd:"" help:"Run a step or all steps. Use --force to ignore state."`
	Get      GetStepCmd      `cmd:"" help:"Show a step's static configuration in a structured format."`
	Describe DescribeStepCmd `cmd:"" help:"Show a step's detailed configuration and current state."`
	Validate ValidateStepCmd `cmd:"" help:"Validate a step's definition or all steps."`
}

// Step-related command implementations

func (r *RunStepCmd) Run(ctx *Context) error {
	if (r.From != "" || r.To != "") && r.Target != "all" {
		return fmt.Errorf("--from and --to flags can only be used with the 'all' target")
	}
	if r.Target == "all" {
		if err := ctx.WHAM.RunAllSteps(r.Force, r.From, r.To); err != nil {
			return err
		}
		// After a successful run, print the summary using the format from the context.
		if _, err := fmt.Println("\n✅ Workflow execution finished."); err != nil {
			return err
		}
		return ctx.WHAM.ShowExecutionSummary(ctx.OutputFormat)
	}
	return ctx.WHAM.RunStep(r.Target, r.Force)
}

func (g *GetStepCmd) Run(ctx *Context) error {
	return ctx.WHAM.GetStep(g.Target, ctx.OutputFormat)
}

func (d *DescribeStepCmd) Run(ctx *Context) error {
	if d.Target == "all" {
		return ctx.WHAM.DescribeAllSteps()
	}
	return ctx.WHAM.DescribeStep(d.Target)
}

func (v *ValidateStepCmd) Run(ctx *Context) error {
	return ctx.WHAM.GetValidationStatus(v.Target, ctx.OutputFormat)
}
