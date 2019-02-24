package widgets

import (
	"fmt"
	"log"
	"sync"
	"time"

	psNet "github.com/shirou/gopsutil/net"

	ui "github.com/cjbassi/gotop/src/termui"
	"github.com/cjbassi/gotop/src/utils"
)

type NetWidget struct {
	*ui.Sparklines
	updateInterval time.Duration

	// used to calculate recent network activity
	prevRecvTotal uint64
	prevSentTotal uint64
}

func NewNetWidget(renderLock *sync.RWMutex) *NetWidget {
	recv := ui.NewSparkline()
	recv.Data = []int{}

	sent := ui.NewSparkline()
	sent.Data = []int{}

	spark := ui.NewSparklines(recv, sent)
	self := &NetWidget{
		Sparklines:     spark,
		updateInterval: time.Second,
	}
	self.Title = " Network Usage "

	self.update()

	go func() {
		for range time.NewTicker(self.updateInterval).C {
			renderLock.RLock()
			self.update()
			renderLock.RUnlock()
		}
	}()

	return self
}

func (self *NetWidget) update() {
	interfaces, err := psNet.IOCounters(true)
	if err != nil {
		log.Printf("failed to get network activity from gopsutil: %v", err)
		return
	}
	var curRecvTotal uint64
	var curSentTotal uint64
	for _, _interface := range interfaces {
		// ignore VPN interface
		if _interface.Name != "tun0" {
			curRecvTotal += _interface.BytesRecv
			curSentTotal += _interface.BytesSent
		}
	}
	var recvRecent uint64
	var sentRecent uint64

	if self.prevRecvTotal != 0 { // if this isn't the first update
		recvRecent = curRecvTotal - self.prevRecvTotal
		sentRecent = curSentTotal - self.prevSentTotal

		if int(recvRecent) < 0 {
			log.Printf("error: negative value for recently received network data from gopsutil. recvRecent: %v", recvRecent)
			// recover from error
			recvRecent = 0
		}
		if int(sentRecent) < 0 {
			log.Printf("error: negative value for recently sent network data from gopsutil. sentRecent: %v", sentRecent)
			// recover from error
			sentRecent = 0
		}

		self.Lines[0].Data = append(self.Lines[0].Data, int(recvRecent))
		self.Lines[1].Data = append(self.Lines[1].Data, int(sentRecent))
	}

	// used in later calls to update
	self.prevRecvTotal = curRecvTotal
	self.prevSentTotal = curSentTotal

	// render widget titles
	for i := 0; i < 2; i++ {
		total, label, recent := func() (uint64, string, uint64) {
			if i == 0 {
				return curRecvTotal, "RX", recvRecent
			}
			return curSentTotal, "Tx", sentRecent
		}()

		recentConv, unitRecent := utils.ConvertBytes(uint64(recent))
		totalConv, unitTotal := utils.ConvertBytes(uint64(total))

		self.Lines[i].Title1 = fmt.Sprintf(" Total %s: %5.1f %s", label, totalConv, unitTotal)
		self.Lines[i].Title2 = fmt.Sprintf(" %s/s: %9.1f %2s/s", label, recentConv, unitRecent)
	}
}
