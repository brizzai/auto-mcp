package requester

import (
	"go.uber.org/fx"
)

// Module provides the requester module dependencies
var Module = fx.Options(
	fx.Provide(
		NewHTTPRequester,
		fx.Annotate(
			NewHTTPAuthManager,
			fx.As(new(AuthManager)),
		),
		NewHTTPRequestBuilder,
	),
)
