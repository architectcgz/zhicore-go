package main

import (
	"log"

	authruntime "github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/runtime"
)

func main() {
	// This process root stays side-effect free until repository, cache, token signer and config wiring exist.
	// Failing fast here is safer than pretending the service can boot without its required runtime owners.
	if _, err := authruntime.Build(authruntime.Deps{}); err != nil {
		log.Fatalf("build auth runtime module: %v", err)
	}
}
