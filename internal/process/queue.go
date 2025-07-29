package process

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

type Executor interface {
	Execute(operationID string) (time.Duration, error)
}

type Queue struct {
	queue                   workqueue.RateLimitingInterface
	executor                Executor
	waitGroup               sync.WaitGroup
	log                     *slog.Logger
	name                    string
	workerExecutionTimes    map[string]time.Time
	workerLastKeys          map[string]string
	warnAfterTime           time.Duration
	healthCheckIntervalTime time.Duration

	speedFactor int64
}

func NewQueue(executor Executor, log *slog.Logger, name string, warnAfterTime, healthCheckIntervalTime time.Duration) *Queue {
	// add queue name field that could be logged later on
	return &Queue{
		queue:                   workqueue.NewRateLimitingQueueWithConfig(workqueue.DefaultControllerRateLimiter(), workqueue.RateLimitingQueueConfig{Name: "operations"}),
		executor:                executor,
		waitGroup:               sync.WaitGroup{},
		log:                     log.With("queueName", name),
		speedFactor:             1,
		name:                    name,
		workerExecutionTimes:    make(map[string]time.Time),
		workerLastKeys:          make(map[string]string),
		warnAfterTime:           warnAfterTime,
		healthCheckIntervalTime: healthCheckIntervalTime,
	}
}

func (q *Queue) Add(processId string) {
	q.queue.Add(processId)
	q.log.Info(fmt.Sprintf("added item %s to the queue %s, queue length is %d", processId, q.name, q.queue.Len()))
}

func (q *Queue) AddAfter(processId string, duration time.Duration) {
	q.queue.AddAfter(processId, duration)
	q.log.Info(fmt.Sprintf("item %s will be added to the queue %s after duration of %d, queue length is %d", processId, q.name, duration, q.queue.Len()))
}

func (q *Queue) ShutDown() {
	q.log.Info(fmt.Sprintf("shutting down the queue, queue length is %d", q.queue.Len()))
	q.queue.ShutDown()
}

func (q *Queue) Run(stop <-chan struct{}, workersAmount int) {
	for i := 0; i < workersAmount; i++ {
		q.waitGroup.Add(1)

		workerLogger := q.log.With("workerId", i)

		q.createWorker(q.queue, q.executor.Execute, stop, &q.waitGroup, workerLogger, fmt.Sprintf("%s-%d", q.name, i))
	}

	// go routine for checking worker execution times and warning if worker has not run for specified amount of time
	go func() {
		wait.Until(func() {
			q.logWorkersSummary()
		}, q.healthCheckIntervalTime, stop)
	}()

}

// SpeedUp changes speedFactor parameter to reduce time between processing operations.
// This method should only be used for testing purposes
func (q *Queue) SpeedUp(speedFactor int64) {
	q.speedFactor = speedFactor
	q.log.Info(fmt.Sprintf("queue speed factor set to %d", speedFactor))
}

func (q *Queue) createWorker(queue workqueue.RateLimitingInterface, process func(id string) (time.Duration, error), stopCh <-chan struct{}, waitGroup *sync.WaitGroup, log *slog.Logger, nameId string) {
	go func() {
		wait.Until(q.worker(queue, process, log, nameId), time.Second, stopCh)
		waitGroup.Done()
	}()
}

func (q *Queue) worker(queue workqueue.RateLimitingInterface, process func(key string) (time.Duration, error), log *slog.Logger, workerNameId string) func() {
	return func() {
		exit := false
		for !exit {
			exit = func() bool {
				key, shutdown := queue.Get()
				if shutdown {
					log.Info("shutting down")
					return true
				}

				id := key.(string)
				workerLogger := log.With("operationID", id)
				workerLogger.Info(fmt.Sprintf("about to process item %s, queue length is %d", id, q.queue.Len()))
				q.updateWorkerTime(id, workerNameId, workerLogger)

				defer func() {
					if err := recover(); err != nil {
						workerLogger.Error(fmt.Sprintf("panic error from process: %v. Stacktrace: %s", err, debug.Stack()))
					}
					q.removeTimeIfInWarnMargin(id, workerNameId, log)
					queue.Done(key)
					workerLogger.Info("queue done processing")
				}()

				when, err := process(id)
				if err == nil && when != 0 {
					workerLogger.Info(fmt.Sprintf("Adding %q item after %s, queue length %d", id, when, q.queue.Len()))
					afterDuration := time.Duration(int64(when) / q.speedFactor)
					queue.AddAfter(key, afterDuration)
					return false
				}
				if err != nil {
					workerLogger.Error(fmt.Sprintf("Error from process: %v", err))
				}

				queue.Forget(key)
				workerLogger.Info(fmt.Sprintf("item for %s has been processed, no retry, element forgotten", id))

				return false
			}()
		}
	}
}

func (q *Queue) updateWorkerTime(key string, workerNameId string, log *slog.Logger) {
	q.workerExecutionTimes[workerNameId] = time.Now()
	q.workerLastKeys[workerNameId] = key
	log.Info(fmt.Sprintf("updating worker time, processing item %s, queue length is %d", key, q.queue.Len()))
}

func (q *Queue) removeTimeIfInWarnMargin(key string, workerNameId string, log *slog.Logger) {

	processingStart, ok := q.workerExecutionTimes[workerNameId]

	if !ok {
		log.Warn(fmt.Sprintf("worker %s has no start time", workerNameId))
		return
	}

	processingTime := time.Since(processingStart)

	if processingTime > q.warnAfterTime {
		log.Info(fmt.Sprintf("worker %s exceeded allowed limit of %s since last execution while processing %s", workerNameId, q.warnAfterTime, key))
	}

	delete(q.workerExecutionTimes, workerNameId)
	delete(q.workerLastKeys, workerNameId)
}

func (q *Queue) logWorkersSummary() {
	healthCheckLog := q.log.With("summary", q.name)

	for name, lastTime := range q.workerExecutionTimes {
		timeSinceLastExecution := time.Since(lastTime)
		lastItemKey, ok := q.workerLastKeys[name]
		if !ok {
			lastItemKey = "'already removed'"
		}

		if timeSinceLastExecution > q.warnAfterTime {
			healthCheckLog.Info(fmt.Sprintf("worker %s exceeded allowed limit of %s since last execution, its last execution is %s, time since last execution %s, while processing item %s", name, q.warnAfterTime, lastTime, timeSinceLastExecution, lastItemKey))
		}
	}
}
