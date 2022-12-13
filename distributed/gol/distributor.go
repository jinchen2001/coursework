package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

const alive = 255
const dead = 0

// Get the address of the server
var server = flag.String("server", "52.87.163.47:8030", "IP:PORT")

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	// Create a 2D slice to store the world.
	c.ioCommand <- ioInput
	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)
	c.ioFilename <- filename

	world := make([][]uint8, p.ImageHeight)
	for i := range world {
		world[i] = make([]uint8, p.ImageWidth)
	}
	for h := 0; h < p.ImageHeight; h++ {
		for w := 0; w < p.ImageWidth; w++ {
			world[h][w] = <-c.ioInput
		}
	}

	displayCurrentWorld(world, 0, c, p)

	flag.Parse()
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()

	// used to exit keyPressesCapture
	keyPressesCaptureDone := make(chan bool)
	// Capture keyPresses
	go keyPressesCapture(client, filename, c, p, keyPresses, keyPressesCaptureDone)

	// Used to exit AliveNumberCapture
	aliveNumberCaptureDone := make(chan bool)
	go AliveNumberCapture(client, c, p, aliveNumberCaptureDone)

	// Execute all turns of the Game of Life.
	response := makeRPC(client, world, p, stubs.RunGOL)

	// Report the final state using FinalTurnCompleteEvent.
	keyPressesCaptureDone <- true
	aliveNumberCaptureDone <- true

	displayCurrentWorld(response.World, response.Turn, c, p)

	newFilename := filename + "x" + strconv.Itoa(response.Turn)
	// Save current world
	saveCurrentWorld(newFilename, response.World, response.Turn, c, p)
	c.events <- FinalTurnComplete{response.Turn, getAllAliveCells(p, response.World)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{response.Turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

// goroutine for capturing keyPresses
func keyPressesCapture(client *rpc.Client, filename string, c distributorChannels, p Params, keyPresses <-chan rune, done <-chan bool) {
	isPause := false
	for true {
		select {
		case <-done:
			return
		case curKey := <-keyPresses:
			switch curKey {
			case 's':
				curWorldResponse := makeRPC(client, nil, p, stubs.CurWorld)
				newFilename := filename + "x" + strconv.Itoa(curWorldResponse.Turn)
				// Save current world
				saveCurrentWorld(newFilename, curWorldResponse.World, curWorldResponse.Turn, c, p)

			case 'p':
				if isPause {
					makeRPC(client, nil, p, stubs.ContinueCurGOL)
					fmt.Println("Continuing")
				} else {
					curResponse := makeRPC(client, nil, p, stubs.PauseCurGOL)
					fmt.Printf("Paused at turn = %d\n", curResponse.Turn)
				}
				isPause = !isPause

			case 'q':
				endGameOfLifeResponse := makeRPC(client, nil, p, stubs.ExitCurGOL)
				newFilename := filename + "x" + strconv.Itoa(endGameOfLifeResponse.Turn)
				// Save current world
				saveCurrentWorld(newFilename, endGameOfLifeResponse.World, endGameOfLifeResponse.Turn, c, p)
				c.events <- FinalTurnComplete{endGameOfLifeResponse.Turn, getAllAliveCells(p, endGameOfLifeResponse.World)}

			case 'k':
				makeRPC(client, nil, p, stubs.QuitAll)
			}
		}
	}
}

// goroutine for reporting the number of alive cells every 2 seconds
func AliveNumberCapture(client *rpc.Client, c distributorChannels, p Params, done <-chan bool) {
	ticker := time.NewTicker(time.Second << 1)
	for true {
		select {
		case <-done:
			return
		case <-ticker.C:
			aliveNumResponse := makeRPC(client, nil, p, stubs.CurAliveNum)
			c.events <- AliveCellsCount{aliveNumResponse.Turn, aliveNumResponse.AliveCells}
		}
	}
}

// Call a RPC
func makeRPC(client *rpc.Client, world [][]uint8, p Params, whichRPC string) *stubs.RPCResponse {
	request := stubs.RPCRequest{
		World:       world,
		Turns:       p.Turns,
		ImageWidth:  p.ImageWidth,
		ImageHeight: p.ImageHeight,
	}

	response := new(stubs.RPCResponse)

	client.Call(whichRPC, request, response)

	return response
}

// Get all the alive cells
func getAllAliveCells(p Params, world [][]uint8) []util.Cell {
	var totalAlive []util.Cell
	for h := 0; h < p.ImageHeight; h++ {
		for w := 0; w < p.ImageWidth; w++ {
			if world[h][w] == alive {
				totalAlive = append(totalAlive, util.Cell{X: w, Y: h})
			}
		}
	}
	return totalAlive
}

// Save the current world
func saveCurrentWorld(filename string, world [][]uint8, turn int, c distributorChannels, p Params) {
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for h := 0; h < p.ImageHeight; h++ {
		for w := 0; w < p.ImageWidth; w++ {
			c.ioOutput <- world[h][w]
		}
	}
	// Send ImageOutputComplete to SDL
	c.events <- ImageOutputComplete{turn, filename}
}

// Display current world
func displayCurrentWorld(world [][]uint8, turn int, c distributorChannels, p Params) {
	for h := 0; h < p.ImageHeight; h++ {
		for w := 0; w < p.ImageWidth; w++ {
			if world[h][w] == alive {
				c.events <- CellFlipped{turn, util.Cell{X: w, Y: h}}
			}
		}
	}
	c.events <- TurnComplete{turn}
}
