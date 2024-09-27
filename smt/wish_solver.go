package smt

import (
	"fadingrose/rosy-nigh/cfg"
	"fadingrose/rosy-nigh/core/vm"
	"fmt"
)

// type WishSolver interface {
// 	AppendWishList(rk *RegKey, wt WishType)
// }

// type WishType int
//
// const (
// 	SLOT WishType = iota
// 	TSTORAGE
// 	MEMORY
// )

// AbstractStatus is the abstract status of a register's solution
// unkonwn: dependencies are not solved or include memory access
// fixed: the value is fixed
// dynamic0: the value is completely determined by the input and magic numbers
// dynamic1: depends partly on the TLOAD and TSTORE
// dynamic2: depends partly on the SLOAD and SSTORE
type AbstractStatus int

const (
	unknown AbstractStatus = iota
	fixed
	dynamic0
	dynamic1
	dynamic2
)

type WishSolver struct {
	cfg *cfg.CFG
}

func NewWishSolver(cfg *cfg.CFG) *WishSolver {
	return &WishSolver{
		cfg: cfg,
	}
}

func (ws *WishSolver) AppendWishList(rk *vm.RegKey) {
	switch rk.OpCode() {
	case vm.SLOAD:
		ws.appendSload(rk)
	default:
		panic("unhandled opcode append to wish list" + rk.OpCode().String())
	}
}

func (ws *WishSolver) appendSload(rk *vm.RegKey) {
	slot := rk.Instance().Slot()
	funcs, exist := ws.cfg.RWMap().Filter(&slot, cfg.Write)
	if !exist {
		fmt.Println("no SSTORE exist")
	} else {
		fmt.Printf("SSTORE exist %v\n", funcs)
	}
}

func (ws *WishSolver) abstractSolve(rk *vm.RegKey) {
}
