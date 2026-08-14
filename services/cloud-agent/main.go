package main

import (
	"os"
	"pythonrpa/orch"
	"strconv"
	"time"
)

func main() {

	var queueId, executionId, workerId, tenantUniqueName, environmentUniqueName, botId, randomText, chipherText string

	tenantUniqueName = os.Getenv("TENANT_UNAME")

	poolInterval, err := strconv.ParseInt(os.Getenv("POOL_INTERVAL"), 10, 0)
	if err != nil {
		poolInterval = 10
	}

	for {
		queueId, executionId, workerId, environmentUniqueName, botId, randomText, chipherText, err = CheckQueue(tenantUniqueName)

		if queueId != "" && err == nil {
			LaunchK8sJob(queueId, executionId, workerId, tenantUniqueName, environmentUniqueName, botId, randomText, chipherText)
			time.Sleep(time.Duration(poolInterval) * time.Second)
		} else {
			if workerId != "" {
				orch.ReleaseWorker(workerId)
			}
		}

		time.Sleep(time.Duration(poolInterval) * time.Second)
	}

}
