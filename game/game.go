package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type tile struct {
	value   uint8
	flagged bool
	clicked bool
}

// Game represents a single mines game being played
type Game struct {
	uid         uuid.UUID // game uuid
	totalMines  uint16    // number of mines that should be on the board
	boardWidth  uint16    // width, in tiles
	boardHeight uint16    // height, in tiles
	flags       uint16    // how many flags are set
	grid        []tile    // master grid
	startTime   time.Time // time game started
	endTime     time.Time // time game ended
}

// NewGame starts a new game
func NewGame(uid uuid.UUID, w, h, m uint16) (g *Game, err error) {
	max := w * h
	if (1 + max) < m {
		err = errors.New(`mines exceed tiles`)
		return nil, err
	}
	g = &Game{
		uid:         uid,
		boardHeight: h,
		boardWidth:  w,
		totalMines:  m,
		startTime:   time.Now(),
	}

	return g, nil
}

// ClickTile activates a tile
func (g *Game) ClickTile(x, y uint16, flag bool) (err error) {
	// validate x
	if g.boardWidth <= x {
		return errors.New("X cannot be larger than the board width")
	}
	// validate y
	if g.boardHeight <= y {
		return errors.New("Y cannot be larger than the board height")
	}
	// add mines, if not already added
	if 0 == len(g.grid) {
		g.addMinesToGrid(x, y)
	}
	// bail if game was lost
	if !g.endTime.IsZero() {
		return errors.New("Game is not active")
	}
	// get tile
	tile := &g.grid[g.boardWidth*y+x]

	if tile.clicked { // tile is already clicked
		if !flag { // not toggling flags, click neighbors
			if f := g.countFlags(x, y); f == tile.value {
				g.clickNeighbors(x, y)
			}
		}
	} else if !tile.flagged { // tile is not flagged
		if flag { // toggle flag
			tile.flagged = true
			g.flags++
		} else { // click tile
			tile.clicked = true
			if 9 == tile.value { // tile is a mine - game over!
				g.endTime = time.Now()
			} else if 0 == tile.value { // tile has 0 neighboring mines - open neighbors too
				g.clickNeighbors(x, y)
			}
		}
	} else if tile.flagged && flag { // tile is flagged and we are turning off the flag
		tile.flagged = false
		g.flags--
	}
	// check win condition
	if g.endTime.IsZero() {
		var total int
		for i := 0; i < len(g.grid); i++ {
			if g.grid[i].clicked || (9 == g.grid[i].value) {
				total++
			}
		}
		if total == len(g.grid) {
			g.endTime = time.Now()
		}
	}
	return
}

// String writes the board state to a string
func (g *Game) String() string {
	var b bytes.Buffer
	// show the start time
	b.WriteString(fmt.Sprintf("Game Started: %v\n", g.startTime))
	// show the end time
	if !g.endTime.IsZero() {
		b.WriteString(fmt.Sprintf("Game Ended: %v\n", g.endTime))
	}
	// no turns taken yet
	if 0 == len(g.grid) {
		return b.String()
	}
	// start the table with some padding
	b.WriteString("  ")
	// init
	var r int
	w := int(g.boardWidth)
	h := int(g.boardHeight)
	// write out the table header
	for i := 0; i < w; i++ {
		if 10 > i {
			b.WriteString(" ")
		}
		b.WriteString(fmt.Sprintf("%d", i))
	}
	// start the first row number
	b.WriteString("\n 0 ")
	// loop the rows
	for i := 0; i < len(g.grid); i++ {
		if g.grid[i].flagged {
			b.WriteString("!")
		} else if !g.grid[i].clicked {
			b.WriteString("?")
		} else if 0 == g.grid[i].value {
			b.WriteString(".")
		} else {
			b.WriteString(fmt.Sprintf("%d", g.grid[i].value))
		}
		b.WriteString(" ") // spacer
		// add newline and next row number
		if 0 == ((i + 1) % w) {
			b.WriteString("\n")
			r++
			if r < h {
				if 10 > r {
					b.WriteString(" ")
				}
				b.WriteString(fmt.Sprintf("%d ", r))
			}
		}
	}
	return b.String()
}

// JSON writes the board state to a JSON string
func (g *Game) JSON() string {
	obj := make(map[string]interface{})
	obj["start"] = g.startTime
	obj["mines"] = g.totalMines
	obj["height"] = g.boardHeight
	obj["width"] = g.boardWidth
	if !g.endTime.IsZero() {
		obj["end"] = g.endTime
	}
	grid := make([]string, g.boardHeight*g.boardWidth)
	for i := 0; i < len(g.grid); i++ {
		var val string
		if g.grid[i].flagged {
			val = "!"
		} else if !g.grid[i].clicked {
			val = "?"
		} else if 0 == g.grid[i].value {
			val = ""
		} else {
			val = fmt.Sprintf("%d", g.grid[i].value)
		}
		grid[i] = val
	}
	obj["grid"] = grid
	json, err := json.Marshal(obj)
	if err != nil {
		return err.Error()
	}
	return string(json)
}

// addMinesToGrid generates the initial mine grid
func (g *Game) addMinesToGrid(ignoreX, ignoreY uint16) {
	g.grid = make([]tile, g.boardHeight*g.boardWidth)
	var mines uint16
	for mines < g.totalMines {
		x := uint16(rand.Intn(int(g.boardWidth)))
		y := uint16(rand.Intn(int(g.boardHeight)))
		if x == ignoreX && y == ignoreY {
			continue
		}
		if set := g.setGridValue(9, x, y); set {
			mines++
		}
	}
	g.numberGrid()
}

// clickNeighbors allows an event to spread across a field of tiles
func (g *Game) clickNeighbors(x, y uint16) {
	var h, w int
	h = int(g.boardHeight)
	w = int(g.boardWidth)
	for j := -1; j < 2; j++ {
		for i := -1; i < 2; i++ {
			// skip 0,0
			if 0 == i && 0 == j {
				continue
			}
			// get new x,y coords
			y2 := int(y) + j
			x2 := int(x) + i
			// skip out of bounds coords
			if 0 > x2 || 0 > y2 || x2 >= w || y2 >= h {
				continue
			}
			// skip neighbors that are clicked
			tile := &g.grid[g.boardWidth*uint16(y2)+uint16(x2)]
			if !tile.clicked && !tile.flagged {
				g.ClickTile(uint16(x2), uint16(y2), false)
			}
		}
	}
}

// countFlags around a tile
func (g *Game) countFlags(x, y uint16) (total uint8) {
	var h, w int
	h = int(g.boardHeight)
	w = int(g.boardWidth)
	for j := -1; j < 2; j++ {
		for i := -1; i < 2; i++ {
			// skip 0,0
			if 0 == i && 0 == j {
				continue
			}
			// get new x,y coords
			y2 := int(y) + j
			x2 := int(x) + i
			// skip out of bounds coords
			if 0 > x2 || 0 > y2 || x2 >= w || y2 >= h {
				continue
			}
			if g.grid[w*y2+x2].flagged {
				total++
			}
		}
	}
	return total
}

// countMines around a tile
func (g *Game) countMines(x, y uint16) (total uint8) {
	var h, w int
	h = int(g.boardHeight)
	w = int(g.boardWidth)
	for j := -1; j < 2; j++ {
		for i := -1; i < 2; i++ {
			// skip 0,0
			if 0 == i && 0 == j {
				continue
			}
			// get new x,y coords
			y2 := int(y) + j
			x2 := int(x) + i
			// skip out of bounds coords
			if 0 > x2 || 0 > y2 || x2 >= w || y2 >= h {
				continue
			}
			if 9 == g.grid[w*y2+x2].value {
				total++
			}
		}
	}
	return total
}

// numberGrid sets grid values from 1-8
func (g *Game) numberGrid() {
	var h, w int
	h = int(g.boardHeight)
	w = int(g.boardWidth)

	// loop through every tile
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// skip mines
			if 9 == g.grid[w*y+x].value {
				continue
			}
			// total mines around this tile
			var total uint8
			for j := -1; j < 2; j++ {
				for i := -1; i < 2; i++ {
					// skip 0,0
					if 0 == i && 0 == j {
						continue
					}
					// get new x,y coords
					y2 := y + j
					x2 := x + i
					// skip out of bounds coords
					if 0 > x2 || 0 > y2 || x2 >= w || y2 >= h {
						continue
					}
					if 9 == g.grid[w*y2+x2].value {
						total++
					}
				}
			}
			g.grid[h*y+x].value = total
		}
	}
}

// setGridValue returns true if the new value does not equal the old value
func (g *Game) setGridValue(value uint8, x, y uint16) (set bool) {
	w := g.boardWidth
	if g.grid[w*y+x].value != value {
		g.grid[w*y+x].value = value
		set = true
	}
	return set
}
