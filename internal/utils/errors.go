package utils

func NewClientErr(clientErr error, context string, cause error) error {
	return &ClientErr{
		ClientErr: clientErr,
		Context:   context,
		Cause:     cause,
	}
}

type ClientErr struct {
	ClientErr error
	Context   string
	Cause     error
}

func (c *ClientErr) Error() string {
	if c.Context != "" {
		return c.ClientErr.Error() + " : " + c.Context
	} else {
		return c.ClientErr.Error()
	}
}

func (c *ClientErr) Unwrap() error {
	return c.ClientErr
}
