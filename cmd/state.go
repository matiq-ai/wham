package cmd

// State-related concrete Command Structs (Verbs)

type GetStateCmd struct {
	Target string `arg:"" help:"Step name to get state for, or 'all'"`
}

type DeleteStateCmd struct {
	Target string `arg:"" help:"Step name to delete state for, or 'all'"`
	Yes    bool   `help:"Bypass confirmation prompt." short:"y"`
}

// State-related command groups (objects)

// StateCmd holds subcommands for managing state.
type StateCmd struct {
	Get    GetStateCmd    `cmd:"" help:"Get the final state of a step or all steps."`
	Delete DeleteStateCmd `cmd:"" help:"Delete the state file for a step or all steps." aliases:"rm"`
}

// State-related command implementations

func (g *GetStateCmd) Run(ctx *Context) error {
	if g.Target == "all" {
		return ctx.WHAM.ShowExecutionSummary(ctx.OutputFormat)
	}
	return ctx.WHAM.GetStepState(g.Target, ctx.OutputFormat)
}

func (d *DeleteStateCmd) Run(ctx *Context) error {
	return ctx.WHAM.DeleteStepState(d.Target, ctx.OutputFormat, d.Yes)
}
