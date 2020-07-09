package job

import (
	"context"
)

type Job interface {
	StepID() string
	Name() string
	Do(ctx context.Context, jobID string, body []byte) error

	// 時間かかるのでgoroutineで呼ぶべき
	Deploy(ctx context.Context) error
}
