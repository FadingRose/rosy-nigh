package mutator

import (
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ArgumentName -> Seed
type MethodVault = map[string]*Vault

type Mutator struct {
	// Method Name -> MethodVault
	MethodVaults      map[string]MethodVault
	MagicNumberVaults map[string]*Vault
	// TODO: in future use of method
	// SequenceVaults []Vault
}

type Magic string

const (
	CallValue Magic = "CallValue"
)

func magics() []Magic {
	return []Magic{
		CallValue,
	}
}

func NewMutator(_abi abi.ABI) *Mutator {
	mvs := make(map[string]MethodVault)
	magicvs := make(map[string]*Vault)

	// Magic Number Vaults
	for _, magic := range magics() {
		mv := new(Vault)
		magicvs[string(magic)] = mv

		switch magic {
		case CallValue:
			_seed := NewSeedImpl[*big.Int](mv)
			mv.baseSeed = _seed
			mv.InitVault(_seed, string(magic))
			mv.Inherit("0x0")
		default:
			fmt.Println("Unknown magic number")
		}
	}

	// Method Vaults
	_abi.Methods[""] = _abi.Constructor
	for _, method := range _abi.Methods {
		mv := make(MethodVault)

		for _, arg := range method.Inputs {
			argType := arg.Type.T // solidity types
			size := arg.Type.Size

			var _seed Seed
			var _vt Vault
			switch argType {
			case abi.UintTy:
				switch size {
				case 8:
					_seed = NewSeedImpl[uint8](&_vt)
				case 16:
					_seed = NewSeedImpl[uint16](&_vt)
				case 32:
					_seed = NewSeedImpl[uint32](&_vt)
				case 64:
					_seed = NewSeedImpl[uint64](&_vt)
				default:
					_seed = NewSeedImpl[*big.Int](&_vt)
				}

			case abi.IntTy:
				switch size {
				case 8:
					_seed = NewSeedImpl[int8](&_vt)
				case 16:
					_seed = NewSeedImpl[int16](&_vt)
				case 32:
					_seed = NewSeedImpl[int32](&_vt)
				case 64:
					_seed = NewSeedImpl[int64](&_vt)
				default:
					_seed = NewSeedImpl[*big.Int](&_vt)
				}

			case abi.BoolTy:
				_seed = NewSeedImpl[bool](&_vt)

			case abi.StringTy:
				_seed = NewSeedImpl[string](&_vt)

			case abi.SliceTy, abi.TupleTy, abi.BytesTy, abi.HashTy, abi.FunctionTy:
				_seed = NewSeedImpl[[]byte](&_vt)
			case abi.ArrayTy:
				_seed = NewSeedImpl[[]byte](&_vt)
			case abi.AddressTy:
				_seed = NewSeedImpl[common.Address](&_vt)

			case abi.FixedBytesTy, abi.FixedPointTy:
				switch size {
				case 1:
					_seed = NewSeedImpl[[1]byte](&_vt)

				case 2:
					_seed = NewSeedImpl[[2]byte](&_vt)

				case 3:
					_seed = NewSeedImpl[[3]byte](&_vt)

				case 4:
					_seed = NewSeedImpl[[4]byte](&_vt)

				case 5:
					_seed = NewSeedImpl[[5]byte](&_vt)

				case 6:
					_seed = NewSeedImpl[[6]byte](&_vt)

				case 7:
					_seed = NewSeedImpl[[7]byte](&_vt)

				case 8:
					_seed = NewSeedImpl[[8]byte](&_vt)

				case 9:
					_seed = NewSeedImpl[[9]byte](&_vt)

				case 10:
					_seed = NewSeedImpl[[10]byte](&_vt)

				case 11:
					_seed = NewSeedImpl[[11]byte](&_vt)

				case 12:
					_seed = NewSeedImpl[[12]byte](&_vt)

				case 13:
					_seed = NewSeedImpl[[13]byte](&_vt)

				case 14:
					_seed = NewSeedImpl[[14]byte](&_vt)

				case 15:
					_seed = NewSeedImpl[[15]byte](&_vt)

				case 16:
					_seed = NewSeedImpl[[16]byte](&_vt)

				case 17:
					_seed = NewSeedImpl[[17]byte](&_vt)

				case 18:
					_seed = NewSeedImpl[[18]byte](&_vt)

				case 19:
					_seed = NewSeedImpl[[19]byte](&_vt)

				case 20:
					_seed = NewSeedImpl[[20]byte](&_vt)

				case 21:
					_seed = NewSeedImpl[[21]byte](&_vt)

				case 22:
					_seed = NewSeedImpl[[22]byte](&_vt)

				case 23:
					_seed = NewSeedImpl[[23]byte](&_vt)

				case 24:
					_seed = NewSeedImpl[[24]byte](&_vt)

				case 25:
					_seed = NewSeedImpl[[25]byte](&_vt)

				case 26:
					_seed = NewSeedImpl[[26]byte](&_vt)

				case 27:
					_seed = NewSeedImpl[[27]byte](&_vt)

				case 28:
					_seed = NewSeedImpl[[28]byte](&_vt)

				case 29:
					_seed = NewSeedImpl[[29]byte](&_vt)

				case 30:
					_seed = NewSeedImpl[[30]byte](&_vt)

				case 31:
					_seed = NewSeedImpl[[31]byte](&_vt)

				case 32:
					_seed = NewSeedImpl[[32]byte](&_vt)

				default:
					fmt.Println("Unknown type")
				}
			}
			_vt.baseSeed = _seed

			argFullName := arg.ArguemntName()
			_vaultname := method.Name + ":" + argFullName

			_vt.InitVault(_seed, _vaultname)
			// all the arg.Name -> arg.ArgumentName() with type info
			mv[arg.ArguemntName()] = &_vt

		}

		mvs[method.Name] = mv
	}

	return &Mutator{
		MethodVaults:      mvs,
		MagicNumberVaults: magicvs,
	}
}

func (m *Mutator) GenerateArgs(me abi.Method) ([]interface{}, []abi.Argument, []Seed) {
	inputs := me.Inputs
	args := make([]interface{}, 0, len(inputs))
	var seeds []Seed
	for _, input := range inputs {
		vault := m.MethodVaults[me.Name][input.ArguemntName()]

		if vault == nil {
			panic("Vault is nil")
		}

		seed := vault.GetSeed()
		seeds = append(seeds, seed)
		args = append(args, seed.Val())
	}

	return args, inputs, seeds
}

func (m *Mutator) GenerateCallValue() *big.Int {
	// fmt.Println(m.MagicNumberVaults[string(CallValue)].Format())
	seed := m.MagicNumberVaults[string(CallValue)].GetSeed()
	// fmt.Println("get seed value: ", seed.Val())
	return seed.Val().(*big.Int)
}

type SolutionType int // 0: method input 1: magic number

const (
	MethodInput SolutionType = iota
	MagicNumber
)

type Solution struct {
	MethodName   string
	ArgumentName string

	MagicNumber string

	Value string

	SolutionType
}

// Impl Mutator Interface
func (m *Mutator) AddSolution(rk vm.RegKey, solution string) {
	var info strings.Builder

	info.WriteString("\n * [Mutator] * Receive Solution\n")
	solutions := m.parseSolution(solution)
	for _, sol := range solutions {
		// fmt.Printf("method: %s, arg: %s, value: %s\n", sol.MethodName, sol.ArgumentName, sol.Value)
		if sol.SolutionType == MagicNumber {
			info.WriteString(fmt.Sprintf(" * [Mutator] %s <- %s\n", sol.MagicNumber, sol.Value))
		} else {
			info.WriteString(
				fmt.Sprintf(
					" * [Mutator] Method: %s, Arg: %s, Value: %s\n",
					sol.MethodName,
					sol.ArgumentName,
					sol.Value,
				),
			)
		}
		m.addSolution(rk, sol)
	}
	info.WriteString(" * [Mutator] * End of Solution\n\n")

	log.Info(info.String())
}

func (m *Mutator) addSolution(rk vm.RegKey, solution Solution) {
	if solution.SolutionType == MagicNumber {
		log.Info(fmt.Sprintf("Inherit MagicVault %s : %s\n", solution.MagicNumber, solution.Value))
		m.MagicNumberVaults[solution.MagicNumber].Inherit(solution.Value)
		return
	}

	methodName := solution.MethodName
	argumentName := solution.ArgumentName
	value := solution.Value

	vault := m.MethodVaults[methodName][argumentName]

	if vault == nil {
		_info := fmt.Sprintf("Vault is nil for  %s:%s", methodName, argumentName)
		panic(_info)
	}

	// TODO: check whether seed creation fails
	vault.Inherit(value)
}

// TODO: report potential divide by zero error
func (m *Mutator) parseSolution(pure string) []Solution {
	ret := make([]Solution, 0)

	// parse solution string
	// <methodname>:<type>_<argumentname> -> <value>
	// :uint256_x -> 1
	// TODO: :uint256,x -> 1
	// <MagicNumber> -> <Value>
	ss := strings.Split(pure, "\n")

	for _, s := range ss {
		if s == "" {
			continue
		}

		sol := strings.Split(s, " -> ")

		if len(sol) != 2 {
			log.Warn("invalid solution format" + s)
			continue
		}

		if sol[1] == "{" {
			log.Warn("invalid solution format" + s)
			continue
		}

		method := ""
		arg := ""

		if sol[0][0] == ':' {
			// constructor
			arg = sol[0][1:]
		} else {
			_ss := strings.Split(sol[0], ":")
			if len(_ss) != 2 {
				// MAGIC NUMBER

				ret = append(ret, Solution{
					MethodName:   "",
					ArgumentName: "",
					Value:        sol[1],
					MagicNumber:  sol[0],
					SolutionType: MagicNumber,
				})
				continue
			}

			method = _ss[0]
			arg = _ss[1]
		}

		ret = append(ret, Solution{
			MethodName:   method,
			ArgumentName: arg,
			Value:        sol[1],
			MagicNumber:  "",
			SolutionType: MethodInput,
		})

	}

	return ret
}

func (m *Mutator) String() string {
	var builder strings.Builder
	for methodName, mv := range m.MethodVaults {
		builder.WriteString(fmt.Sprintf("Method: %s\n", methodName))
		for argName, v := range mv {
			builder.WriteString(fmt.Sprintf("    Arg: %s\n", argName))
			builder.WriteString(v.String())
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	return builder.String()
}
