package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

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

type meeting struct {
	Day, Location byte
	Start, End    int
	Name          string
}

var scheds = []meeting{
	{Day: 'T', Location: '3', Start: 840, End: 920, Name: "INTRO TO PHILOSOPHY"},
	{Day: 'H', Location: '3', Start: 840, End: 920, Name: "INTRO TO PHILOSOPHY"},
	{Day: 'M', Location: '1', Start: 1060, End: 1140, Name: "INTRO TO ETHICS"},
	{Day: 'W', Location: '1', Start: 1060, End: 1140, Name: "INTRO TO ETHICS"},
	{Day: 'T', Location: '4', Start: 620, End: 700, Name: "INTRO TO MUSIC I"},
	{Day: 'H', Location: '4', Start: 620, End: 700, Name: "INTRO TO MUSIC I"},
	{Day: 'M', Location: '2', Start: 730, End: 810, Name: "METH INQUIRY ENGRS"},
	{Day: 'M', Location: '2', Start: 840, End: 920, Name: "COLLEGE WRITING"},
	{Day: 'W', Location: '2', Start: 840, End: 920, Name: "COLLEGE WRITING"},
}

func dayN(day byte) int {
	switch day {
	case 'M':
		return 0
	case 'T':
		return 1
	case 'W':
		return 2
	case 'H':
		return 3
	case 'F':
		return 4
	}
	return -1
}

func campusColor(c byte) tcell.Color {
	switch c {
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
	return fmt.Sprintf("%s\n%d:%02d%s - %d:%02d%s", meet.Name, sh, sm, sp, eh, em, ep)
}

type meetView struct {
	meeting
	*tview.TextView
}

type scheduleView struct {
	meets []*meetView
	*tview.Box
}

func newScheduleView(meets []meeting) *scheduleView {
	sched := new(scheduleView)
	box := tview.NewBox()
	box.SetDrawFunc(sched.draw)
	meetv := make([]*meetView, 0, len(meets))
	for _, meet := range meets {
		meetv = append(meetv, newMeetView(meet))
	}
	return &scheduleView{meetv, box}
}

func newMeetView(meet meeting) *meetView {
	tv := tview.NewTextView().SetText("[::b]" + meetText(meet))
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
	sched := newScheduleView(scheds)
	box := tview.NewBox().
		SetBorderAttributes(tcell.AttrBold).
		SetDrawFunc(sched.draw)
	// box := newScheduleView(scheds)
	if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
		panic(err)
	}
	abc
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln(err)
	}
	inputs := string(input)
	// inputs := xxx
	var schedules [][]meeting
	for _, line := range strings.Split(inputs, "\n") {
		meets, _, _ := strings.Cut(line, "§")
		splat := strings.Split(meets, "☐")
		sched := make([]meeting, len(splat))
		valid := true
		for i, meet := range splat {
			var (
				weekday, campus byte
				start, end      int
				name            string
			)
			_, err := fmt.Sscanf(meet, "%c%c%d,%d=", &weekday, &campus, &start, &end)
			if err != nil {
				log.Printf("Error scanning %s: %s", meet, err)
				valid = false
				break
			}
			if weekday == '-' {
				valid = false
				break
			}
			if end == -1 {
				valid = false
				break
			}
			_, name, _ = strings.Cut(meet, "=")
			m := meeting{
				Day:      weekday,
				Location: campus,
				Start:    start,
				End:      end,
				Name:     name,
			}
			sched[i] = m
		}
		if !valid {
			continue
		}
		schedules = append(schedules, sched)
	}
}
