package btpmgrcreds

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type foo struct {
	x, y int
}

type fakeService struct {
	reachable bool
	ticks     int
}

func (fs *fakeService) fakeCall1() (int, error) {
	return 91, nil
}

func (fs *fakeService) fakeCall2() (foo, error) {
	foo := foo{
		x: 1,
		y: 4,
	}
	return foo, nil
}

func (fs *fakeService) fakeCall3() (any, error) {
	return nil, fmt.Errorf("expected error")
}

func (fs *fakeService) fakeCall4() (any, error) {
	return nil, nil
}

func (fs *fakeService) fakeCall5() (foo, error) {
	if fs.reachable {
		return foo{x: -1, y: -1999}, nil
	} else {
		return foo{}, fmt.Errorf("expected error")
	}
}

func TestCallWithRetry(t *testing.T) {
	t.Run("call without any problems 1", func(t *testing.T) {
		fakeService := &fakeService{}
		result, err := CallWithRetry(func() (int, error) {
			return fakeService.fakeCall1()
		}, 1, time.Second*1)
		assert.NoError(t, err)
		assert.Equal(t, 91, result)
	})

	t.Run("call without any problems 2", func(t *testing.T) {
		fakeService := &fakeService{}
		result, err := CallWithRetry(func() (foo, error) {
			return fakeService.fakeCall2()
		}, 1, time.Second*1)
		assert.NoError(t, err)
		assert.Equal(t, 1, result.x)
		assert.Equal(t, 4, result.y)
	})

	t.Run("call with error 1", func(t *testing.T) {
		fakeService := &fakeService{}
		result, err := CallWithRetry(func() (any, error) {
			return fakeService.fakeCall3()
		}, 1, time.Second*1)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("call with error 2", func(t *testing.T) {
		fakeService := &fakeService{}
		result, err := CallWithRetry(func() (any, error) {
			return fakeService.fakeCall3()
		}, 1, time.Second*1)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("call with no error and no response", func(t *testing.T) {
		fakeService := &fakeService{}
		result, err := CallWithRetry(func() (any, error) {
			return fakeService.fakeCall4()
		}, 1, time.Second*1)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("service become reachable after some time", func(t *testing.T) {
		//Service will be not reachable for 10s.
		//The retry will try to connect 5 times * 5 seconds = 25 seconds. 25 > 10 so Result should be returned without error
		fakeService := &fakeService{ticks: 10}
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range time.Tick(time.Second * 1) {
				fakeService.ticks--
				if fakeService.ticks <= 0 {
					fakeService.reachable = true
					return
				}
			}
		}()
		result, err := CallWithRetry(func() (foo, error) {
			return fakeService.fakeCall5()
		}, 5, time.Second*5)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result)
		assert.Equal(t, -1, result.x)
		assert.Equal(t, -1999, result.y)
		wg.Wait()
	})

	t.Run("service is not reachable for all time", func(t *testing.T) {
		//Service will be not reachable for 10s.
		//The retry will try to connect 2 times * 2 seconds = 4 seconds. 4 < 10 so Result should be returned with error
		fakeService := &fakeService{ticks: 10}
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range time.Tick(time.Second * 1) {
				fakeService.ticks--
				if fakeService.ticks <= 0 {
					fakeService.reachable = true
					return
				}
			}
		}()
		result, err := CallWithRetry(func() (foo, error) {
			return fakeService.fakeCall5()
		}, 2, time.Second*2)
		assert.Error(t, err)
		assert.Empty(t, result)
		wg.Wait()
	})
}
