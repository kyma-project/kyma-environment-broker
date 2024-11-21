package process

import (
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
	queue     workqueue.RateLimitingInterface
	executor  Executor
	waitGroup sync.WaitGroup
	log       logrus.FieldLogger

	speedFactor int64
}

func NewQueue(executor Executor, log logrus.FieldLogger, name string) *Queue {
	// add queue name field that could be logged later on
	return &Queue{
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "operations"),
		executor:  executor,
		waitGroup: sync.WaitGroup{},
		log:       log.WithField("queueName", name),
		speedFactor: 1,
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

		q.createWorker(q.queue, q.executor.Execute, stop, &q.waitGroup, workerLogger)
	}
}

// SpeedUp changes speedFactor parameter to reduce time between processing operations.
// This method should only be used for testing purposes
func (q *Queue) SpeedUp(speedFactor int64) {
	q.speedFactor = speedFactor
	q.log.Infof("queue speed factor set to %s", speedFactor)
}

func (q *Queue) createWorker(queue workqueue.RateLimitingInterface, process func(id string) (time.Duration, error), stopCh <-chan struct{}, waitGroup *sync.WaitGroup, log logrus.FieldLogger) {
	go func() {
		log.Info("worker routine - starting")
		wait.Until(q.worker(queue, process, log), time.Second, stopCh)
		waitGroup.Done()
		log.Info("worker done")
	}()
}

func (q *Queue) worker(queue workqueue.RateLimitingInterface, process func(key string) (time.Duration, error), log logrus.FieldLogger) func() {
	return func() {
		exit := false
		for !exit {
			exit = func() bool {
				key, shutdown := queue.Get()
				if shutdown {
					log.Infof("shutting down")
					return true
				}

				log.Infof("processing item %s, queue len is %d", key, queue.Len())

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
