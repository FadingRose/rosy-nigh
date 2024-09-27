package fuzz

import (
	"crypto/rand"
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/cfg"
	"fadingrose/rosy-nigh/core"
	"fadingrose/rosy-nigh/core/state"
	"fadingrose/rosy-nigh/core/tracing"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fadingrose/rosy-nigh/mutator"
	"fadingrose/rosy-nigh/oracle"
	"fadingrose/rosy-nigh/smt"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type Mutator interface {
	GenerateArgs(abi.Method) ([]interface{}, []abi.Argument, []mutator.Seed)
	GenerateCallValue() *big.Int
	AddSolution(vm.RegKey, string)
	String() string
}

type Solver interface {
	SolveJumpIcondition(vm.RegKey) (string, bool)
	AddExclusion(name string, val *big.Int)
}

type WishSolver interface {
	AppendWishList(rk *vm.RegKey)
}

type Scheduler interface {
	GetFuncsSequence(rwmap *cfg.RWMap) ([]abi.Method, int)
	GetSingleFuncList() []abi.Method
	BadFuncs()
	GoodFuncs()
}

type Oracle interface {
	HumanReport() string
	DivideZeroCheck(model string)
}

type FuzzHost struct {
	StateDB      *state.StateDB
	BlockContext vm.BlockContext
	ChainConfig  params.ChainConfig
	EVMConfig    vm.Config

	OwnerAddress common.Address
	DeployAt     common.Address
	Attackers    []common.Address

	Mutator
	Solver
	WishSolver
	Scheduler // FunctionScheduler
	Oracle

	Target *Contract

	Err error

	evm *vm.EVM // Lastest EVM instance

	CFG         *cfg.CFG
	runtimeCode []byte

	// MethodImpl map[*abi.Method][]abi.Argument
}

func NewFuzzHost(target *Contract, statedb *state.StateDB, blockCtx vm.BlockContext, chainConfig params.ChainConfig, config vm.Config) *FuzzHost {
	mutator := mutator.NewMutator(target.ABI)
	solver := smt.NewSolver()
	scheduler := NewScheduler(target.ABI)
	oracle := oracle.NewOracleHost()
	attackers := func() []common.Address {
		nums := 10
		addrs := make([]common.Address, nums)
		for i := 0; i < nums; i++ {
			bs := make([]byte, 20)
			_, err := rand.Read(bs)
			if err != nil {
				panic(err)
			}
			addrs[i] = common.BytesToAddress(bs)
		}
		return addrs
	}()
	// cfg := cfg.NewCFG(target.CreationBin)
	return &FuzzHost{
		StateDB:      statedb,
		BlockContext: blockCtx,
		ChainConfig:  chainConfig,
		EVMConfig:    config,

		OwnerAddress: target.Creator,
		Attackers:    attackers,

		Mutator: mutator,

		Target:     target,
		Solver:     solver,
		WishSolver: nil,
		Scheduler:  scheduler,
		Oracle:     oracle,

		CFG:         nil,
		runtimeCode: make([]byte, 0),
		DeployAt:    common.Address{},

		// MethodImpl: make(map[*abi.Method][]abi.Argument),
	}
}

func (host *FuzzHost) RunForDeployOnchain() {
	nonce := host.StateDB.GetNonce(host.OwnerAddress)
	amount := big.NewInt(0)
	host.StateDB.AddBalance(host.OwnerAddress, uint256.NewInt(uint64(10000000)), tracing.BalanceChangeUnspecified)
	gasLimit := uint64(1000000)
	gasPrice := big.NewInt(0)
	signer := types.MakeSigner(&host.ChainConfig, big.NewInt(0), 0)
	basefee := big.NewInt(0)
	gaspool := core.GasPool(gasLimit)

	log.Info("(onchain) Fuzzing contract deployment", "constructor", host.Target.ABI.Constructor)

	codes := host.Target.CreationBin

	tx := types.NewContractCreation(nonce, amount, gasLimit, gasPrice, codes)
	msg, _ := core.TransactionToMessage(tx, signer, basefee)
	msg.From = host.OwnerAddress
	msg.SkipAccountChecks = true // do NOT check nonce
	context := core.NewEVMTxContext(msg)

	var (
		rules    = host.ChainConfig.Rules(host.BlockContext.BlockNumber, host.BlockContext.Random != nil, host.BlockContext.Time)
		coinbase = host.BlockContext.Coinbase
	)

	host.StateDB.Prepare(rules, host.OwnerAddress, coinbase, nil, vm.ActivePrecompiles(rules), nil)
	evm := vm.NewEVM(host.BlockContext, context, host.StateDB, &host.ChainConfig, host.EVMConfig)
	host.evm = evm
	result, err := core.ApplyMessage(evm, msg, &gaspool)
	if err != nil {
		log.Warn("Failed: ", "err", err, "name", host.Target.Name)
		host.Err = err
	}
	if result.Failed() {
		log.Warn("Failed: ", "err", result.Err, "name", host.Target.Name)
		if result.Err == vm.ErrExecutionReverted {
			revertData := result.Revert()
			if len(revertData) > 0 {
				log.Info("Revert with data: ", "data", revertData)
			}
		}
		host.Err = result.Err
	}

	// 5.a 5.b rebuild and update CFG
	// in RegKeyList(), regpool will rebuild itself
	// regList := host.evm.SymbolicPool.RegKeyList()
	// host.CFG.Update(regList)

	// log.Debug(host.CFG.String())

	if host.Err == nil {
		// update deploy at
		host.DeployAt = result.ContractAddr
		host.runtimeCode = result.ReturnData
		host.CFG = cfg.NewCFG(host.runtimeCode)
		host.WishSolver = smt.NewWishSolver(host.CFG)
		log.Info(fmt.Sprintf("deploy success at %s", host.DeployAt.Hex()))
	} else {
		log.Warn("Failed: ", "err", host.Err, "name", host.Target.Name)
	}
}

// WARN: Deprecated
func (host *FuzzHost) RunForDeploy() {
	nonce := host.StateDB.GetNonce(host.OwnerAddress)
	amount := big.NewInt(0)
	host.StateDB.AddBalance(host.OwnerAddress, uint256.NewInt(uint64(10000000)), tracing.BalanceChangeUnspecified)
	gasLimit := uint64(1000000)
	gasPrice := big.NewInt(0)
	signer := types.MakeSigner(&host.ChainConfig, big.NewInt(0), 0)
	basefee := big.NewInt(0)
	gaspool := core.GasPool(gasLimit)

	log.Info("Fuzzing contract deployment", "constructor", host.Target.ABI.Constructor)

	for {
		// if provides creation code, pack it directly
		// 1. Generate parameters
		//    1.a Create ArgIndexList for params
		// 2. Pack it into a transaction and sign it to get a Message
		// 3. Create a new EVM and run the message
		// 4. Success -> return, else
		// 5. Error ->
		//    5.a rebuild the regpool
		//    5.b update CFG, statement coverage and branch coverage
		//    5.c send to SMT, desire a better input / magic number

		var (
			argList []abi.ArgIndex
			regList []vm.RegKey
		)

		args, params, _ := host.Mutator.GenerateArgs(host.Target.ABI.Constructor)

		// 1.a Create ArgIndexList for params
		offset := uint64(0x80)
		argListToString := func(a []abi.ArgIndex) string {
			var s string
			for _, v := range a {
				s += v.String() + "\n"
			}
			return s
		}

		for i := range params {
			argList = append(argList,
				abi.ArgIndex{
					Contract: host.Target.Name,
					Method:   "",
					Type:     params[i].Type.String(),
					Name:     params[i].Name,
					Offset:   offset,
					Size:     uint64(params[i].Type.GetSize()),
					Val:      args[i],
				})
			offset += uint64(params[i].Type.GetSize())
		}
		log.Debug(fmt.Sprintf("deploy with args:\n%v", argListToString(argList)))

		// 2. Pack it into a transaction and sign it to get a Message
		var codes []byte
		var err error

		codes, err = PackTxConstructor(host.Target.ABI, host.Target.StaticBin, args...)
		if err != nil {
			fmt.Println("Error: ", err)
		}

		tx := types.NewContractCreation(nonce, amount, gasLimit, gasPrice, codes)
		msg, _ := core.TransactionToMessage(tx, signer, basefee)
		msg.From = host.OwnerAddress
		msg.SkipAccountChecks = true // do NOT check nonce
		context := core.NewEVMTxContext(msg)

		// 3. Create a new EVM and run the message
		var (
			rules    = host.ChainConfig.Rules(host.BlockContext.BlockNumber, host.BlockContext.Random != nil, host.BlockContext.Time)
			coinbase = host.BlockContext.Coinbase
		)

		host.StateDB.Prepare(rules, host.OwnerAddress, coinbase, nil, vm.ActivePrecompiles(rules), nil)
		evm := vm.NewEVM(host.BlockContext, context, host.StateDB, &host.ChainConfig, host.EVMConfig)
		host.evm = evm
		result, err := core.ApplyMessage(evm, msg, &gaspool)
		if err != nil {
			log.Warn("Failed: ", "err", err, "name", host.Target.Name)
			host.Err = err
		}
		if result.Failed() {
			log.Warn("Failed: ", "err", result.Err, "name", host.Target.Name)
			if result.Err == vm.ErrExecutionReverted {
				revertData := result.Revert()
				if len(revertData) > 0 {
					log.Info("Revert with data: ", "data", revertData)
				}
			}
			host.Err = result.Err
		}

		// 5.a 5.b rebuild and update CFG
		// in RegKeyList(), regpool will rebuild itself
		regList = host.evm.SymbolicPool.RegKeyList()
		host.CFG.Update(regList, "") // constructor's mehotd name is ""
		// host.wrapCandidates(argList, regList)

		log.Debug(host.CFG.String())

		// 4. Success -> return, else
		if host.Err == nil {
			log.Info("Success: ", "address", host.Target.Name)
			break
		}
		// HACK:  for debug, just break
		break
	}
}

func (host *FuzzHost) FuzzOnce(method abi.Method, sender common.Address) (runtimePath *cfg.Path, funcCovered int, funcTotal int, fzerr error) {
	// 0. Generate Sender's receive()
	// 1. Generate parameters
	//    1.a Create ArgIndexList for params
	// 2. Pack it into a transaction and sign it to get a Message
	// 3. Create a new EVM and run the message
	// 4. Success -> return, else
	// 5. Error ->
	//    5.a rebuild the regpool
	//    5.b update CFG, statement coverage and branch coverage
	//    5.c send to SMT, desire a better input / magic number

	var (
		nonce = host.StateDB.GetNonce(host.OwnerAddress)
		// from     = host.Attackers[0]
		from     = sender
		to       = host.DeployAt
		amount   = host.Mutator.GenerateCallValue()
		gasLimit = uint64(1000000)
		gasPrice = big.NewInt(0)
		signer   = types.MakeSigner(&host.ChainConfig, big.NewInt(0), 0)
		basefee  = big.NewInt(0)
		gaspool  = core.GasPool(gasLimit)
		argList  []abi.ArgIndex
		regList  []vm.RegKey
		receiver = SafeReceiver()
	)

	host.registReceiver(receiver)

	args, params, seeds := host.Mutator.GenerateArgs(method)
	// 1.a Create ArgIndexList for params
	offset := uint64(0x4) // the fisrt 4 bytes are method ID, args start from 4 bytes
	argListToString := func(a []abi.ArgIndex) string {
		var s string
		for _, v := range a {
			s += v.String() + "\n"
		}
		return s
	}
	for i := range params {
		argList = append(argList,
			abi.ArgIndex{
				Contract: host.Target.Name,
				Method:   method.Name,
				Type:     params[i].Type.String(),
				Name:     params[i].Name,
				Offset:   offset,
				Size:     uint64(params[i].Type.GetSize()),
				Val:      args[i],
			})
		offset += uint64(params[i].Type.GetSize())
	}
	log.Debug(fmt.Sprintf("runs with args:\nmsg.value %s\n%v", amount.String(), argListToString(argList)))

	// 2. Pack it into a transaction and sign it to get a Message
	data, err := host.Target.ABI.Pack(method.Name, args...)
	if err != nil {
		log.Warn("Failed to pack data: ", "err", err)
		return nil, 0, 0, err
	}
	// NOTE:argsdata := data[4:], data = method.ID + args

	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data)
	msg, _ := core.TransactionToMessage(tx, signer, basefee)
	msg.From = from
	msg.SkipAccountChecks = true // do NOT check nonce
	context := core.NewEVMTxContext(msg)
	// 3. Create a new EVM and run the message
	var (
		rules    = host.ChainConfig.Rules(host.BlockContext.BlockNumber, host.BlockContext.Random != nil, host.BlockContext.Time)
		coinbase = host.BlockContext.Coinbase
	)

	host.StateDB.Prepare(rules, from, coinbase, nil, vm.ActivePrecompiles(rules), nil)
	evm := vm.NewEVM(host.BlockContext, context, host.StateDB, &host.ChainConfig, host.EVMConfig)
	host.evm = evm
	result, err := core.ApplyMessage(evm, msg, &gaspool)
	if err != nil {
		log.Warn("Failed: ", "err", err, "name", host.Target.Name)
		host.Err = err
	}
	if result.Failed() {
		log.Warn("Failed: ", "err", result.Err, "name", host.Target.Name)
		if result.Err == vm.ErrExecutionReverted {
			revertData := result.Revert()
			if len(revertData) > 0 {
				log.Info("Revert with data: ", "data", revertData)
			}
		}
		host.Err = result.Err
	}

	// if host.Err != nil , drop seed, otherwise increase priority
	if host.Err != nil {
		for _, seed := range seeds {
			name, val := seed.Drop()
			host.Solver.AddExclusion(name, val)
		}
	} else {
		for _, seed := range seeds {
			name, val := seed.IncreasePriority()
			host.Solver.AddExclusion(name, val)
		}
	}

	// 5.a 5.b rebuild and update CFG
	// in RegKeyList(), regpool will rebuild itself
	regList = host.evm.SymbolicPool.RegKeyList()
	host.CFG.Update(regList, method.Name)

	runtimePath = host.CFG.ExtractPath(regList)

	// NOTE: if flag is True, it means all the solvable branch for the method has been covered
	// solvable: see wrapCandidates, ingeneral, after we expand a JUMPI, not all the BASE value source can be symbolized
	_, funcCovered, funcTotal = host.wrapCandidates(argList, regList, runtimePath)

	return runtimePath, funcCovered, funcTotal, nil
}

func PackTxConstructor(abi abi.ABI, code []byte, args ...interface{}) ([]byte, error) {
	// Pack the constructor data
	// constructor's name should be empty, in old versions, it should as same as contract name
	data, err := abi.Pack("", args...)
	if err != nil {
		return nil, err
		// panic(fmt.Sprintf("failed to pack data: %s", err))
	}
	code = append(code, data...)
	return code, nil
}

func (host *FuzzHost) Debug() {
	host.evm.SymbolicPool.Debug()
}

// wrapCandidates processes the register keys to identify and handle JUMPI instructions,
// binding arguments to registers and calculating branch coverage. It also sends candidates
// to the SMT solver for further analysis and updates the mutator with solutions.
func (host *FuzzHost) wrapCandidates(argList []abi.ArgIndex, regList []vm.RegKey, runtimePath *cfg.Path) ([]vm.RegKey, int, int) {
	// this is for create function
	isBind := func(rk vm.RegKey) (abi.ArgIndex, bool) {
		// Magic Reg
		if rk.IsMagic() {
			return abi.ArgIndex{}, true
		}

		// Bind Argument
		for _, arg := range argList {
			if rk.OpCode() != vm.MLOAD && rk.OpCode() != vm.CALLDATALOAD && rk.OpCode() != vm.CALLVALUE {
				continue
			}

			if rk.Offset() == arg.Offset {
				// Add verification for the value
				var (
					hex = rk.Instance().Data.Hex()
					dec = rk.Instance().Data.Dec()
					val = fmt.Sprintf("%v", arg.Val)
				)

				hex = strings.ToUpper(hex)
				val = strings.ToUpper(val)

				// at least one of them should be equal
				if hex != val && dec != val {
					log.Warn(fmt.Sprintf("Reg bind but value Mismatch: got %s(%s), want %s(%s)", hex, dec, val, arg.Name))
				}

				rk.Instance().ArgIndex = &arg
				return arg, true
			}
		}
		return abi.ArgIndex{}, false
	}

	var (
		candidates               []vm.RegKey
		totalFuncBranchCoverage  = 0
		coverdFuncBranchCoverage = 0
	)
	for _, rk := range regList {
		if rk.OpCode() == vm.JUMPI {
			// log.Debug(fmt.Sprintf("JUMPI\n%s\n", rk.Expand()))
			relies := rk.Relies()
			// NOTE: for now, we only impl branch coverage
			for _, relie := range relies {
				// log.Debug(fmt.Sprintf("relies: %s <- %s", relie.OpCode(), relie.IndexString()))
				if relie.IsBarrier() {
					fmt.Println("WARNING: relie is barrier at ", relie.String())
					fmt.Println(rk.Expand())

					host.WishSolver.AppendWishList(&relie)
				}

				if _, ok := isBind(relie); ok {
					coverd, total := host.CFG.BranchCoverageLine(rk.PC())
					// fmt.Printf("Branch coverage at %d: %d/%d\n", rk.PC(), coverd, total)
					totalFuncBranchCoverage += total
					coverdFuncBranchCoverage += coverd
					if coverd == total {
						continue
					}
					candidates = append(candidates, rk)
				}
			}
		}
	}

	log.Debug(fmt.Sprintf("candidate size: %d", len(candidates)))
	// DONE:collect all the JUMPI, expand it, if there is a bind in the collection, send it to SMT

	for _, candidate := range candidates {
		log.Debug(fmt.Sprintf("\n%s", candidate.Expand()))
		if model, satisfied := host.Solver.SolveJumpIcondition(candidate); satisfied {
			host.Oracle.DivideZeroCheck(model)
			host.Mutator.AddSolution(candidate, model)
		}
	}
	return candidates, coverdFuncBranchCoverage, totalFuncBranchCoverage
}

// host.registReceiver(receiver)
func (host *FuzzHost) registReceiver(receiver []byte) {
	host.StateDB.SetCode(host.OwnerAddress, receiver)
}
