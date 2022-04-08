package manager

const (
	S_CHECKTCP = iota
	S_CHECKUDP
	S_UPDATEUDPHEADER
	S_GETTCPDATACHAN
	S_GETUDPCHANS
	S_GETUDPHEADER
	S_CLOSETCP
	S_CHECKSOCKSREADY
	S_FORCESHUTDOWN
)

type socksManager struct {
	socksStatusMap map[uint64]*socksStatus
	SocksMessChan  chan interface{}

	TaskChan   chan *SocksTask
	ResultChan chan *socksResult
}

type SocksTask struct {
	Mode int
	Seq  uint64

	SocksHeaderAddr string
	SocksHeader     []byte
}

type socksResult struct {
	OK bool

	SocksSeqExist  bool
	DataChan       chan []byte
	ReadyChan      chan string
	SocksID        uint64
	SocksUDPHeader []byte
}

type socksStatus struct {
	isUDP bool
	tcp   *tcpSocks
	udp   *udpSocks
}

type tcpSocks struct {
	dataChan chan []byte
}

type udpSocks struct {
	dataChan    chan []byte
	readyChan   chan string
	headerPairs map[string][]byte
}

func newSocksManager() *socksManager {
	manager := new(socksManager)

	manager.socksStatusMap = make(map[uint64]*socksStatus)
	manager.SocksMessChan = make(chan interface{}, 5)

	manager.ResultChan = make(chan *socksResult)
	manager.TaskChan = make(chan *SocksTask)

	return manager
}

func (manager *socksManager) run() {
	for {
		task := <-manager.TaskChan

		switch task.Mode {
		case S_GETTCPDATACHAN:
			manager.getTCPDataChan(task)
		case S_GETUDPCHANS:
			manager.getUDPChans(task)
		case S_GETUDPHEADER:
			manager.getUDPHeader(task)
		case S_CHECKTCP:
			manager.checkTCP(task)
		case S_CHECKUDP:
			manager.checkUDP(task)
		case S_UPDATEUDPHEADER:
			manager.updateUDPHeader(task)
		case S_CLOSETCP:
			manager.closeTCP(task)
		case S_CHECKSOCKSREADY:
			manager.checkSocksReady()
		case S_FORCESHUTDOWN:
			manager.forceShutdown()
		}
	}
}

func (manager *socksManager) getTCPDataChan(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.ResultChan <- &socksResult{
			SocksSeqExist: true,
			DataChan:      manager.socksStatusMap[task.Seq].tcp.dataChan,
		}
	} else {
		manager.socksStatusMap[task.Seq] = new(socksStatus)
		manager.socksStatusMap[task.Seq].tcp = new(tcpSocks)
		manager.socksStatusMap[task.Seq].tcp.dataChan = make(chan []byte, 5) // register it!
		manager.ResultChan <- &socksResult{
			SocksSeqExist: false,
			DataChan:      manager.socksStatusMap[task.Seq].tcp.dataChan,
		} // tell upstream result
	}
}

func (manager *socksManager) getUDPChans(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.ResultChan <- &socksResult{
			OK:        true,
			DataChan:  manager.socksStatusMap[task.Seq].udp.dataChan,
			ReadyChan: manager.socksStatusMap[task.Seq].udp.readyChan,
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) checkTCP(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.ResultChan <- &socksResult{OK: true}
	} else {
		manager.ResultChan <- &socksResult{OK: false} // avoid the scenario that admin conn ask to fin before "socks.buildConn()" call "updateTCP()"
	}
}

func (manager *socksManager) checkUDP(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.socksStatusMap[task.Seq].isUDP = true
		manager.socksStatusMap[task.Seq].udp = new(udpSocks)
		manager.socksStatusMap[task.Seq].udp.dataChan = make(chan []byte, 5)
		manager.socksStatusMap[task.Seq].udp.readyChan = make(chan string)
		manager.socksStatusMap[task.Seq].udp.headerPairs = make(map[string][]byte)
		manager.ResultChan <- &socksResult{OK: true} // tell upstream work done
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) updateUDPHeader(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		manager.socksStatusMap[task.Seq].udp.headerPairs[task.SocksHeaderAddr] = task.SocksHeader
	}
	manager.ResultChan <- &socksResult{}
}

func (manager *socksManager) getUDPHeader(task *SocksTask) {
	if _, ok := manager.socksStatusMap[task.Seq]; ok {
		if _, ok := manager.socksStatusMap[task.Seq].udp.headerPairs[task.SocksHeaderAddr]; ok {
			manager.ResultChan <- &socksResult{
				OK:             true,
				SocksUDPHeader: manager.socksStatusMap[task.Seq].udp.headerPairs[task.SocksHeaderAddr],
			}
		} else {
			manager.ResultChan <- &socksResult{OK: false}
		}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) closeTCP(task *SocksTask) {
	close(manager.socksStatusMap[task.Seq].tcp.dataChan)

	if manager.socksStatusMap[task.Seq].isUDP {
		close(manager.socksStatusMap[task.Seq].udp.dataChan)
		close(manager.socksStatusMap[task.Seq].udp.readyChan)
		manager.socksStatusMap[task.Seq].udp.headerPairs = nil
	}

	delete(manager.socksStatusMap, task.Seq) // upstream not waiting
}

func (manager *socksManager) checkSocksReady() {
	if len(manager.socksStatusMap) == 0 {
		manager.ResultChan <- &socksResult{OK: true}
	} else {
		manager.ResultChan <- &socksResult{OK: false}
	}
}

func (manager *socksManager) forceShutdown() {
	for seq, status := range manager.socksStatusMap {
		close(status.tcp.dataChan)

		if status.isUDP {
			close(status.udp.dataChan)
			close(status.udp.readyChan)
			status.udp.headerPairs = nil
		}

		delete(manager.socksStatusMap, seq)
	}

	manager.ResultChan <- &socksResult{OK: true}
}
