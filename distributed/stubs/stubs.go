package stubs

// Control GOL
var RunGOL = "GOL_RPC.RunGOL"
var ExitCurGOL = "GOL_RPC.ExitCurGOL"
var PauseCurGOL = "GOL_RPC.PauseCurGOL"
var ContinueCurGOL = "GOL_RPC.ContinueCurGOL"

// Get the current state of GOL
var CurAliveNum = "GOL_RPC.CurAliveNum"
var CurWorld = "GOL_RPC.CurWorld"

// Quit all, including server and client
var QuitAll = "GOL_RPC.QuitAll"

// The request for rpc
// Not all RPCs use all parameters
type RPCRequest struct {
	World 		[][]uint8
	Turns 		int
	ImageWidth  int
	ImageHeight int
}

// The response for rpc
// Not all RPCs use all parameters
type RPCResponse struct {
	World 		[][]uint8
	Turn 		int
	AliveCells 	int
}