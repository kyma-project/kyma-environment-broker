package process

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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
	log                     logrus.FieldLogger
	name                    string
	workerExecutionTimes    map[string]time.Time
	warnAfterTime           time.Duration
	healthCheckIntervalTime time.Duration

	speedFactor int64
}

func NewQueue(executor Executor, log logrus.FieldLogger, name string, warnAfterTime, healthCheckIntervalTime time.Duration) *Queue {
	// add queue name field that could be logged later on
	return &Queue{
		queue:                   workqueue.NewRateLimitingQueueWithConfig(workqueue.DefaultControllerRateLimiter(), workqueue.RateLimitingQueueConfig{Name: "operations"}),
		executor:                executor,
		waitGroup:               sync.WaitGroup{},
		log:                     log.WithField("queueName", name),
		speedFactor:             1,
		name:                    name,
		workerExecutionTimes:    make(map[string]time.Time),
		warnAfterTime:           warnAfterTime,
		healthCheckIntervalTime: healthCheckIntervalTime,
	}
}

func (q *Queue) Add(processId string) {
	q.queue.Add(processId)
	q.log.Infof("added item %s to the queue %s, queue length is %d", processId, q.name, q.queue.Len())
}

func (q *Queue) AddAfter(processId string, duration time.Duration) {
	q.queue.AddAfter(processId, duration)
	q.log.Infof("item %s will be added to the queue %s after duration of %d, queue length is %d", processId, q.name, duration, q.queue.Len())
}

func (q *Queue) ShutDown() {
	q.log.Infof("shutting down the queue, queue length is %d", q.queue.Len())
	q.queue.ShutDown()
}

func (q *Queue) Run(stop <-chan struct{}, workersAmount int) {
	q.log.Infof("starting %d worker(s), queue length is %d", workersAmount, q.queue.Len())
	for i := 0; i < workersAmount; i++ {
		q.waitGroup.Add(1)

		workerLogger := q.log.WithField("workerId", i)
		workerLogger.Infof("starting worker with id %d", i)

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
	q.log.Infof("queue speed factor set to %d", speedFactor)
}

func (q *Queue) createWorker(queue workqueue.RateLimitingInterface, process func(id string) (time.Duration, error), stopCh <-chan struct{}, waitGroup *sync.WaitGroup, log logrus.FieldLogger, nameId string) {
	go func() {
		log.Info("worker routine - starting")
		wait.Until(q.worker(queue, process, log, nameId), time.Second, stopCh)
		waitGroup.Done()
		log.Info("worker done")
	}()
}

func (q *Queue) worker(queue workqueue.RateLimitingInterface, process func(key string) (time.Duration, error), log logrus.FieldLogger, nameId string) func() {
	return func() {
		exit := false
		for !exit {
			exit = func() bool {
				key, shutdown := queue.Get()
				if shutdown {
					log.Infof("shutting down")
					return true
				}

				id := key.(string)
				log = log.WithField("operationID", id)
				log.Infof("about to process item %s, queue length is %d", id, q.queue.Len())
				q.logAndUpdateWorkerTimes(key.(string), nameId, log)

				defer func() {
					if err := recover(); err != nil {
						log.Errorf("panic error from process: %v. Stacktrace: %s", err, debug.Stack())
					}
					queue.Done(key)
					log.Info("queue done processing")
				}()

				when, err := process(id)
				if err == nil && when != 0 {
					log.Infof("Adding %q item after %s, queue length %d", id, when, q.queue.Len())
					afterDuration := time.Duration(int64(when) / q.speedFactor)
					queue.AddAfter(key, afterDuration)
					return false
				}
				if err != nil {
					log.Errorf("Error from process: %v", err)
				}

				queue.Forget(key)
				log.Infof("item for %s has been processed, no retry, element forgotten", id)

				return false
			}()
		}
	}
}

func (q *Queue) logAndUpdateWorkerTimes(key string, name string, log logrus.FieldLogger) {
	// log time
	now := time.Now()
	lastTime, ok := q.workerExecutionTimes[name]
	if ok {
		log.Infof("execution - worker %s last execution time %s, executed after %s", name, lastTime, now.Sub(lastTime))
	}
	q.workerExecutionTimes[name] = now

	log.Infof("processing item %s, queue length is %d", key, q.queue.Len())
}

func (q *Queue) logWorkersSummary() {
	healthCheckLog := q.log.WithField("summary", q.name)
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("health - queue length %d", q.queue.Len()))

	for name, lastTime := range q.workerExecutionTimes {
		timeSinceLastExecution := time.Since(lastTime)

		buffer.WriteString(fmt.Sprintf(", [worker %s, last execution time: %s, since last execution: %s]", name, lastTime, timeSinceLastExecution))
	}

	healthCheckLog.Info(buffer.String())

	for name, lastTime := range q.workerExecutionTimes {
		timeSinceLastExecution := time.Since(lastTime)
		if timeSinceLastExecution > q.warnAfterTime {
			healthCheckLog.Infof("worker %s exceeded allowed limit of %s since last execution, its last execution is %s, time since last execution %s", name, q.warnAfterTime, lastTime, timeSinceLastExecution)
		}
	}
}
