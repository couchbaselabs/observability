package progress

import "fmt"

type ClusterNotFoundError struct {
	uuid string
}

func (e *ClusterNotFoundError) Error() string {
	return fmt.Sprintf("cluster '%s' not found", e.uuid)
}
