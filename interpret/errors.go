package interpret

// Runtime errors
const (
	NO_ERROR = iota
	DESTRUCTURE_ERROR
	TYPE_ERROR
)

var errorText = []string{
	NO_ERROR:          "No error",
	DESTRUCTURE_ERROR: "Couldn't destructure object",
	TYPE_ERROR:        "Unexpected type",
}

func runtimeErrorText(errNum int) string {
	return errorText[errNum]
}
