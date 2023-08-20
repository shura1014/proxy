package proxy

const (
	RequestMethod  = "request_method"
	RequestPath    = "request_path"
	InvalidReferer = "invalid_referer"

	Return = "return"
)

var SupportKey = []string{RequestMethod, RequestPath, InvalidReferer}

type Expression struct {
	key        string
	expect     string
	returnCode int
	headers    map[string]string
}

func NewExpression() *Expression {
	return &Expression{
		headers: make(map[string]string),
	}
}
