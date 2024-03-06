package typeconv

type TypeConvError struct {
	// ShortDescription is a key that can be used to uniquely identify a conversion
	// failure type. It is friendly to display in frontends instead of the longer
	// Message.
	// We use this instead of an ENUM as the size can be quite large.
	ShortDescription string
	// Message denotes the message to display detailing the conversion failure.
	Message string
	// Blocking indicates whether the conversion failure blocks a statement
	// from being marked as "complete".
	Blocking bool
}

func (e *TypeConvError) Error() string {
	return e.ShortDescription
}
