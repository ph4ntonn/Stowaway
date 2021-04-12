package manager

import "net"

const (
	B_GETNEWSEQ = iota
)

type backwardManager struct {
	backwardSeq    uint64
	backwardSeqMap map[uint64]*bwSeqRelationship // map[seq](port+uuid) just for accelerate the speed of searching detail only by seq
	backwardMap    map[string]map[string]*backward

	BackwardMessChan chan interface{}
	BackwardReady    chan bool

	TaskChan   chan *BackwardTask
	ResultChan chan *backwardResult
	Done       chan bool
}

type BackwardTask struct {
	Mode int
	UUID string // node uuid
	Seq  uint64 // seq

	Port string
}

type backwardResult struct {
	OK bool

	DataChan    chan []byte
	BackwardSeq uint64
}

type backward struct {
	remotePort string

	backwardStatusMap map[uint64]*backwardStatus
}

type backwardStatus struct {
	dataChan chan []byte
	conn     net.Conn
}

type bwSeqRelationship struct {
	uuid string
	port string
}

func newBackwardManager() *backwardManager {
	manager := new(backwardManager)

	manager.backwardMap = make(map[string]map[string]*backward)
	manager.backwardSeqMap = make(map[uint64]*bwSeqRelationship)
	manager.BackwardMessChan = make(chan interface{}, 5)
	manager.BackwardReady = make(chan bool)

	manager.TaskChan = make(chan *BackwardTask)
	manager.Done = make(chan bool)
	manager.ResultChan = make(chan *backwardResult)

	return manager
}

func (manager *backwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case B_GETNEWSEQ:
			manager.getNewSeq(task)
		}
	}
}

func (manager *backwardManager) getNewSeq(task *BackwardTask) {
	manager.backwardSeqMap[manager.backwardSeq] = &bwSeqRelationship{port: task.Port, uuid: task.UUID}
	manager.ResultChan <- &backwardResult{BackwardSeq: manager.backwardSeq}
	manager.backwardSeq++
}
