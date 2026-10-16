package object

const PROMISE_OBJ = "PROMISE"

// Awaitable is a value `await` can wait for. IsAwaitable reports whether the
// value is really a promise, as a JavaScript value may or may not be one.
// Then registers the functions to call with the result or the reason it
// failed, once it settles.
type Awaitable interface {
	Object
	IsAwaitable() bool
	Then(resolve func(Object), reject func(Object))
}

// Promise is a result that becomes available later, such as the result of
// a function that is waiting for something with await.
type Promise struct {
	settled  bool
	rejected bool
	value    Object

	// Paused is set for the promise of a function that paused at an
	// await. Its caller waits for it, as if it had called the function
	// directly, since calling a function that awaits needs no await.
	Paused bool

	resolvers []func(Object)
	rejecters []func(Object)
}

func NewPromise() *Promise {
	return &Promise{}
}

func (p *Promise) Type() ObjectType { return PROMISE_OBJ }
func (p *Promise) Inspect() string {
	switch {
	case !p.settled:
		return "<promise>"
	case p.rejected:
		return "<promise rejected: " + p.value.Inspect() + ">"
	default:
		return "<promise: " + p.value.Inspect() + ">"
	}
}

func (p *Promise) IsAwaitable() bool { return true }

// Settled reports whether the promise has a result yet.
func (p *Promise) Settled() bool { return p.settled }

// Result returns the value the promise resolved to, or the reason it was
// rejected, with rejected set.
func (p *Promise) Result() (value Object, rejected bool) {
	return p.value, p.rejected
}

// Then calls resolve or reject when the promise settles, right away if it
// already has.
func (p *Promise) Then(resolve func(Object), reject func(Object)) {
	if !p.settled {
		p.resolvers = append(p.resolvers, resolve)
		p.rejecters = append(p.rejecters, reject)
		return
	}
	if p.rejected {
		reject(p.value)
	} else {
		resolve(p.value)
	}
}

// Resolve settles the promise with a value. Settling it again does nothing.
func (p *Promise) Resolve(value Object) {
	p.settle(value, false)
}

// Reject settles the promise as failed, with the reason it failed.
func (p *Promise) Reject(reason Object) {
	p.settle(reason, true)
}

func (p *Promise) settle(value Object, rejected bool) {
	if p.settled {
		return
	}
	p.settled, p.rejected, p.value = true, rejected, value

	callbacks := p.resolvers
	if rejected {
		callbacks = p.rejecters
	}
	p.resolvers, p.rejecters = nil, nil
	for _, callback := range callbacks {
		callback(value)
	}
}
