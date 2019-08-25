package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jeffchannell/mineserver/game"
)

var (
	games map[uuid.UUID]*game.Game
)

func init() {
	games = make(map[uuid.UUID]*game.Game)
}

func main() {
	// favicon, for browsers
	http.HandleFunc(`/favicon.ico`, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, `static/favicon.ico`)
	})
	// no content in root
	http.HandleFunc(`/`, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	// handle /games routes
	http.HandleFunc(`/games/`, func(w http.ResponseWriter, r *http.Request) {
		// break up the path
		p := strings.Split(strings.TrimPrefix(r.URL.Path, "/games/"), "/")
		// switch by method first
		switch r.Method {
		case `GET`:
			switch p[0] {
			case "":
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			default:
				game, err := getGameByUUIDString(p[0])
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, err.Error())
					return
				}
				s := game.Pretty()
				fmt.Fprintf(w, s.String())
				return
			}
		case `POST`:
			switch p[0] {
			// empty path - create a new game
			case "":
				// generate a new uuid
				uuid, err := uuid.NewUUID()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, err.Error())
					return
				}
				// generate a new game
				game, err := game.NewGame(uuid, 12, 12, 20)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, err.Error())
					return
				}
				// store the game in memory
				games[uuid] = game
				// send the new game uuid back to the client
				w.Header().Set("X-Game-Uuid", uuid.String())
				w.WriteHeader(http.StatusCreated)
				return
			// update game by UUID
			default:
				// find the requested game
				game, err := getGameByUUIDString(p[0])
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, err.Error())
					return
				}
				// read the contents of POST
				err = r.ParseForm()
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, err.Error())
					return
				}
				// get the POSTed X value
				xString := r.Form.Get("x")
				if "" == xString {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				// convert X into uint
				xUint, err := strconv.ParseUint(xString, 10, 16)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, err.Error())
					return
				}
				x := uint16(xUint)
				// get the POSTed Y value
				yString := r.Form.Get("y")
				if "" == yString {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				// convert Y into uint
				yUint, err := strconv.ParseUint(yString, 10, 16)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, err.Error())
					return
				}
				y := uint16(yUint)
				// are we toggling flags?
				flag := "1" == r.Form.Get("flag")

				err = game.ClickTile(x, y, flag)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, err.Error())
					return
				}

				w.WriteHeader(http.StatusAccepted)

				s := game.Pretty()
				fmt.Fprintf(w, s.String())
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	// start webserver
	http.ListenAndServe(`:55555`, nil)
}

func getGameByUUIDString(uuidStr string) (g *game.Game, err error) {
	uuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return nil, err
	}
	if g, ok := games[uuid]; ok {
		return g, nil
	}
	return nil, errors.New("invalid Game")
}
