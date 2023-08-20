package proxy

type Server struct {
	check *Check
	// 只有探测成功才有效
	valid   bool
	Address string

	// 连续的
	successCount int
	failCount    int
	// 配置balance为weight时需要
	weight int
}

func (server *Server) IsValid() bool {
	return server.valid
}

func (server *Server) GetWeight() int {
	return server.weight
}

// Success 探测成功
func (server *Server) Success() {
	server.successCount++
	server.failCount = 0
	if server.successCount >= server.check.success {
		server.valid = true
	}
}

// Fail 探测失败
func (server *Server) Fail() {
	server.failCount++
	server.successCount = 0
	if server.failCount >= server.check.fail {
		server.valid = false
	}
}
