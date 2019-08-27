package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jeffchannell/mineserver/mines"
)

var (
	games map[uuid.UUID]*mines.Game
)

func init() {
	games = make(map[uuid.UUID]*mines.Game)
}

func logRequest(r *http.Request) {
	log.Printf("%s %s\n", r.Method, r.URL.Path)
}

func main() {
	// favicon, for browsers
	http.HandleFunc(`/favicon.ico`, func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		http.ServeFile(w, r, `static/favicon.ico`)
	})
	// no content in root
	http.HandleFunc(`/`, func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		w.WriteHeader(http.StatusNoContent)
	})
	// handle /games routes
	http.HandleFunc(`/games/`, func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		// add cors headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Origin, X-GAME-UUID")
		w.Header().Set("Access-Control-Max-Age", "86400")
		// break up the path
		p := strings.Split(strings.TrimPrefix(r.URL.Path, "/games/"), "/")
		// switch by method first
		switch r.Method {
		case `OPTIONS`:
			switch p[0] {
			case "":
				w.WriteHeader(http.StatusNoContent)
			default:
				_, err := getGameByUUIDString(p[0])
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, err.Error())
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
			return
		case `DELETE`:
			switch p[0] {
			case "":
				w.WriteHeader(http.StatusMethodNotAllowed)
			default:
				game, err := getGameByUUIDString(p[0])
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, err.Error())
					return
				}
				game.End(false)
				w.WriteHeader(http.StatusNoContent)
			}
			return
		case `GET`:
			switch p[0] {
			case "":
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"games":%d}`, len(games))
				return
			default:
				game, err := getGameByUUIDString(p[0])
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, err.Error())
					return
				}
				var state string
				if 1 < len(p) {
					state, err = game.Turn(p[1])
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprintf(w, err.Error())
						return
					}
				} else {
					state, err = game.JSON()
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintf(w, err.Error())
						return
					}
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, state)
				return
			}
		case `POST`:
			switch p[0] {
			// empty path - create a new game
			case "":
				// read the contents of POST
				err := r.ParseForm()
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, err.Error())
					return
				}
				width, err := strconv.ParseUint(r.Form.Get("w"), 10, 16)
				if err != nil {
					width = 12
				}
				height, err := strconv.ParseUint(r.Form.Get("h"), 10, 16)
				if err != nil {
					height = 12
				}
				minecount, err := strconv.ParseUint(r.Form.Get("m"), 10, 16)
				if err != nil {
					minecount = 20
				}
				// generate a new game
				game, err := mines.NewGame(uint16(width), uint16(height), uint16(minecount))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, err.Error())
					return
				}
				uid := game.UUID()
				// store the game in memory
				games[uid] = game
				// send the new game uuid back to the client
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				fmt.Fprintf(w, fmt.Sprintf(`{"uuid":"%s"}`, uid.String()))
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
				s, err := game.JSON()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, err.Error())
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				fmt.Fprintf(w, s)
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	// start webserver
	http.ListenAndServe(`:55555`, nil)
}

func getGameByUUIDString(uuidStr string) (g *mines.Game, err error) {
	uid, err := uuid.Parse(uuidStr)
	if err != nil {
		return nil, err
	}
	if g, ok := games[uid]; ok {
		return g, nil
	}
	return nil, errors.New("invalid Game")
}
