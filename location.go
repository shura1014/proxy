package proxy

import (
	"github.com/shura1014/balance"
)

type Location struct {
	Rule    string
	Servers []*Server
	// 健康检查
	Check *Check
	// 负载均衡策略
	Balance balance.Balance[*Server]

	// 是否是一个静态资源服务器
	IsStatic bool
	// 如果有root 那么就是一个静态资源服务器
	Root string
	// 如果是静态资源服务器，可以提供默认访问的首页地址
	Index string

	// add_header
	headers map[string]string

	expression []*Expression

	referer *Referer
}

func NewLocation() *Location {
	return &Location{
		Servers: make([]*Server, 0),
		Check:   &Check{},
		headers: make(map[string]string),
	}
}
