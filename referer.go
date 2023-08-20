package proxy

const (
	None         = "none"
	Blocked      = "blocked"
	ValidReferer = "valid_referer"
)

type Referer struct {
	// "" none
	// "" 表示如果没有referer可以跳过
	// node 包括校验不携带referer
	validType string
	// 是否配置
	blocked bool
	// 域名名单
	domain []string
}

func NewReferer() *Referer {
	return &Referer{domain: make([]string, 0)}
}
