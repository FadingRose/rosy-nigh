package smt

import (
	"fadingrose/rosy-nigh/core/vm"

	"github.com/aclements/go-z3/z3"
)

type SMTSolverZ3 struct {
	solver *z3.Solver
	config *z3.Config

	// The true ctx should split with different path resolve
	ctx []*z3.Context

	// Only for development using
	_ctx *z3.Context

	// exclusion []Exclusion
	// exclusion map[ExclusionIndex]Exclusion
}

func NewSolver() *SMTSolverZ3 {
	config := z3.NewContextConfig()
	context := z3.NewContext(config)
	solver := z3.NewSolver(context)

	return &SMTSolverZ3{
		solver: solver,
		config: config,
		ctx:    []*z3.Context{},
		_ctx:   context,
	}
}

// SolveJumpIcondition(vm.RegKey) (string, bool)
// TODO; impl this
func (s *SMTSolverZ3) SolveJumpIcondition(regKey vm.RegKey) (string, bool) {
	// Reset Solver
	s.solver.Reset()
	return "", false
}
