package zookeeper

import (
	"fmt"

	"github.com/newrelic/nri-kafka/src/args"
)

// Path takes a zookeeper path and prepends it with argument args.GlobalArgs
func Path(path string) string {
	return fmt.Sprintf("%s%s", args.GlobalArgs.ZookeeperPath, path)
}
