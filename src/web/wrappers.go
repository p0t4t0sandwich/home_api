package web

import "github.com/a-h/templ"

// FuncWrapper A generic wrapper type that returns a component
type FuncWrapper[T any] func(i T) templ.Component
