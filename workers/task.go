package workers

import "context"

type Task interface {
	Do(context.Context) error
	OnFinish(context.Context, error)
}

type TaskImpl struct {
	Task
	function func(context.Context) error
	callback func(context.Context, error)
}

func NewTask(function func(context.Context) error, callback func(context.Context, error)) *TaskImpl {
	return &TaskImpl{
		function: function,
		callback: callback,
	}
}

func (t *TaskImpl) Do(ctx context.Context) error {
	return t.function(ctx)
}

func (t *TaskImpl) OnFinish(ctx context.Context, err error) {
	t.callback(ctx, err)
}

type TaskWithParam struct {
	Task
	function func(context.Context, ...interface{}) error
	callback func(context.Context, error, ...interface{})
	params   []interface{}
}

func NewTaskWithParams(
	function func(context.Context, ...interface{}) error,
	callback func(context.Context, error, ...interface{}),
	params []interface{}) *TaskWithParam {
	return &TaskWithParam{
		params:   params,
		function: function,
		callback: callback,
	}
}

func (t *TaskWithParam) Do(ctx context.Context) error {
	return t.function(ctx, t.params...)
}

func (t *TaskWithParam) OnFinish(ctx context.Context, err error) {
	t.callback(ctx, err, t.params...)
}
