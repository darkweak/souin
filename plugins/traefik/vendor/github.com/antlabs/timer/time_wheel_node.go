package timer

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/antlabs/stl/list"
)

const (
	haveStop = uint32(1)
)

// 先使用sync.Mutex实现功能
// 后面使用cas优化
type Time struct {
	timeNode
	sync.Mutex

	// |---16bit---|---16bit---|------32bit-----|
	// |---level---|---index---|-------seq------|
	// level 在near盘子里就是1, 在T2ToTt[0]盘子里就是2起步
	// index 就是各自盘子的索引值
	// seq   自增id
	version uint64
}

func newTimeHead(level uint64, index uint64) *Time {
	head := &Time{}
	head.version = genVersionHeight(level, index)
	head.Init()
	return head
}

func genVersionHeight(level uint64, index uint64) uint64 {
	return level<<(32+16) | index<<32
}

func (t *Time) lockPushBack(node *timeNode, level uint64, index uint64) {
	t.Lock()
	defer t.Unlock()
	if atomic.LoadUint32(&node.stop) == haveStop {
		return
	}

	t.AddTail(&node.Head)
	atomic.StorePointer(&node.list, unsafe.Pointer(t))
	//更新节点的version信息
	atomic.StoreUint64(&node.version, atomic.LoadUint64(&t.version))
}

type timeNode struct {
	expire     uint64
	userExpire time.Duration
	callback   func()
	stop       uint32
	list       unsafe.Pointer //存放表头信息
	version    uint64         //保存节点版本信息
	isSchedule bool

	list.Head
}

// 一个timeNode节点有4个状态
// 1.存在于初始化链表中
// 2.被移动到tmp链表
// 3.1 和 3.2是if else的状态
// 	3.1被移动到new链表
// 	3.2直接执行
// 1和3.1状态是没有问题的
// 2和3.2状态会是没有锁保护下的操作,会有数据竞争
func (t *timeNode) Stop() {

	atomic.StoreUint32(&t.stop, haveStop)

	// 使用版本号算法让timeNode知道自己是否被移动了
	// timeNode的version和表头的version一样表示没有被移动可以直接删除
	// 如果不一样，可能在第2或者3.2状态，使用惰性删除
	cpyList := (*Time)(atomic.LoadPointer(&t.list))
	cpyList.Lock()
	defer cpyList.Unlock()
	if atomic.LoadUint64(&t.version) != atomic.LoadUint64(&cpyList.version) {
		return
	}

	cpyList.Del(&t.Head)
}
