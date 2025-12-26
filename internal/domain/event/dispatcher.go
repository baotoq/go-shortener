package event

// Handler is the interface for handling domain events.
type Handler interface {
	Handle(event Event) error
}

// Dispatcher dispatches events to registered handlers.
type Dispatcher struct {
	handlers map[string][]Handler
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string][]Handler),
	}
}

// Register registers a handler for a specific event type.
func (d *Dispatcher) Register(eventName string, handler Handler) {
	d.handlers[eventName] = append(d.handlers[eventName], handler)
}

// Dispatch dispatches an event to all registered handlers.
func (d *Dispatcher) Dispatch(e Event) error {
	handlers, ok := d.handlers[e.EventName()]
	if !ok {
		return nil
	}

	for _, handler := range handlers {
		if err := handler.Handle(e); err != nil {
			return err
		}
	}
	return nil
}

// DispatchAll dispatches multiple events.
func (d *Dispatcher) DispatchAll(events []Event) error {
	for _, e := range events {
		if err := d.Dispatch(e); err != nil {
			return err
		}
	}
	return nil
}
