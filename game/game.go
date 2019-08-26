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
	won         bool      // game was won
}

// NewGame starts a new game
func NewGame(uid uuid.UUID, w, h, m uint16) (g *Game, err error) {
	var maxW, maxH, maxM int
	maxW = 250
	maxH = 250
	maxM = int(w)*int(h) - 2
	if maxW < int(w) {
		return nil, errors.New("width exceeds max")
	}
	if maxH < int(h) {
		return nil, errors.New("height exceeds max")
	}
	if maxM < int(m) {
		return nil, errors.New("mines exceed tiles")
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
			g.End(true)
		}
	}
	return
}

// End the game
func (g *Game) End(won bool) {
	g.endTime = time.Now()
	g.won = true
}

// String writes the board state to a string
func (g *Game) String() string {
	var b bytes.Buffer
	// show the start time
	b.WriteString(fmt.Sprintf("Game Started: %v\n", g.startTime))
	// show the end time
	if !g.endTime.IsZero() {
		b.WriteString(fmt.Sprintf("Game Ended: %v\n", g.endTime))
		if g.won {
			b.WriteString("Game Ended with a WIN\n")
		}
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
	for i := 0; i < len(g.grid); i++ {
		b.WriteString(fmt.Sprintf("%d ", g.grid[i].value))
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
	obj["flags"] = g.flags
	if !g.endTime.IsZero() {
		obj["end"] = g.endTime
		if g.won {
			obj["won"] = true
			obj["flags"] = g.totalMines
		}
	}
	grid := make([]string, g.boardHeight*g.boardWidth)
	lost := !g.won && !g.endTime.IsZero()
	for i := 0; i < len(g.grid); i++ {
		var val string
		isMine := 9 == g.grid[i].value
		if lost && isMine && !g.grid[i].flagged {
			// expose non-flagged mines if the game is over and lost
			val = "9"
		} else if lost && !isMine && g.grid[i].flagged {
			// mark incorrect flags if the game is over and lost
			val = "X"
		} else if g.grid[i].flagged || (g.won && isMine) {
			// mark flags or mines if the game was won
			val = "!"
		} else if !g.grid[i].clicked {
			// mark unchecked tiles
			val = "?"
		} else if 0 == g.grid[i].value {
			// leave empty open tiles with no label
			val = ""
		} else {
			// label values 1-8
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
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			// tile index
			idx := w*y + x
			// skip mines
			if 9 == g.grid[idx].value {
				continue
			}
			// total mines around this tile
			g.grid[idx].value = g.countMines(uint16(x), uint16(y))
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
