package smt

import (
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"math/big"
	"strings"

	"github.com/aclements/go-z3/z3"
	"github.com/holiman/uint256"
)

type exclusion struct {
	name string
	val  *big.Int
}

type exclusionIndex string

func (ex *exclusion) Index() exclusionIndex {
	return exclusionIndex(ex.name + ex.val.String())
}

type SMTSolverZ3 struct {
	solver *z3.Solver
	config *z3.Config

	// The true ctx should split with different path resolve
	ctx []*z3.Context

	// Only for development using
	_ctx *z3.Context

	exclusion map[exclusionIndex]exclusion
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

		exclusion: make(map[exclusionIndex]exclusion),
	}
}

// AddExclusion add a exclusion to SMT, invoked by duplicate path detector
func (s *SMTSolverZ3) AddExclusion(name string, val *big.Int) {
	index := exclusionIndex(name + val.String())
	if _, ok := s.exclusion[index]; !ok {
		log.Info(fmt.Sprintf(" * [SMT] * Receive Exclusion %s != %s", name, val))
		s.exclusion[index] = exclusion{name, val}
	} else {
		log.Info("add exclusion failed, already existed")
	}
}

func (s *SMTSolverZ3) loadExclusions() {
	for _, e := range s.exclusion {
		_var := s._ctx.IntConst(e.name)
		_low_quality := s._ctx.FromBigInt(e.val, s._ctx.IntSort()).(z3.Int)
		log.Info(fmt.Sprintf(" * [SMT] * Load Exclusion %s != %s", _var, _low_quality))

		s.solver.Assert(_var.NE(_low_quality))
	}
}

// SolveJumpIcondition(vm.RegKey) (string, bool)
// TODO; impl this
func (s *SMTSolverZ3) SolveJumpIcondition(regKey vm.RegKey) (string, bool) {
	// Reset Solver
	s.solver.Reset()

	cond := regKey.Instance().L

	// the condition's conditon may be a Literal
	if cond.OpCode().IsPush() {
		return "", false
	}

	s.resolveCondition(cond)

	// Add exclusions Condition
	// example :uint256_x -> 0 is a Low-Quaity Seed, then add a Assertion
	// :uint256_x != 0
	//
	//	_y := s._ctx.IntConst(":uint256_y")
	//	_low_quality :=.FromBigInt(big.NewInt(int64(9)), s._ctx.IntSort()).(z3.Int)
	//	s.solver.Assert(_y.NE(_low_quality))
	//
	//s.AddExclusion(":uint256_y", big.NewInt(int64(9)))
	s.loadExclusions()

	if sat, err := s.solver.Check(); err != nil || !sat {
		log.Warn(fmt.Sprintf("<- * [SMT] * End : Not satified, %s", err))
		return "", false
	}

	log.Info(fmt.Sprintf("\n * !![SMT]!! * Satisfied *********\n * !![SMT]!! * s.solver.Model():\n%v * !![SMT]!! *************** *********", s.solver.Model()))
	return s.solver.Model().String(), true
}

func (s *SMTSolverZ3) resolveCondition(cond *vm.Reg) {
	l := s.resolve(cond.L)

	// there are two potential condition
	// _r == nil or _r != nil

	if cond.M == nil {
		// [1] _r == nil means the condition is a Unary Expression
		// we try to assert ~_l.val
		// condition L eq 0x00?
		// -> Reg.val is 0x01 -> try to assert NEQ
		// -> Reg.val is 0x00 -> try to assert EQ
		log.Info(fmt.Sprintf("   -- SMT: Nary Expression %s\n", cond.String()))
		zero := uint256.NewInt(uint64(0))
		_zero := s._ctx.FromBigInt(zero.ToBig(), s._ctx.IntSort()).(z3.Int)

		if cond.Data.Eq(zero) {
			s.solver.Assert(l.Eq(_zero))
		} else {
			s.solver.Assert(l.NE(_zero))
		}

		return
	}

	// [2] _r != nil means the condition is a Binary Expression
	// we try to assert

	log.Info(fmt.Sprintf("   -- SMT: Binary Expression %s\n", cond.String()))
	r := s.resolve(cond.M)

	zero := uint256.NewInt(uint64(0))
	z3True := s._ctx.FromBigInt(big.NewInt(int64(1)), s._ctx.IntSort()).(z3.Int)
	z3False := s._ctx.FromBigInt(big.NewInt(int64(0)), s._ctx.IntSort()).(z3.Int)
	z3Zero := s._ctx.FromBigInt(zero.ToBig(), s._ctx.IntSort()).(z3.Int)
	expr := ""
	switch cond.OpCode() {
	case vm.EQ:
		// condition L eq R?
		if cond.Data.Eq(zero) { // R != L for now
			// try to assert -> R == L
			expr = fmt.Sprintf(" %s == %s ", r, l)
			s.solver.Assert(l.Eq(r))
		} else { // R == L for now
			// try to assert -> R != L
			expr = fmt.Sprintf(" %s != %s ", r, l)
			s.solver.Assert(l.NE(r))
		}
	case vm.LT, vm.SLT: // R > L ?
		if cond.Data.Eq(zero) { // R < L for now
			// try to assert -> R > L
			expr = fmt.Sprintf(" %s > %s ", r, l)
			s.solver.Assert(r.GT(l))
		} else { // R > L for now
			// try to assert -> R <= L
			expr = fmt.Sprintf(" %s <= %s ", r, l)
			s.solver.Assert(r.LE(l))
		}
	case vm.GT, vm.SGT: // R < L?
		if cond.Data.Eq(zero) { // R > L for now
			// try to assert -> R < L
			expr = fmt.Sprintf(" %s < %s ", r, l)
			s.solver.Assert(r.LT(l))
		} else { // R < L for now
			// try to assert -> R >= L
			expr = fmt.Sprintf(" %s >= %s ", r, l)
			s.solver.Assert(r.GE(l))
		}

	case vm.AND: // R & L
		if cond.Data.Eq(zero) { // R & L == 0 for now
			// try to assert -> R & L == 1
			expr = fmt.Sprintf(" %s & %s == 1 ", r, l)
			s.solver.Assert(r.Eq(z3True))
			s.solver.Assert(l.Eq(z3True))
		} else { // R & L == 1 for now
			// TODO there are three branches to validate
			// try to assert three cases
			s.solver.Assert(r.Eq(z3False))
			s.solver.Assert(l.Eq(z3False))
		}
	case vm.OR: // R | L
		if cond.Data.Eq(zero) {
			// R | L == 0 for now
			// try to assert -> R | L != 0
			// at least R or L is not 0

			expr = fmt.Sprintf(" %s | %s != 0 ", r, l)
			s.solver.Assert(r.Eq(z3True))
			s.solver.Assert(l.Eq(z3True))

		} else {
			// R | L == 1 for now

			expr = fmt.Sprintf(" %s | %s == 0 ", r, l)
			s.solver.Assert(r.Eq(z3False))
			s.solver.Assert(l.Eq(z3False))
		}

	case vm.SUB: // R - L
		if cond.Data.Eq(zero) { // R - L == 0 for now
			// try to assert -> R - L != 0
			expr = fmt.Sprintf(" %s - %s != 0 ", r, l)
			s.solver.Assert(r.NE(l))
		} else {
			// try to assert -> R - L == 0
			expr = fmt.Sprintf(" %s - %s == 0 ", r, l)
			s.solver.Assert(r.Eq(l))
		}

	case vm.MUL: // R * L
		if cond.Data.Eq(zero) { // R * L == 0 for now
			// try to asert -> R * L != 0
			expr = fmt.Sprintf(" %s * %s != 0 ", r, l)
			// TODO split to two assertions
			// WARNING NOW the assertion is imprefect
			s.solver.Assert(r.NE(z3Zero))
			s.solver.Assert(l.NE(z3Zero))
		} else {
			// try to assert -> R * L == 0
			expr = fmt.Sprintf(" %s * %s == 0 ", r, l)
			s.solver.Assert(r.Eq(z3Zero))
			s.solver.Assert(l.Eq(z3Zero))
		}

	case vm.DIV: // R / L
		if cond.Data.Eq(zero) { // R / L == 0 for now
			// try to assert -> R / L != 0
			expr = fmt.Sprintf(" %s / %s != 0 ", r, l)
			s.solver.Assert(r.NE(z3Zero))
			s.solver.Assert(l.NE(z3Zero))
		} else {
			// try to assert -> R / L == 0
			expr = fmt.Sprintf(" %s / %s == 0 ", r, l)
			s.solver.Assert(r.Eq(z3Zero))
			s.solver.Assert(l.NE(z3Zero))
		}
	default:
		panic("unhandled condition reg's op " + cond.OpCode().String())
	}

	log.Info(fmt.Sprintf("   -- SMT: Tring Asserting  %s\n", expr))
}

func (s *SMTSolverZ3) resolve(reg *vm.Reg) z3.Int {
	// Stop resolve at these condition
	op := reg.OpCode()

	log.Info(fmt.Sprintf("   -- SMT: Resolving %s\n", reg.String()))

	// Create Symbolic Variable First
	if name, err := reg.GetBindName(); err == nil {
		log.Info(fmt.Sprintf("   -! SMT: Creating Symbolic Variable %s -> %s\n", reg.Name(), name))
		tpname, _ := reg.GetBindType()
		log.Info(fmt.Sprintf("   -! SMT: Add %s type assertion constrains", tpname))
		if tpname == "" {
			log.Warn("no type for " + name)
		}

		if strings.Contains(tpname, "uint") {

			tps := ""
			var i *big.Int

			if strings.Contains(tpname, "8") {
				i = new(big.Int).SetUint64(uint64(255))
				tps = "uint8"
			}

			if strings.Contains(tpname, "16") {
				i = new(big.Int).SetUint64(uint64(65535))
				tps = "uint16"
			}

			if strings.Contains(tpname, "32") {
				i = new(big.Int).SetUint64(uint64(4294967295))
				tps = "uint32"
			}

			if strings.Contains(tpname, "64") {
				i = new(big.Int).SetUint64(uint64(18446744073709551615))
				tps = "uint64"
			}

			if strings.Contains(tpname, "256") {
				_i := uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
				i = _i.ToBig()
				tps = "uint256"
			}

			if tps == "" {
				log.Warn("unknown int type : " + tpname)
			}

			_zero := s._ctx.FromBigInt(big.NewInt(int64(0)), s._ctx.IntSort()).(z3.Int)
			_upper := s._ctx.FromBigInt(i, s._ctx.IntSort()).(z3.Int)
			_var := s._ctx.IntConst(name)
			s.solver.Assert(_var.GE(_zero))
			s.solver.Assert(_var.LE(_upper))

			log.Info(fmt.Sprintf("   -! SMT: Add %s type assertion constrains", tps))
			return _var
		}

		if strings.Contains(tpname, "int") {
			tps := ""

			var i, j *big.Int
			if strings.Contains(tpname, "8") {
				i = new(big.Int).SetInt64(int64(127))
				j = new(big.Int).SetInt64(int64(-128))
				tps = "int8"
			}

			if strings.Contains(tpname, "16") {
				i = new(big.Int).SetInt64(int64(32767))
				j = new(big.Int).SetInt64(int64(-32768))
				tps = "int16"
			}

			if strings.Contains(tpname, "32") {
				i = new(big.Int).SetInt64(int64(2147483647))
				j = new(big.Int).SetInt64(int64(-2147483648))
				tps = "int32"
			}

			if strings.Contains(tpname, "64") {
				i = new(big.Int).SetInt64(int64(9223372036854775807))
				j = new(big.Int).SetInt64(int64(-9223372036854775808))
				tps = "int64"
			}

			if tps == "" {
				log.Warn("int unknown type")
			}

			_upper := s._ctx.FromBigInt(i, s._ctx.IntSort()).(z3.Int)
			_floor := s._ctx.FromBigInt(j, s._ctx.IntSort()).(z3.Int)
			_var := s._ctx.IntConst(name)
			s.solver.Assert(_var.GE(_floor))
			s.solver.Assert(_var.LE(_upper))

			log.Info(fmt.Sprintf("   -! SMT: Add %s type assertion constrains", tps))
			return _var
		}

		return s._ctx.IntConst(name)
	}

	var exec func(*SMTSolverZ3, *vm.Reg) z3.Int
	if op.IsPush() {
		log.Info(fmt.Sprintf("   -- *SMT: Loading Literal %s <- %s\n", &reg.Data, reg.Name()))
		exec = literal
	} else {
		exec = smtHandlermap()[op]
	}

	if exec == nil {
		log.Warn(fmt.Sprintf("unhandled op %s", op.String()))
	}

	return exec(s, reg)
}

type smtHandlerMap map[vm.OpCode]smtHandler

type smtHandler func(*SMTSolverZ3, *vm.Reg) z3.Int

func smtHandlermap() smtHandlerMap {
	return smtHandlerMap{
		vm.CALLCODE:     literal,
		vm.CODESIZE:     literal,
		vm.CALLDATACOPY: literal,
		vm.CALLDATASIZE: literal,
		vm.CALLVALUE:    literal,
		vm.CALLER:       literal,
		vm.MLOAD:        literal,
		vm.SLOAD:        literal,
		vm.KECCAK256:    literal,
		vm.ISZERO:       literal,
		vm.EXTCODESIZE:  literal,
		vm.ADDRESS:      literal,
		vm.ORIGIN:       literal,
		vm.TIMESTAMP:    literal,
		vm.CHAINID:      literal,
		vm.NUMBER:       literal,
		vm.ADD:          alu,
		vm.SUB:          alu,
		vm.MUL:          alu,
		vm.DIV:          alu,
		vm.NOT:          alu,
		vm.AND:          alu,
		vm.OR:           alu,
		vm.EXP:          alu,
		vm.SHL:          alu,
		vm.SHR:          alu,
		vm.MOD:          alu,
		vm.LT:           logic,
		vm.SLT:          logic,
		vm.GT:           logic,
		vm.SGT:          logic,
		vm.EQ:           logic,
	}
}

func literal(smt *SMTSolverZ3, reg *vm.Reg) z3.Int {
	if reg.OpCode().IsPush() {

		log.Info(fmt.Sprintf("   -- *SMT: Loading Literal %s <- %s\n", &reg.Data, reg.String()))
		val := reg.Data
		_val := smt._ctx.FromBigInt(val.ToBig(), smt._ctx.IntSort()).(z3.Int)
		return _val
	}

	log.Info(fmt.Sprintf("   -- *SMT: Loading Literal  %s <- %s\n", &reg.Data, reg.OpCode().String()))

	if reg.OpCode() == vm.CALLVALUE {
		_var := smt._ctx.IntConst("CallValue")
		_zero := smt._ctx.FromBigInt(big.NewInt(int64(0)), smt._ctx.IntSort()).(z3.Int)
		smt.solver.Assert(_var.GE(_zero))
		return _var
	}

	val := reg.Data
	_val := smt._ctx.FromBigInt(val.ToBig(), smt._ctx.IntSort()).(z3.Int)
	return _val
}

func logic(smt *SMTSolverZ3, reg *vm.Reg) z3.Int {
	var expression string
	var _val z3.Int
	switch reg.OpCode() {
	case vm.SLT, vm.LT:
		left := smt.resolve(reg.L)
		right := smt.resolve(reg.M)

		_bool := right.LT(left)
		smt.solver.Assert(_bool)

		val := reg.Data
		_val = smt._ctx.FromBigInt(val.ToBig(), smt._ctx.IntSort()).(z3.Int)

		expression = fmt.Sprintf(" %s < %s ", reg.L.Name(), reg.M.Name())
	case vm.SGT, vm.GT:
		left := smt.resolve(reg.L)
		right := smt.resolve(reg.M)

		_bool := right.GT(left)
		smt.solver.Assert(_bool)

		val := reg.Data
		_val = smt._ctx.FromBigInt(val.ToBig(), smt._ctx.IntSort()).(z3.Int)

		expression = fmt.Sprintf(" %s > %s ", reg.L.Name(), reg.M.Name())
	case vm.EQ:
		left := smt.resolve(reg.L)
		right := smt.resolve(reg.M)

		_bool := right.Eq(left)
		smt.solver.Assert(_bool)

		val := reg.Data
		_val = smt._ctx.FromBigInt(val.ToBig(), smt._ctx.IntSort()).(z3.Int)

		expression = fmt.Sprintf(" %s == %s ", reg.L.Name(), reg.M.Name())
	default:
		_info := fmt.Sprintf("smt : unhandled logic %s", reg.OpCode().String())
		panic(_info)
	}
	log.Info(fmt.Sprintf("   -- *SMT: Logic Operations : %s\n", expression))

	return _val
}

func alu(smt *SMTSolverZ3, reg *vm.Reg) z3.Int {
	var expression string
	var ret z3.Int
	switch reg.OpCode() {
	case vm.ADD:
		left := smt.resolve(reg.L)
		right := smt.resolve(reg.M)

		ret = right.Add(left)

		expression = fmt.Sprintf(" %s + %s ", reg.M.Name(), reg.L.Name())
	case vm.SUB:
		left := smt.resolve(reg.L)
		right := smt.resolve(reg.M)

		ret = right.Sub(left)

		expression = fmt.Sprintf(" %s - %s ", reg.M.Name(), reg.L.Name())
	case vm.MUL:

		left := smt.resolve(reg.L)
		right := smt.resolve(reg.M)

		ret = right.Mul(left)

		expression = fmt.Sprintf(" %s * %s ", reg.M.Name(), reg.L.Name())
	case vm.DIV:

		left := smt.resolve(reg.L)
		right := smt.resolve(reg.M)

		ret = right.Div(left)

		expression = fmt.Sprintf(" %s / %s ", reg.M.Name(), reg.L.Name())
	case vm.AND:

		left := smt.resolve(reg.L).ToBV(256)
		right := smt.resolve(reg.M).ToBV(256)

		_ret := right.And(left)
		ret = _ret.SToInt()

		expression = fmt.Sprintf(" %s & %s ", reg.M.Name(), reg.L.Name())
	case vm.OR:
		left := smt.resolve(reg.L).ToBV(256)
		right := smt.resolve(reg.M).ToBV(256)

		_ret := right.Or(left)
		ret = _ret.SToInt()

		expression = fmt.Sprintf(" %s | %s ", reg.M.Name(), reg.L.Name())
	case vm.EXP, vm.SHL, vm.SHR, vm.MOD:
		val := reg.Data
		_val := smt._ctx.FromBigInt(val.ToBig(), smt._ctx.IntSort()).(z3.Int)
		ret = _val

		expression = fmt.Sprintf(" %s %s ", reg.OpCode().String(), val)
	case vm.NOT:
		left := smt.resolve(reg.L).ToBV(256)
		_ret := left.Not()
		ret = _ret.SToInt()
		expression = fmt.Sprintf(" ~ %s ", reg.L.Name())
	default:
		panic(fmt.Sprintf("******** SMT: Unresolved operation %s of %s", reg.OpCode(), reg.String()))
	}
	log.Info(fmt.Sprintf("   -- *SMT: ALU Operations : %s\n", expression))
	return ret
}
