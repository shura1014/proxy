package main

import (
	"github.com/shura1014/common"
	"github.com/shura1014/common/env"
	"github.com/shura1014/proxy"
)

// go build -o proxy main.go
func main() {
	conf, _ := env.GetEnv("config")
	gws := proxy.ParseConfig(conf)
	for _, gw := range gws {
		go gw.Start()
	}
	common.Wait()
}
