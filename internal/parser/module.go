package parser

import "go.uber.org/fx"

// Module provides the parser dependencies
var Module = fx.Module("parser",
	fx.Provide(
		fx.Annotate(
			NewSwaggerParser,
			fx.As(new(Parser)),
		),
		NewAdjuster,
	),
)
