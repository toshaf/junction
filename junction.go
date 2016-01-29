// Represents a junction between any number of inbound channels
// of values and a target value. Updates sent down the source
// channels are reflected int the state tracked by the call to
// Run(Output) and sent down Output
package junction

import (
	"reflect"
)

// The location to place the output channel (*chan Target)
type Output interface{}

// Represents a source of updates to
// the target value
type Source struct {
	// The channel down which update values
	// arrive (of type chan X)
	Input interface{}
	// A func that takes two args:
	// - A pointer to the value being updated (*Target)
	// - A single argument to use in the update (X)
	// This can be a function of the form func(*Target, X), or
	// a method of the form (*Target) func(X)
	Update interface{}
	// A pointer to the Target value being updated, or a func
	// of the form func(X) (*Target, bool)
	Model interface{}
}

// Associates the value pointed to by output
// with all the Sources specified and creates
// a new output channel in the location pointed
// to by Output - this location is overwritten by
// this call so doesn't need to be
// (read shouldn't be) initialised.
// A Goroutine is then started that uses the sources
// to keep the model(s) updated and publishes new
// versions through the output channel
func New(output Output, sources []Source) {
	if err := Validate(output, sources); err != nil {
		panic(err)
	}

	channel := makeOutputChan(output)

	go func() {
		cases := buildCases(sources)

		for {
			choice, value, ok := reflect.Select(cases)
			if !ok {
				panic("Input closed")
			}

			source := sources[choice]

			update := reflect.ValueOf(source.Update)

			target, ok := source.getTarget(value)

			if ok {
				update.Call([]reflect.Value{target, value})

				channel.Send(target.Elem())
			}
		}
	}()
}

// Performs type checking on the specified output and sources
// but does not create the output channel or process any of the
// sources
func Validate(output Output, sources []Source) error {
	otype := reflect.TypeOf(output)
	if otype.Kind() != reflect.Ptr {
		return Err("Output must be a pointer")
	}
	if otype.Elem().Kind() != reflect.Chan {
		return Err("Output must be a pointer to a channel")
	}
	// this is a pointer to chan X
	mtype := reflect.TypeOf(output).Elem().Elem()
	for _, src := range sources {
		target := reflect.ValueOf(src.Model)
		if target.Kind() == reflect.Ptr {
			if mtype != target.Elem().Type() {
				return Err("Differing target types, want %v got %v", mtype, target.Elem().Type())
			}
			continue
		}
		if target.Kind() == reflect.Func {
			ttype := target.Type()
			itype := reflect.TypeOf(src.Input).Elem()
			if ttype.NumIn() == 1 && itype.AssignableTo(ttype.In(0)) {
				if ttype.NumOut() == 2 {
					r0 := ttype.Out(0)
					if mtype != r0.Elem() {
						return Err("Differing target types")
					}
					r1 := ttype.Out(1)
					if r1 != reflect.TypeOf(true) {
						return Err("Second return arg is not a bool")
					}
					continue
				}
			}
		}
		return Err("Target must be a pointer or func(X)(*Model,bool), not " + target.String())
	}

	return nil
}

func (source Source) getTarget(update reflect.Value) (reflect.Value, bool) {
	model := reflect.ValueOf(source.Model)
	if model.Kind() == reflect.Ptr {
		return model, true
	}

	if model.Kind() == reflect.Func {
		results := model.Call([]reflect.Value{update})
		if results[1].Interface().(bool) {
			return results[0], true
		}

		return reflect.Value{}, false
	}

	panic("Wrong kind of model: " + model.Kind().String())
}

func makeOutputChan(o Output) reflect.Value {
	output := reflect.ValueOf(o)

	channel := reflect.MakeChan(output.Elem().Type(), 0)

	output.Elem().Set(channel)

	return channel
}

func buildCases(sources []Source) []reflect.SelectCase {
	cases := make([]reflect.SelectCase, 0, len(sources))

	for _, src := range sources {
		cases = append(cases, reflect.SelectCase{
			Chan: reflect.ValueOf(src.Input),
			Dir:  reflect.SelectRecv,
		})
	}

	return cases
}
