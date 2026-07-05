package main

import (
	"fmt"
	"log"

	commentruntime "github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/runtime"
)

func main() {
	if err := buildModule(); err != nil {
		log.Fatal(err)
	}
}

func buildModule() error {
	// 当前切片只建立进程根和 runtime 边界。生产 repository、RabbitMQ
	// dispatcher、下游 client 和配置加载落地前，启动必须 fail fast。
	if _, err := commentruntime.Build(commentruntime.Deps{}); err != nil {
		return fmt.Errorf("build comment runtime module: %w", err)
	}
	return nil
}
