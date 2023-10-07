package timer

import (
	"container/heap"
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

var _ Timer = (*minHeap)(nil)

type minHeap struct {
	mu sync.Mutex
	minHeaps
	chAdd    chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	wait     sync.WaitGroup
	runCount uint32 // 测试时使用
}

// 一次性定时器
func (m *minHeap) AfterFunc(expire time.Duration, callback func()) TimeNoder {
	return m.addCallback(expire, nil, callback, false)
}

// 周期性定时器
func (m *minHeap) ScheduleFunc(expire time.Duration, callback func()) TimeNoder {
	return m.addCallback(expire, nil, callback, true)
}

// 自定义下次的时间
func (m *minHeap) CustomFunc(n Next, callback func()) TimeNoder {
	return m.addCallback(time.Duration(0), n, callback, true)
}

// 加任务
func (m *minHeap) addCallback(expire time.Duration, n Next, callback func(), isSchedule bool) TimeNoder {
	select {
	case <-m.ctx.Done():
		panic("cannot add a task to a closed timer")
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	node := minHeapNode{
		callback:   callback,
		userExpire: expire,
		next:       n,
		absExpire:  time.Now().Add(expire),
		isSchedule: isSchedule,
		root:       m,
	}

	if n != nil {
		node.absExpire = n.Next(time.Now())
	}

	heap.Push(&m.minHeaps, &node)
	select {
	case m.chAdd <- struct{}{}:
	default:
	}

	return &node
}

func (m *minHeap) removeTimeNode(node *minHeapNode) {
	m.mu.Lock()
	if node.index < 0 || node.index > len(m.minHeaps) || len(m.minHeaps) == 0 {
		m.mu.Unlock()
		return
	}

	heap.Remove(&m.minHeaps, node.index)
	m.mu.Unlock()
}

// 运行
// 为了避免空转cpu, 会等待一个chan, 只要AfterFunc或者ScheduleFunc被调用就会往这个chan里面写值
func (m *minHeap) Run() {
	timeout := time.Hour
	tm := time.NewTimer(timeout)
	for {
		select {
		case <-tm.C:
			for {
				m.mu.Lock()
				now := time.Now()
				if m.minHeaps.Len() == 0 {
					tm.Reset(timeout)
					m.mu.Unlock()
					goto next
				}

				for {
					first := m.minHeaps[0]

					// 时间未到直接过滤掉
					if !now.After(first.absExpire) {
						break
					}

					callback := first.callback
					if first.isSchedule {
						first.absExpire = first.Next(now)
						heap.Fix(&m.minHeaps, first.index)
					} else {
						heap.Pop(&m.minHeaps)
					}
					atomic.AddUint32(&m.runCount, 1)
					go callback()

					if m.minHeaps.Len() == 0 {
						tm.Reset(timeout)
						m.mu.Unlock()
						goto next
					}
				}

				first := m.minHeaps[0]
				if time.Now().Before(first.absExpire) {
					to := time.Duration(math.Abs(float64(time.Since(m.minHeaps[0].absExpire))))
					tm.Reset(to)
					m.mu.Unlock()
					goto next
				}
				m.mu.Unlock()
			}
		case <-m.chAdd:
			m.mu.Lock()
			// 极端情况，加完任务立即给删除了, 判断下当前堆中是否有元素
			if m.minHeaps.Len() > 0 {
				tm.Reset(m.minHeaps[0].absExpire.Sub(time.Now()))
			}
			m.mu.Unlock()
			// 进入事件循环，如果为空就会从事件循环里面退出
		case <-m.ctx.Done():
			// 等待所有任务结束
			m.wait.Wait()
			return
		}
	next:
	}
}

// 停止所有定时器
func (m *minHeap) Stop() {
	m.cancel()
}

func newMinHeap() (mh *minHeap) {
	mh = &minHeap{}
	heap.Init(&mh.minHeaps)
	mh.chAdd = make(chan struct{}, 1)
	mh.ctx, mh.cancel = context.WithCancel(context.TODO())
	return
}
