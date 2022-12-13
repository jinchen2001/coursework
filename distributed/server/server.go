package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
	"os"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type GOL_RPC struct {}

/***************************************************************************************************/
/***************************************************************************************************/
/***************************************** helpful variable ****************************************/
/***************************************************************************************************/
/***************************************************************************************************/
const alive = 255
const dead  = 0

// current state
var curTurn int
var curWorld [][]uint8
var curAliveNum int
var curStateMutex sync.Mutex

// Used to control the pause of RunGOL
var isPause bool = false
var pauseMutex sync.Mutex

// Used to exit the RunGOL when ExitCurGOL is called
var isExitCurTurn bool = false
var exitCurTurnMutex sync.Mutex

var waitGOL bool = true


/***************************************************************************************************/
/***************************************************************************************************/
/*********************************************** main **********************************************/
/***************************************************************************************************/
/***************************************************************************************************/
func main(){
	pAddr := flag.String("port","8030","Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GOL_RPC{})
	listener, _ := net.Listen("tcp", ":" + *pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}


/***************************************************************************************************/
/***************************************************************************************************/
/***************************************** helpful function ****************************************/
/***************************************************************************************************/
/***************************************************************************************************/
// Get the total number of alive cells
func getAliveCellsNumber(world [][]uint8, width int, height int) int {
	result := 0
	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			if world[h][w] == alive {
				result++
			}
		}
	}
	return result
}

// Get the number of alive neighbors of cell(h, w)
func getAliveNeighboursNum(world [][]uint8, width int, height int, h int, w int) int {
	result := 0
	for hb := -1; hb <= 1; hb++ {
		for wb := -1; wb <= 1; wb++ {
			if (hb == 0 && wb == 0) {
				continue
			}
			curh := (h + hb + height) % height
			curw := (w + wb + width) % width
			if world[curh][curw] == alive {
				result++
			}
		}
	}
	return result
}

// Get the next world
// No modification in current world
func getNextWorld(world [][]uint8, width int, height int, nextWorld [][]uint8) {
	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			curState := world[h][w]
			aliveNeighborNum := getAliveNeighboursNum(world, width, height, h, w)
			if curState == alive {
				if aliveNeighborNum < 2 || aliveNeighborNum > 3 {
					nextWorld[h][w] = dead
				} else {
					nextWorld[h][w] = alive
				}
			} else {
				if aliveNeighborNum == 3 {
					nextWorld[h][w] = alive
				} else {
					nextWorld[h][w] = dead
				}
			}
		}
	}
}


/***************************************************************************************************/
/***************************************************************************************************/
/*********************************************** RPC ***********************************************/
/***************************************************************************************************/
/***************************************************************************************************/
// RPC for RunGOL
// Start Game of Life
func (s *GOL_RPC) RunGOL(request stubs.RPCRequest, response *stubs.RPCResponse) (err error) {
	// Initial
	waitGOL = true
	curStateMutex.Lock()
	curWorld = request.World
	curTurn = 0
	curAliveNum = getAliveCellsNumber(curWorld, request.ImageWidth, request.ImageHeight)
	curStateMutex.Unlock()

	// start turns of GOL
	for curTurn < request.Turns {
		if isExitCurTurn {
			exitCurTurnMutex.Lock()
			isExitCurTurn = false
			exitCurTurnMutex.Unlock()
			break
		}

		for isPause {
			// wait until ContinueCurGOL is called
		}

		// store the next turn in nextWorld
		nextWorld := make([][]uint8, request.ImageHeight)
		for i := range nextWorld {
			nextWorld[i] = make([]uint8, request.ImageWidth)
		}
		getNextWorld(curWorld, request.ImageWidth, request.ImageHeight, nextWorld)

		tmp := getAliveCellsNumber(nextWorld, request.ImageWidth, request.ImageHeight)
		curStateMutex.Lock()
		curWorld = nextWorld
		curAliveNum = tmp
		curTurn++
		curStateMutex.Unlock()
	}

	response.Turn = curTurn
	response.World = curWorld
	fmt.Printf("The Game of Life ends with turns = %d\n", curTurn)
	waitGOL = false
	return
}

// RPC for PauseCurGOL
// Pause the current turn of GOL
func (s *GOL_RPC) PauseCurGOL(request stubs.RPCRequest, response *stubs.RPCResponse) (err error) {
	pauseMutex.Lock()
	isPause = true
	pauseMutex.Unlock()

	curStateMutex.Lock()
	response.World = curWorld
	response.Turn = curTurn + 1
	curStateMutex.Unlock()	
	return
}

// RPC for ContinueCurGOL
// Continue the current turn of GOL
func (s *GOL_RPC) ContinueCurGOL(request stubs.RPCRequest, response *stubs.RPCResponse) (err error) {
	pauseMutex.Lock()
	isPause = false
	pauseMutex.Unlock()

	curStateMutex.Lock()
	response.Turn = curTurn + 1
	curStateMutex.Unlock()	
	return
}

// RPC for ExitCurGOL
// Exit current turn of GOL
func (s *GOL_RPC) ExitCurGOL(request stubs.RPCRequest, response *stubs.RPCResponse) (err error) {
	exitCurTurnMutex.Lock()
	isExitCurTurn = true
	exitCurTurnMutex.Unlock()
	
	curStateMutex.Lock()
	response.World = curWorld
	response.Turn = curTurn
	response.AliveCells = curAliveNum
	curStateMutex.Unlock()	

	return
}

// RPC for CurAliveNum
// Get the number of alive cells in current state
func (s *GOL_RPC) CurAliveNum(request stubs.RPCRequest, response *stubs.RPCResponse) (err error) {
	curStateMutex.Lock()
	response.AliveCells = curAliveNum
	response.Turn = curTurn
	curStateMutex.Unlock()
	return
}

// RPC for CurWorld
// Get current world
func (s *GOL_RPC) CurWorld(request stubs.RPCRequest, response *stubs.RPCResponse) (err error) {
	curStateMutex.Lock()
	response.World = curWorld
	response.Turn = curTurn
	curStateMutex.Unlock()
	return
}

// RPC for QuitAll
// Exit the server
// And it need wait for the end of GOL
func (s *GOL_RPC) QuitAll(request stubs.RPCRequest, response *stubs.RPCResponse) (err error) {
	exitCurTurnMutex.Lock()
	isExitCurTurn = true
	exitCurTurnMutex.Unlock()
	for waitGOL {
		// wait for RunGOL to finish 
	}
	time.Sleep(1 * time.Second)
	fmt.Println("Quit")
	os.Exit(0)
	return
}