package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	CAC    byte = '1'
	Busch  byte = '2'
	Livi   byte = '3'
	CD     byte = '4'
	Online byte = 'O'
)

type schedule struct {
	Meetings []meeting `json:"meetings"`
	Sections []string  `json:"indexes"`
}

type meeting struct {
	Day      string `json:"day"`
	Location string `json:"location"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Name     string `json:"name"`
}

func dayN(day string) int {
	switch day {
	case "M":
		return 0
	case "T":
		return 1
	case "W":
		return 2
	case "H":
		return 3
	case "F":
		return 4
	}
	return -1
}

func campusColor(c string) tcell.Color {
	switch c[0] {
	case Busch:
		return tcell.ColorLightCyan
	case Livi:
		return tcell.ColorOrange
	case CAC:
		return tcell.ColorYellow
	case CD:
		return tcell.ColorGreen
	default:
		return tcell.ColorDefault
	}
}

func meetText(meet meeting) string {
	ts := func(ts int) (int, int, string) {
		eh := ts / 60
		em := ts % 60
		ep := "AM"
		if eh >= 12 {
			if eh != 12 {
				eh -= 12
			}
			ep = "PM"
		}
		return eh, em, ep
	}
	sh, sm, sp := ts(meet.Start)
	eh, em, ep := ts(meet.End)
	return fmt.Sprintf("[::b]%s[::B]\n%d:%02d%s - %d:%02d%s", meet.Name, sh, sm, sp, eh, em, ep)
}

type meetView struct {
	meeting
	*tview.TextView
}

type scheduleView struct {
	meets     []*meetView
	schedules []schedule
	current   int
	*tview.Box
}

func newScheduleView(scheds []schedule) *scheduleView {
	view := new(scheduleView)
	box := tview.NewBox()
	box.SetDrawFunc(view.draw)
	meets := scheds[0].Meetings
	meetv := make([]*meetView, 0, len(scheds))
	for _, meet := range meets {
		meetv = append(meetv, newMeetView(meet))
	}
	return &scheduleView{meetv, scheds, 0, box}
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func max(a, b int) int {
	if b > a {
		return b
	}
	return a
}

func (s *scheduleView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	now := -1
	switch key {
	case tcell.KeyLeft:
		now = min(0, s.current-1)
	case tcell.KeyRight:
		now = min(len(s.schedules)-1, s.current+1)
	default:
		return event
	}
	if now != -1 {
		s.meets = s.meets[:1]
		s.current = now
		meets := s.schedules[now].Meetings
		for _, meet := range meets {
			s.meets = append(s.meets, newMeetView(meet))
		}
	}
	return nil
}

func newMeetView(meet meeting) *meetView {
	tv := tview.NewTextView().SetDynamicColors(true).SetText(meetText(meet))
	tv.SetBorder(true)
	// tv.Box.SetBorderColor(campusColor(meet.Location))
	tv.Box.SetBorderColor(tcell.ColorBlack)
	tv.Box.SetBorderAttributes(tcell.AttrDim)
	// tv.Box.SetBorderAttributes(tcell.AttrBlink)
	tv.SetBackgroundColor(campusColor(meet.Location))
	tv.SetTextColor(tcell.ColorBlack)
	return &meetView{meet, tv}
}

func (sched *scheduleView) draw(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	dayw := width / 5
	// meet := scheds[0]
	mpc := (60 * 16) / height
	// minutes per cell
	for _, meet := range sched.meets {
		mx := x + dayw*dayN(meet.Day)
		my := y + (meet.Start-7*60)/mpc
		meet.SetRect(mx, my, dayw, 5)
		meet.Draw(screen)
	}
	return x, y, width, height
}

func main() {
	var schedules []schedule
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		var sched schedule
		if err := json.Unmarshal(s.Bytes(), &sched); err != nil {
			log.Fatalln(err)
		}
		schedules = append(schedules, sched)
	}
	sched := newScheduleView(schedules)
	box := tview.NewBox().
		SetBorderAttributes(tcell.AttrBold).
		SetDrawFunc(sched.draw).
		SetInputCapture(sched.handleInput)
	if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
		panic(err)
	}
}
