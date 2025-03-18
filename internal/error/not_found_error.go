package error

type NotFoundError struct {
	reason    Reason
	component Component
}

func NewNotFoundError(reason Reason, component Component) NotFoundError {
	return NotFoundError{
		reason:    reason,
		component: component,
	}
}

func (e NotFoundError) Error() string {
	return "not found"
}

func (e NotFoundError) IsNotFound() bool {
	return true
}

func (e NotFoundError) GetReason() Reason {
	return e.reason
}

func (e NotFoundError) GetComponent() Component {
	return e.component
}

func IsNotFoundError(err error) bool {
	cause := UnwrapAll(err)
	nfe, ok := cause.(interface {
		IsNotFound() bool
	})
	return ok && nfe.IsNotFound()
}
