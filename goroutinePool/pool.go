package goroutinepool

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	Handler func(v ...interface{})
	Params  []interface{}
}

type Pool struct {
	capacity       uint64
	runningWorkers uint64
	status         int64
	chTask         chan *Task
	sync.Mutex
	PanicHandler func(interface{})
}

var ErrInvalidPoolCap = errors.New("invalid pool cap")

const (
	Running  = 1
	Stopping = 0
)

func NewPool(capacity uint64) (*Pool, error) {
	if capacity <= 0 {
		return nil, ErrInvalidPoolCap
	}
	return &Pool{
		capacity: capacity,
		status:   Running,
		chTask:   make(chan *Task, capacity),
	}, nil
}

// func (p *Pool)run() {
// 	p.runningWorkers++
// 	go func ()  {
// 		defer func ()  {
// 			p.runningWorkers--
// 		}()

// 		for{
// 			select {
// 			case task,ok:=<-p.chTask:
// 				if !ok {
// 					return
// 				}
// 				task.Handler(task.Params...)
// 			}
// 		}
// 	}()
// }

func (p *Pool) incRunning() {
	atomic.AddUint64(&p.runningWorkers, 1)
}

func (p *Pool) decRunning() {
	atomic.AddUint64(&p.runningWorkers, ^uint64(0))
}

func (p *Pool) GetRunningWorks() uint64 {
	return atomic.LoadUint64(&p.runningWorkers)
}

func (p *Pool) run() {
	p.incRunning()
	go func() {
		defer func() {
			p.decRunning()
			if r := recover(); r != nil {
				if p.PanicHandler != nil {
					p.PanicHandler(r)
				} else {
					log.Printf("Worker panic: %s\n", r)
				}
			}
		}()

		for {
			select {
			case task, ok := <-p.chTask:
				if !ok {
					return
				}
				task.Handler(task.Params...)
			}
		}
	}()
}

func (p *Pool) GetCap() uint64 {
	return p.capacity
}

func (p *Pool) setStatus(status int64) bool {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if p.status == status {
		return false
	}
	p.status = status
	return true
}

var ErrPoolAlreadyClosed = errors.New("pool already closed")

func (p *Pool) put(task *Task) error {
	p.Lock()
	defer p.Unlock()
	if p.status == Stopping { // 如果任务池处于关闭状态, 再 put 任务会返回 ErrPoolAlreadyClosed 错误
		return ErrPoolAlreadyClosed
	}
	if p.GetRunningWorks() < p.GetCap() {
		p.run()
	}
	if p.status == Running {
		p.chTask <- task
	}

	return nil
}

func (p *Pool) Close() {
	p.setStatus(Stopping) // 设置 status 为已停止

	for len(p.chTask) > 0 { // 阻塞等待所有任务被 worker 消费
		time.Sleep(1e6) // 防止等待任务清空 cpu 负载突然变大, 这里小睡一下
	}

	close(p.chTask) // 关闭任务队列
}

func main() {
	// 创建任务池
	pool, err := NewPool(10)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 20; i++ {
		// 任务放入池中
		pool.put(&Task{
			Handler: func(v ...interface{}) {
				fmt.Println(v)
			},
			Params: []interface{}{i},
		})
	}

	time.Sleep(1e9) // 等待执行
}
