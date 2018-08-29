package zookeeper

import (
	"fmt"
	"github.com/newrelic/nri-kafka/src/args"
)

func Path(path string) string {
	return fmt.Sprintf("%s%s", args.GlobalArgs.ZookeeperPath, path)
}
