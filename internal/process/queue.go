package process

import (
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
	queue                workqueue.RateLimitingInterface
	executor             Executor
	waitGroup            sync.WaitGroup
	log                  logrus.FieldLogger
	name                 string
	workerExecutionTimes map[string]time.Time

	speedFactor int64
}

func NewQueue(executor Executor, log logrus.FieldLogger, name string) *Queue {
	// add queue name field that could be logged later on
	return &Queue{
		queue:                workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "operations"),
		executor:             executor,
		waitGroup:            sync.WaitGroup{},
		log:                  log.WithField("queueName", name),
		speedFactor:          1,
		name:                 name,
		workerExecutionTimes: make(map[string]time.Time),
	}
}

func (q *Queue) Add(processId string) {
	q.log.Infof("adding item to the queue %s", processId)
	q.queue.Add(processId)
}

func (q *Queue) AddAfter(processId string, duration time.Duration) {
	q.log.Infof("adding item to the queue %s after duration of %s", processId, duration)
	q.queue.AddAfter(processId, duration)
}

func (q *Queue) ShutDown() {
	q.log.Infof("shutting down the queue")
	q.queue.ShutDown()
}

func (q *Queue) Run(stop <-chan struct{}, workersAmount int) {
	q.log.Infof("starting workers, queue len is %s", q.queue.Len())
	for i := 0; i < workersAmount; i++ {
		q.waitGroup.Add(1)

		workerLogger := q.log.WithField("workerId", i)
		workerLogger.Infof("starting worker with id %d", i)

		q.createWorker(q.queue, q.executor.Execute, stop, &q.waitGroup, workerLogger, fmt.Sprintf("%s-%d", q.name, i))
	}

	wait.Until(func() {
		healthCheckLog := q.log.WithField("healthCheck", q.name)
		for name, lastTime := range q.workerExecutionTimes {
			healthCheckLog.Infof("worker %s last execution time %s", name, lastTime)
			timeSinceLastExecution := time.Since(lastTime)
			if timeSinceLastExecution > 5*time.Minute {
				healthCheckLog.Warnf("no execution for %s", timeSinceLastExecution)
			}
		}
	}, time.Minute, stop)
}

// SpeedUp changes speedFactor parameter to reduce time between processing operations.
// This method should only be used for testing purposes
func (q *Queue) SpeedUp(speedFactor int64) {
	q.speedFactor = speedFactor
	q.log.Infof("queue speed factor set to %s", speedFactor)
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

				q.logWorkerTimes(key.(string), nameId, log)

				id := key.(string)
				log = log.WithField("operationID", id)
				defer func() {
					if err := recover(); err != nil {
						log.Errorf("panic error from process: %v. Stacktrace: %s", err, debug.Stack())
					}
					queue.Done(key)
					log.Info("queue done processing")
				}()

				when, err := process(id)
				if err == nil && when != 0 {
					log.Infof("Adding %q item after %s", id, when)
					afterDuration := time.Duration(int64(when) / q.speedFactor)
					queue.AddAfter(key, afterDuration)
					return false
				}
				if err != nil {
					log.Errorf("Error from process: %v", err)
				}

				queue.Forget(key)
				log.Infof("Item for %q has been processed, no retry, element forgotten", id)

				return false
			}()
		}
	}
}

func (q *Queue) logWorkerTimes(key string, name string, log logrus.FieldLogger) {
	// log time
	now := time.Now()
	lastTime, ok := q.workerExecutionTimes[name]
	if ok {
		log.Infof("worker %s last execution time %s, executed after %s seconds", name, lastTime, now.Sub(lastTime).Seconds())
	}
	q.workerExecutionTimes[name] = now

	log.Infof("processing item %s, queue len is %d", key, q.queue.Len())
}
