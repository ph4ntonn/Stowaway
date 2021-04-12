package manager

import "net"

const (
	B_NEWBACKWARD = iota
	B_GETSEQCHAN
	B_ADDCONN
	B_GETDATACHAN
)

type backwardManager struct {
	backwardSeqMap   map[uint64]string
	backwardMap      map[string]*backward
	BackwardMessChan chan interface{}

	TaskChan   chan *BackwardTask
	ResultChan chan *backwardResult
	Done       chan bool
}

type BackwardTask struct {
	Mode int
	Seq  uint64

	Listener       net.Listener
	Port           string
	BackwardSocket net.Conn
}

type backwardResult struct {
	OK bool

	SeqChan  chan uint64
	DataChan chan []byte
}

type backward struct {
	listener net.Listener
	seqChan  chan uint64

	backwardStatusMap map[uint64]*backwardStatus
}

type backwardStatus struct {
	dataChan chan []byte
	conn     net.Conn
}

func newBackwardManager() *backwardManager {
	manager := new(backwardManager)

	manager.backwardSeqMap = make(map[uint64]string)
	manager.backwardMap = make(map[string]*backward)
	manager.BackwardMessChan = make(chan interface{}, 5)

	manager.ResultChan = make(chan *backwardResult)
	manager.TaskChan = make(chan *BackwardTask)
	manager.Done = make(chan bool)

	return manager
}

func (manager *backwardManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case B_NEWBACKWARD:
			manager.newBackward(task)
		case B_GETSEQCHAN:
			manager.getSeqChan(task)
			<-manager.Done
		case B_ADDCONN:
			manager.addConn(task)
		case B_GETDATACHAN:
			manager.getDataChan(task)
		}
	}
}

func (manager *backwardManager) newBackward(task *BackwardTask) {
	manager.backwardMap[task.Port] = new(backward)
	manager.backwardMap[task.Port].listener = task.Listener
	manager.backwardMap[task.Port].backwardStatusMap = make(map[uint64]*backwardStatus)
	manager.backwardMap[task.Port].seqChan = make(chan uint64)
	manager.ResultChan <- &backwardResult{OK: true}
}

func (manager *backwardManager) getSeqChan(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.Port]; ok {
		manager.ResultChan <- &backwardResult{
			OK:      true,
			SeqChan: manager.backwardMap[task.Port].seqChan,
		}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}
}

func (manager *backwardManager) addConn(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.Port]; ok {
		manager.backwardSeqMap[task.Seq] = task.Port
		manager.backwardMap[task.Port].backwardStatusMap[task.Seq] = new(backwardStatus)
		manager.backwardMap[task.Port].backwardStatusMap[task.Seq].conn = task.BackwardSocket
		manager.backwardMap[task.Port].backwardStatusMap[task.Seq].dataChan = make(chan []byte)
		manager.ResultChan <- &backwardResult{OK: true}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}
}

func (manager *backwardManager) getDataChan(task *BackwardTask) {
	if _, ok := manager.backwardMap[task.Port]; ok {
		manager.ResultChan <- &backwardResult{
			OK:       true,
			DataChan: manager.backwardMap[task.Port].backwardStatusMap[task.Seq].dataChan,
		}
	} else {
		manager.ResultChan <- &backwardResult{OK: false}
	}
}
