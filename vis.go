package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
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

type Game struct {
	schedules     []schedule
	current       int
	down          time.Time
	height, width float32
}

const linkfmt = "https://sims.rutgers.edu/webreg/editSchedule.htm?login=cas&semesterSelection=12024&indexList=%s\n"

func (g *Game) Update() error {
	leftclick := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	if leftclick || inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		ix, iy := ebiten.CursorPosition()
		fx, fy := float32(ix), float32(iy)
		for _, meet := range g.schedules[g.current].Meetings {
			x, y, width, height := g.meetgeo(meet)
			text := "like"
			if !leftclick {
				text = "dislike"
			}
			if fx >= x && fy >= y && fx <= x+width && fy <= y+height {
				fmt.Println(text, fmt_meet(meet))
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		log.Printf(linkfmt, g.schedules[g.current].Sections[0])
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		f, err := os.OpenFile("favs.txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("opening favs file: %s\n", err)
		}
		json.NewEncoder(f).Encode(g.schedules[g.current])
		f.Close()
	}
	left := ebiten.IsKeyPressed(ebiten.KeyLeft)
	if !(left || ebiten.IsKeyPressed(ebiten.KeyRight)) {
		g.down = time.Time{}
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.schedules = append(g.schedules[:g.current], g.schedules[g.current+1:]...)
		g.current -= 1
		if g.current < -1 {
			g.current = 0
		}
	}
	do := func() {
		n := 1
		if left {
			n = -1
		}
		g.current = g.current + n
		if g.current > len(g.schedules)-1 {
			g.current = len(g.schedules) - 1
		} else if g.current < 0 {
			g.current = 0
		}
	}
	if g.down.IsZero() {
		do()
		g.down = time.Now()
	} else if time.Since(g.down) > 500*time.Millisecond {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			do()
			return nil
		}
		do()
		g.down = time.Now().Add(-460 * time.Millisecond)
	}
	return nil
}

func dayToN(day string) int {
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
	panic(day)
}

func campusColor(campus string) color.RGBA {
	switch campus[0] {
	case '1':
		return color.RGBA{0xff, 0xff, 0xcc, 0xff}
	case '2':
		return color.RGBA{0xcc, 0xee, 0xff, 0xff}
	case '3':
		return color.RGBA{0xFF, 0xCC, 0x99, 0xff}
	case '4':
		return color.RGBA{0xDD, 0xFF, 0xDD, 0xff}
	case 'O':
		return color.RGBA{0xFF, 0x80, 0x80, 0xff}
	default:
		return color.RGBA{0x00, 0x00, 0x00, 0xff}
	}
}

func (g *Game) meetgeo(meet meeting) (x, y, width, height float32) {
	var mpx float32 = (16 * 60) / g.height
	width = g.width / float32(5)
	height = float32(meet.End-meet.Start) / mpx
	y = float32(meet.Start-7*60) / mpx
	x = width * float32(dayToN(meet.Day))
	return
}

func (g *Game) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, g.width, g.height, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
	sched := g.schedules[g.current]
	for _, meet := range sched.Meetings {
		x, y, width, height := g.meetgeo(meet)
		vector.DrawFilledRect(screen, x, y, width, height, campusColor(meet.Location), false)
		sh := meet.Start / 60
		sm := meet.Start % 60
		sp := "AM"
		if sh >= 12 {
			if sh != 12 {
				sh -= 12
			}
			sp = "PM"
		}
		eh := meet.End / 60
		em := meet.End % 60
		ep := "AM"
		if eh >= 12 {
			if eh != 12 {
				eh -= 12
			}
			ep = "PM"
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s\n%d:%02d%s-%d:%02d%s", meet.Name, sh, sm, sp, eh, em, ep), int(x), int(y))
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("%d : %s", g.current, g.schedules[g.current].Sections[0]))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.width = float32(outsideWidth)
	g.height = float32(outsideHeight)
	return outsideWidth, outsideHeight
}

func fmt_meet(meet meeting) string {
	return fmt.Sprintf("%s%s%d,%d=%s",
		meet.Day,
		meet.Location,
		meet.Start,
		meet.End,
		meet.Name)
}

func main() {
	game := Game{}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var sched schedule
		if err := json.Unmarshal(scanner.Bytes(), &sched); err != nil {
			log.Printf("Error parsing JSON: %s", err)
			continue
		}
		game.schedules = append(game.schedules, sched)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Schedules")
	if err := ebiten.RunGame(&game); err != nil {
		log.Fatal(err)
	}
}
