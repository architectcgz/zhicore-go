package main

import (
	"fmt"
	"log"

	userruntime "github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/runtime"
)

func main() {
	if err := buildModule(); err != nil {
		log.Fatal(err)
	}
}

func buildModule() error {
	// 当前切片只建立进程根和 runtime 边界。生产 repository、File client、
	// outbox、cache 和配置加载落地前，启动必须 fail fast，避免伪装成可运行服务。
	if _, err := userruntime.Build(userruntime.Deps{}); err != nil {
		return fmt.Errorf("build user runtime module: %w", err)
	}
	return nil
}
