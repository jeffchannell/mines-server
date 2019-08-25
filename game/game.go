package game

import (
	"bytes"
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
	mines       uint16    // number of mines currently on the board
	boardWidth  uint16    // width, in tiles
	boardHeight uint16    // height, in tiles
	flags       uint16    // how many flags are set
	active      bool      // minefield is active
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
	if g.mines != g.totalMines {
		fmt.Println("Adding mines")
		g.addMinesToGrid(x, y)
		g.active = true
	}
	// bail if game is not active
	if !g.active {
		return errors.New("Game is not active")
	}
	// get tile
	tile := &g.grid[g.boardWidth*y+x]

	if tile.clicked {
		// tile is already clicked
		fmt.Println("Tile is already clicked")
		if !flag {
			// not toggling flags, click neighbors
			fmt.Println("Tile is not flagged")
			if f := g.countFlags(x, y); f == tile.value {
				fmt.Println("Opening neighbors")
				g.clickNeighbors(x, y)
			}
		}
	} else if !tile.flagged {
		fmt.Println("Tile is not flagged")
		// tile is not flagged
		if flag {
			fmt.Println("Flagging tile")
			// toggle flag
			tile.flagged = true
			g.flags++
		} else {
			fmt.Println("Clicking tile")
			// click tile
			tile.clicked = true
			if 9 == tile.value {
				// tile is a mine - game over!
				fmt.Println("GAME OVER")
				g.active = false
			} else if 0 == tile.value {
				fmt.Println("Empty, opening neighbors")
				// tile has 0 neighboring mines - open neighbors too
				g.clickNeighbors(x, y)
			} else {
				fmt.Println("Tile clicked")
			}
		}
	} else if tile.flagged && flag {
		fmt.Println("Removing flag")
		// tile is flagged and we are turning off the flag
		tile.flagged = false
		g.flags--
	}
	// check win condition
	if g.active {
		var total int
		for i := 0; i < len(g.grid); i++ {
			if g.grid[i].clicked || (9 == g.grid[i].value) {
				total++
			}
		}
		if total == len(g.grid) {
			fmt.Println("YOU WIN")
			g.active = false
			g.endTime = time.Now()
		}
	}
	return
}

// Pretty print the board to a bytes buffer
func (g *Game) Pretty() bytes.Buffer {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("Game Started: %v\n", g.startTime))
	if !g.endTime.IsZero() {
		b.WriteString(fmt.Sprintf("Game Ended: %v\n", g.endTime))
	}
	if 0 == len(g.grid) {
		return b
	}
	b.WriteString("  ")
	w := int(g.boardWidth)
	h := int(g.boardHeight)
	for i := 0; i < w; i++ {
		if 10 > i {
			b.WriteString(" ")
		}
		b.WriteString(fmt.Sprintf("%d", i))
	}
	var r int
	b.WriteString("\n 0 ")
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
		b.WriteString(" ")
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
	return b
}

func (g *Game) addMinesToGrid(ignoreX, ignoreY uint16) {
	g.grid = make([]tile, g.boardHeight*g.boardWidth)
	g.mines = 0
	for g.mines < g.totalMines {
		x := uint16(rand.Intn(int(g.boardWidth)))
		y := uint16(rand.Intn(int(g.boardHeight)))
		if x == ignoreX && y == ignoreY {
			continue
		}
		if set := g.setGridValue(9, x, y); set {
			g.mines++
		}
	}
	g.numberGrid()
}

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

func (g *Game) setGridValue(value uint8, x, y uint16) (set bool) {
	w := g.boardWidth
	if g.grid[w*y+x].value != value {
		g.grid[w*y+x].value = value
		set = true
	}
	return set
}
