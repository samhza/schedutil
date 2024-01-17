package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"strings"
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

type schedule []meeting

type meeting struct {
	day    rune
	campus rune
	start  int
	end    int
	name   string
}

type Game struct {
	schedules     []schedule
	sections      []string
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
		for _, meet := range g.schedules[g.current] {
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
		log.Printf(linkfmt, g.sections[g.current])
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		f, err := os.OpenFile("favs.txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("opening favs file: %s\n", err)
		}
		f.WriteString(g.fmt_schedule(g.current))
		f.Write([]byte{'\n'})
		f.Close()
	}
	left := ebiten.IsKeyPressed(ebiten.KeyLeft)
	if !(left || ebiten.IsKeyPressed(ebiten.KeyRight)) {
		g.down = time.Time{}
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.schedules = append(g.schedules[:g.current], g.schedules[g.current+1:]...)
		g.sections = append(g.sections[:g.current], g.sections[g.current+1:]...)
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

func dayToN(day rune) int {
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
	panic(string(day))
}

func campusColor(campus rune) color.RGBA {
	switch campus {
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
	height = float32(meet.end-meet.start) / mpx
	y = float32(meet.start-7*60) / mpx
	x = width * float32(dayToN(meet.day))
	return
}
func (g *Game) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, g.width, g.height, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
	sched := g.schedules[g.current]
	for _, meet := range sched {
		x, y, width, height := g.meetgeo(meet)
		vector.DrawFilledRect(screen, x, y, width, height, campusColor(meet.campus), false)
		sh := meet.start / 60
		sm := meet.start % 60
		sp := "AM"
		if sh >= 12 {
			if sh != 12 {
				sh -= 12
			}
			sp = "PM"
		}
		eh := meet.end / 60
		em := meet.end % 60
		ep := "AM"
		if eh >= 12 {
			if eh != 12 {
				eh -= 12
			}
			ep = "PM"
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s\n%d:%02d%s-%d:%02d%s", meet.name, sh, sm, sp, eh, em, ep), int(x), int(y))
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("%d : %s", g.current, g.sections[g.current]))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.width = float32(outsideWidth)
	g.height = float32(outsideHeight)
	return outsideWidth, outsideHeight
}

func fmt_meet(meet meeting) string {
	return fmt.Sprintf("%c%c%d,%d=%s",
		meet.day,
		meet.campus,
		meet.start,
		meet.end,
		meet.name)
}

//ggo:embed x.txt
//var xxx string

func (g *Game) fmt_schedule(i int) string {
	sched := g.schedules[i]
	sections := g.sections[i]
	var sb strings.Builder
	for i, meet := range sched {
		sb.WriteString(fmt_meet(meet))
		if i != len(sched)-1 {
			sb.WriteByte(':')
		}
	}
	sb.WriteByte('§')
	sb.WriteString(sections)
	return sb.String()
}

func main() {
	game := Game{}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln(err)
	}
	inputs := string(input)
	// inputs := xxx
	for _, line := range strings.Split(inputs, "\n") {
		meets, sections, _ := strings.Cut(line, "§")
		splat := strings.Split(meets, ":")
		sched := make(schedule, len(splat))
		valid := true
		for i, meet := range splat {
			var (
				weekday, campus rune
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
				day:    weekday,
				campus: campus,
				start:  start,
				end:    end,
				name:   name,
			}
			sched[i] = m
		}
		if !valid {
			continue
		}
		game.sections = append(game.sections, sections)
		game.schedules = append(game.schedules, sched)
	}
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Schedules")
	if err := ebiten.RunGame(&game); err != nil {
		log.Fatal(err)
	}
}
