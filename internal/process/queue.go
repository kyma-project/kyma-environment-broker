package process

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

type Executor interface {
	Execute(operationID string) (time.Duration, error)
}

type Queue struct {
	queue     workqueue.TypedRateLimitingInterface[string]
	executor  Executor
	waitGroup sync.WaitGroup
	log       *slog.Logger
	name      string

	speedFactor       int64
	workersInUseGauge prometheus.Gauge
}

var queueWorkersInUseMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "kcp",
	Subsystem: "keb_v2",
	Name:      "queue_workers_in_use",
	Help:      "Number of queue workers currently processing items",
}, []string{"queue_name"})

func NewQueue(executor Executor, log *slog.Logger, name string) *Queue {
	// add queue name field that could be logged later on
	return &Queue{
		queue:             workqueue.NewTypedRateLimitingQueueWithConfig(workqueue.DefaultTypedControllerRateLimiter[string](), workqueue.TypedRateLimitingQueueConfig[string]{Name: "operations"}),
		executor:          executor,
		waitGroup:         sync.WaitGroup{},
		log:               log.With("queueName", name),
		speedFactor:       1,
		name:              name,
		workersInUseGauge: queueWorkersInUseMetric.WithLabelValues(name),
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

}

// SpeedUp changes speedFactor parameter to reduce time between processing operations.
// This method should only be used for testing purposes
func (q *Queue) SpeedUp(speedFactor int64) {
	q.speedFactor = speedFactor
	q.log.Info(fmt.Sprintf("queue speed factor set to %d", speedFactor))
}

func (q *Queue) createWorker(queue workqueue.TypedRateLimitingInterface[string], process func(id string) (time.Duration, error), stopCh <-chan struct{}, waitGroup *sync.WaitGroup, log *slog.Logger, nameId string) {
	go func() {
		wait.Until(q.worker(queue, process, log, nameId), time.Second, stopCh)
		waitGroup.Done()
	}()
}

func (q *Queue) worker(queue workqueue.TypedRateLimitingInterface[string], process func(key string) (time.Duration, error), log *slog.Logger, workerNameId string) func() {
	return func() {
		exit := false
		for !exit {
			exit = func() bool {
				key, shutdown := queue.Get()
				if shutdown {
					log.Info("shutting down")
					return true
				}

				q.workersInUseGauge.Inc()
				id := key
				workerLogger := log.With("operationID", id)
				workerLogger.Info(fmt.Sprintf("about to process item %s, queue length is %d", id, q.queue.Len()))

				defer func() {
					q.workersInUseGauge.Dec()
					if err := recover(); err != nil {
						workerLogger.Error(fmt.Sprintf("panic error from process: %v. Stacktrace: %s", err, debug.Stack()))
					}
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
