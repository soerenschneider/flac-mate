package internal

type GenericResult[T any] struct {
	Operation string
	Data      T

	Execute func(action *GenericResult[T]) error
}

func (a *GenericResult[T]) Run() error {
	if a.Execute == nil {
		return nil
	}
	return a.Execute(a)
}
