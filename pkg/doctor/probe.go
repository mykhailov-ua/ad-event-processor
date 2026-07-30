package doctor

import "context"

type Probe interface {
	Name() string
	Run(ctx context.Context) Result
}
