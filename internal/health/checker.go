package health

import "context"

// Checker 抽象一个可被探活的依赖。
// HTTP 层只依赖这个接口，就不用知道底层到底连的是 MySQL、Redis 还是其他组件。
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}
