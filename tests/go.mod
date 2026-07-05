module github.com/architectcgz/zhicore-go/tests

go 1.26.0

require (
	github.com/architectcgz/zhicore-go/libs/contracts v0.0.0
	github.com/lib/pq v1.10.9
)

replace github.com/architectcgz/zhicore-go/libs/contracts => ../libs/contracts
