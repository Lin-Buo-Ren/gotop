package widgets

import (
	"fmt"
	"image"
	"sort"
	"sync"
	"time"

	ui "github.com/gizak/termui"

	"github.com/cjbassi/gotop/src/utils"
)

type TemperatureMode uint32

const (
	Celcius      TemperatureMode = 0
	Fahrenheight                 = 1
)

type TempWidget struct {
	*ui.Block      // inherits from Block instead of a premade Widget
	updateInterval time.Duration
	Data           map[string]int
	TempThreshold  int32
	TempLowColor   ui.Color
	TempHighColor  ui.Color
	TempMode       TemperatureMode
}

func NewTempWidget(renderLock *sync.RWMutex, tempMode TemperatureMode) *TempWidget {
	self := &TemperatureWidget{
		Block:          ui.NewBlock(),
		updateInterval: time.Second * 5,
		Data:           make(map[string]int),
		TempThreshold:  80, // temp at which color should change
		TempMode:       tempMode,
	}
	self.Title = " Temperatures "

	if tempMode == Fahrenheight {
		self.TempThreshold = utils.CelsiusToFahrenheit(self.TempThreshold)
	}

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

// We implement a custom Draw method instead of inheriting from a generic Widget.
func (self *TempWidget) Draw(buf *ui.Buffer) {
	self.Block.Draw(buf)

	var keys []string
	for key := range self.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for y, key := range keys {
		if y+1 > self.Inner.Dy() {
			break
		}

		fg := self.TempLowColor
		if self.Data[key] >= self.TempThreshold {
			fg = self.TempHighColor
		}

		s := ui.TrimString(key, (self.Inner.Dx() - 4))
		buf.SetString(s,
			ui.Theme.Default,
			image.Pt(self.Inner.Min.X, self.Inner.Min.Y+y),
		)
		if self.Fahrenheit {
			buf.SetString(
				fmt.Sprintf("%3dF", self.Data[key]),
				ui.NewStyle(fg),
				image.Pt(self.Inner.Max.X-4, self.Inner.Min.Y+y),
			)
		} else {
			buf.SetString(
				fmt.Sprintf("%3dC", self.Data[key]),
				ui.NewStyle(fg),
				image.Pt(self.Inner.Max.X-4, self.Inner.Min.Y+y),
			)
		}
	}
}
