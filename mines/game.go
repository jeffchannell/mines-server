package mines

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// tile in the game
type tile struct {
	value   uint8
	flagged bool
	clicked bool
}

// turn taken in the game
type turn struct {
	uid     uuid.UUID // turn uuid
	x       uint16    // click x coordinate
	y       uint16    // click y coordinate
	flag    bool      // tile flagging was enabled
	takenAt time.Time // time turn was taken
	tiles   []tile    // game tiles
}

// newTurn for the game
func newTurn(x uint16, y uint16, f bool) (t *turn, err error) {
	uid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	t = &turn{
		uid:     uid,
		x:       x,
		y:       y,
		flag:    f,
		takenAt: time.Now(),
	}
	return t, nil
}

// Game represents a single mines game being played
type Game struct {
	uid       uuid.UUID    // game uuid
	width     uint16       // width, in tiles
	height    uint16       // height, in tiles
	mines     uint16       // number of mines that should be on the board
	flags     uint16       // how many flags are set
	startedAt time.Time    // time game started
	endedAt   time.Time    // time game ended
	grid      []tile       // master grid
	won       bool         // game was won
	history   map[int]turn // game history
}

// NewGame starts a new game
func NewGame(w, h, m uint16) (g *Game, err error) {
	var maxW, maxH, maxM int
	maxW = 250
	maxH = 250
	maxM = int(w)*int(h) - 2
	uid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
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
		uid:       uid,
		height:    h,
		width:     w,
		mines:     m,
		startedAt: time.Now(),
	}
	g.history = make(map[int]turn)

	return g, nil
}

// ClickTile activates a tile
func (g *Game) ClickTile(x, y uint16, flag bool) (err error) {
	// validate x
	if g.width <= x {
		return errors.New("X cannot be larger than the board width")
	}
	// validate y
	if g.height <= y {
		return errors.New("Y cannot be larger than the board height")
	}
	// generate turn object
	turn, err := newTurn(x, y, flag)
	if err != nil {
		return err
	}
	// add tiles to turn and add turn to history stack
	var tiles []tile
	if 0 == len(g.history) { // add first turn
		tiles = g.generateTiles(x, y)
	} else { // add subsequent turn
		tiles = make([]tile, g.height*g.width)
		copy(tiles, g.history[len(g.history)-1].tiles)
	}
	turn.tiles = tiles
	g.history[len(g.history)] = *turn

	// bail if game was lost
	if !g.endedAt.IsZero() {
		return errors.New("Game is not active")
	}
	// get tile
	tile := &turn.tiles[g.width*y+x]

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
				g.endedAt = time.Now()
			} else if 0 == tile.value { // tile has 0 neighboring mines - open neighbors too
				g.clickNeighbors(x, y)
			}
		}
	} else if tile.flagged && flag { // tile is flagged and we are turning off the flag
		tile.flagged = false
		g.flags--
	}
	// check win condition
	if g.endedAt.IsZero() {
		var total int
		for i := 0; i < len(turn.tiles); i++ {
			if turn.tiles[i].clicked || (9 == turn.tiles[i].value) {
				total++
			}
		}
		if total == len(turn.tiles) {
			g.End(true)
		}
	}
	return
}

// End the game
func (g *Game) End(won bool) {
	g.endedAt = time.Now()
	g.won = true
}

// JSON writes the board state to a JSON string
func (g *Game) JSON() string {
	obj := make(map[string]interface{})
	obj["started_at"] = g.startedAt
	obj["mines"] = g.mines
	obj["height"] = g.height
	obj["width"] = g.width
	obj["flags"] = g.flags
	if !g.endedAt.IsZero() {
		obj["ended_at"] = g.endedAt
		if g.won {
			obj["won"] = true
			obj["flags"] = g.mines
		}
	}
	grid := make([]string, g.height*g.width)
	lost := !g.won && !g.endedAt.IsZero()
	turn := g.history[len(g.history)-1]
	for i := 0; i < len(turn.tiles); i++ {
		var val string
		isMine := 9 == turn.tiles[i].value
		if lost && isMine && !turn.tiles[i].flagged {
			// expose non-flagged mines if the game is over and lost
			val = "9"
		} else if lost && !isMine && turn.tiles[i].flagged {
			// mark incorrect flags if the game is over and lost
			val = "X"
		} else if turn.tiles[i].flagged || (g.won && isMine) {
			// mark flags or mines if the game was won
			val = "!"
		} else if !turn.tiles[i].clicked {
			// mark unchecked tiles
			val = "?"
		} else if 0 == turn.tiles[i].value {
			// leave empty open tiles with no label
			val = ""
		} else {
			// label values 1-8
			val = fmt.Sprintf("%d", turn.tiles[i].value)
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

// UUID of this game
func (g *Game) UUID() uuid.UUID {
	return g.uid
}

func (g *Game) generateTiles(ignoreX, ignoreY uint16) []tile {
	var mines uint16
	tiles := make([]tile, g.height*g.width)
	for mines < g.mines {
		x := uint16(rand.Intn(int(g.width)))
		y := uint16(rand.Intn(int(g.height)))
		if x == ignoreX && y == ignoreY {
			continue
		}
		if tiles[g.width*y+x].value != 9 {
			tiles[g.width*y+x].value = 9
			mines++
		}
	}
	var h, w int
	h = int(g.height)
	w = int(g.width)

	// loop through every tile
	for tileX := 0; tileX < w; tileX++ {
		for tileY := 0; tileY < h; tileY++ {
			// tile index
			idx := w*tileY + tileX
			// skip mines
			if 9 == tiles[idx].value {
				continue
			}
			// total mines around this tile
			for j := -1; j < 2; j++ {
				for i := -1; i < 2; i++ {
					// skip 0,0
					if 0 == i && 0 == j {
						continue
					}
					// get new x,y coords
					y2 := int(tileY) + j
					x2 := int(tileX) + i
					// skip out of bounds coords
					if 0 > x2 || 0 > y2 || x2 >= w || y2 >= h {
						continue
					}
					if 9 == tiles[w*y2+x2].value {
						tiles[idx].value++
					}
				}
			}
		}
	}

	return tiles
}

// clickNeighbors allows an event to spread across a field of tiles
func (g *Game) clickNeighbors(x, y uint16) {
	var h, w int
	h = int(g.height)
	w = int(g.width)
	tiles := g.history[len(g.history)-1].tiles
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
			tile := tiles[g.width*uint16(y2)+uint16(x2)]
			if !tile.clicked && !tile.flagged {
				g.ClickTile(uint16(x2), uint16(y2), false)
			}
		}
	}
}

// countFlags around a tile
func (g *Game) countFlags(x, y uint16) (total uint8) {
	var h, w int
	h = int(g.height)
	w = int(g.width)
	tiles := g.history[len(g.history)-1].tiles
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
			if tiles[w*y2+x2].flagged {
				total++
			}
		}
	}
	return total
}
