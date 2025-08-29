package cmd

// DAG-related concrete command structs (verbs)

type GetDAGCmd struct{}

// DAG-related command groups (objects)

// DAGCmd holds subcommands for the DAG.
type DAGCmd struct {
	Get GetDAGCmd `cmd:"" help:"Get the entire workflow's execution graph (DAG)."`
}

// DAG-related command implementations

func (g *GetDAGCmd) Run(ctx *Context) error {
	return ctx.WHAM.GetDAG(ctx.OutputFormat)
}
