package fuzz

import (
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/core"
	"fadingrose/rosy-nigh/core/state"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fadingrose/rosy-nigh/onchain"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type Contract struct {
	ABI         abi.ABI
	StaticBin   []byte // static bytecode without depoy arguments
	CreationBin []byte // static bytecode with deploy arguments
	RuntimeBin  []byte // runtime bytecode
	Version     string
	Name        string
	Creator     common.Address
}

func CreateEVMRuntimeEnironment() (statedb *state.StateDB, blockCtx vm.BlockContext, chainConfig params.ChainConfig, config vm.Config) {
	statedb = state.NewStateDB()

	// BlockContext
	blockCtx = vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,

		GetHash: core.GetHashFn(nil, nil),

		Coinbase:    common.Address{},
		BlockNumber: big.NewInt(1),
		Time:        uint64(1), // TO ENABLE SHANGHAI FORK
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(0),
		BlobBaseFee: big.NewInt(0),
		GasLimit:    0,

		Random: &common.Hash{}, // TO ENABLE SHANGHAI FORK
	}
	// ChainConfig
	shanghaiTime := uint64(0)
	chainConfig = params.ChainConfig{
		ChainID:      big.NewInt(1),
		LondonBlock:  big.NewInt(0),
		ShanghaiTime: &shanghaiTime,
	}
	// Config
	config = vm.Config{
		Tracer:                  nil,
		NoBaseFee:               true,
		EnablePreimageRecording: false,
	}
	return statedb, blockCtx, chainConfig, config
}

// PrepareOnchainCache is used to prepare the onchain cache
func PrepareOnchainCache(onchainAddress string) (cacheFolder string, err error) {
	if cacheDir, ok := hasCached(onchainAddress); ok {
		log.Info(fmt.Sprintf("cache find at %s", cacheDir))
		return cacheDir, nil
	}

	// 1. Get CreationTx and ABI
	// 2. From Creationtx, get the deployed bytecode(means with the creation arguments)
	var (
		api     = onchain.ApiKeys()[onchain.ETH]
		eth     = onchain.Chain(onchain.ETH)
		txhash  string
		creator common.Address
		// from          common.Address
		creationCodes []byte
		// to            common.Address
		abi string
	)

	creator, txhash, err = eth.GetCreation(onchainAddress, api)
	if err != nil {
		return "", fmt.Errorf("failed to get contract creation: %w", err)
	}

	abi, err = eth.GetABI(onchainAddress, api)
	if err != nil {
		return "", fmt.Errorf("failed to get contract abi: %w", err)
	}

	_, creationCodes, _, err = eth.GetTx(txhash, api)
	if err != nil {
		return "", fmt.Errorf("failed to get contract transaction: %w", err)
	}

	return saveDeployedBin(onchainAddress, creationCodes, abi, creator.Hex())
}

// Execute is the main entry point for the local fuzz
func Execute(contractFolder string, debug bool) error {
	// Load and resolve contracts
	contracts, err := loadContractsFromDir(contractFolder)
	if err != nil {
		return fmt.Errorf("failed to load contracts: %w", err)
	}
	inherit := NewInheritancer(contracts)
	orphans := inherit.FindInheritance()

	targets := func() []*Contract {
		var res []*Contract
		for _, contract := range contracts {
			if contains(orphans, contract.Name) {
				if len(contract.StaticBin) > 0 || len(contract.CreationBin) > 0 {
					res = append(res, contract) // ignore interface
				}
			}
		}
		return res
	}()

	log.Info("Fuzzing Tasks: ", "orphans", orphans)

	for _, contract := range targets {
		err := execute(contract, debug)
		if err != nil {
			return fmt.Errorf("failed to execute fuzzing on contract %s: %w", contract.Name, err)
		}
	}

	return nil
}

// execute is the internal entry point for a fuzz task
func execute(contract *Contract, debug bool) error {
	// NOTE: Entry point for a fuzzing task
	//  0. Create ERE then deploy
	//  1. Symbolic resolve CFG
	//  2. Fuzzing
	//    2.a Contract Creator as msg.sender, link-generation function sequence
	//    2.b External Address(Attacker[0]) as msg.sender, link-generation function sequence
	//    2.c External Address(Attacker) as msg.sender, R/W guided generation function sequence, enable reentrancy
	//  3. summary:
	//     3.a function branch coverage
	//     3.b total branch coverage and statement coverage
	//     3.c error
	var (
		host      *FuzzHost
		initState int
	)

	defer func() {
		if debug {
			host.Debug()
		}
	}()

	// NOTE: 0 Create ERE then deploy
	fmt.Println("[1/4]Create ERE then deploy")
	statedb, blockCtx, chainConfig, config := CreateEVMRuntimeEnironment()
	host = NewFuzzHost(contract, statedb, blockCtx, chainConfig, config)

	log.Info("Fuzzing contract: ", "name", contract.Name)
	host.RunForDeployOnchain()

	// take a snapshot for the init state
	initState = host.StateDB.Snapshot()

	log.Info("Init state snapshot: ", "snapshot", initState)

	// NOTE: 1 Symbolic resolve CFG
	fmt.Println("[2/4]Symbolic resolve CFG")
	// host.WishSolver.SymbolicResolveCFG()
	host.CFG.SymbolicResolve()

	// NOTE: 2.a Fuzz Stage 1 Single Functional Fuzzing
	// This step try to fuzz each function, collect the coverage
	var wg sync.WaitGroup

	sum := newSummary(host.Scheduler.GetSingleFuncList())
	var (
		throughputTotal    uint64
		throughputSuccess  uint64
		throughputFail     uint64
		throughputMeanning uint64
	)

	start := time.Now()

	var (
		sender1 = host.OwnerAddress
		epoch1  = 1000
	)

	wg.Add(1)
	fmt.Println("[3/4]Start fuzzing each function")
	go func() {
		// for {
		// 	select {
		// 	case <-ctx.Done():
		// 		wg.Done()
		// 		return
		// 	default:
		// func() {
		funcs := host.Scheduler.GetSingleFuncList()
		for i, f := range funcs {
			// NOTE: Loop for each function until timeout or reach full coverage
			cnt := 0

			fmt.Printf(":::[%d/%d] %s ", i+1, len(funcs), f.Name)
			initState = host.StateDB.RevertToInitState(initState)
			for {

				throughputTotal++

				_, funcCover, funcTotal, err := host.FuzzOnce(f, sender1)

				if err == nil && host.Err == nil {
					sum.FunctionBranchCoverage[f.Name] = [2]int{funcCover, funcTotal}
				}

				if err != nil {
					sum.Errors = append(sum.Errors, [2]string{"!" + f.Name, err.Error()})
				} else {
					sum.FunctionBranchCoverage[f.Name] = [2]int{funcCover, funcTotal}
				}

				if host.Err != nil {
					sum.Errors = append(sum.Errors, [2]string{f.Name, host.Err.Error()})
					host.Err = nil
					throughputFail++
				}

				// TODO: measure the meaning value
				throughputMeanning++
				throughputSuccess++

				cnt++
				if cnt > epoch1 {
					break
				}
			}

			fmt.Printf(" [func cov.: %d/%d]\n", sum.FunctionBranchCoverage[f.Name][0], sum.FunctionBranchCoverage[f.Name][1])
		}
		wg.Done()
	}()

	wg.Wait()

	func() {
		duration := time.Since(start)
		sum.Throughput = throughput{
			total:    throughputTotal,
			success:  throughputSuccess,
			fail:     throughputFail,
			meanning: throughputMeanning,
			duration: duration,
		}
		sum.CFGCoverage = host.CFG.CoverageString()
		sum.FunctionAcceessList = host.CFG.AccessList()
		sum.saveToFile(
			"Stage 1 Single Funcs Fuzz",
			fmt.Sprintln(sum.string()),
		)
	}()

	// NOTE: Stage 2 Fuzzing with FuncSequence
	fmt.Println("[2/3] Start fuzzing FuncSequence with no Retreecy")

	var (
		sender2 = host.Attackers[0]
		epoch2  = 1000
	)
	wg.Add(1)
	// timeout2 := time.Duration(300) * time.Second
	// ctx2, cancel2 := context.WithTimeout(context.Background(), timeout2)
	// defer cancel2()

	go func() {
		// for {
		// 	select {
		// 	case <-ctx2.Done():
		// 		wg.Done()
		// 		return
		// 	default:
		step := 1
		for {
			funcs, depth := host.Scheduler.GetFuncsSequence(host.CFG.RWMap())
			if depth > 10 {
				break
			}
			if len(funcs) == 0 {
				break
			}
			// NOTE: we reset statedb to the initial state before execute the func sequence
			initState = host.StateDB.RevertToInitState(initState)
			goodflag := true

			fmt.Printf(":::[%d] %s\n", step, func() string {
				var ret string
				for _, f := range funcs {
					ret += f.Name + ", "
				}
				return ret
			}())

			step++

			for _, f := range funcs {
				// NOTE: each step has a retry limit, if over the limit, we will stop this funcs sequence and mark it as BadFuncs()
				retry := epoch2
				halt := true

				for retry > 0 {

					throughputTotal++
					_, funcCover, funcTotal, err := host.FuzzOnce(f, sender2)
					// fmt.Println("funcCover: ", funcCover, "funcTotal: ", funcTotal, "err: ", err, "host.Err: ", host.Err)
					if err == nil && host.Err == nil {
						// NOTE: at least executed successfully once
						halt = false
						sum.FunctionBranchCoverage[f.Name] = [2]int{funcCover, funcTotal}
						throughputSuccess++
						throughputMeanning++
						break
					}

					if err != nil {
						sum.Errors = append(sum.Errors, [2]string{"!" + f.Name, err.Error()})
					}

					if host.Err != nil {
						sum.Errors = append(sum.Errors, [2]string{f.Name, host.Err.Error()})
						host.Err = nil
						throughputFail++
					}

					retry--
				}

				if halt {
					goodflag = false
					break
				}

			}

			if goodflag {
				host.Scheduler.GoodFuncs()
			} else {
				host.Scheduler.BadFuncs()
			}
		}
		wg.Done()
	}()

	wg.Wait()

	duration := time.Since(start)
	sum.Throughput = throughput{
		total:    throughputTotal,
		success:  throughputSuccess,
		fail:     throughputFail,
		meanning: throughputMeanning,
		duration: duration,
	}

	sum.CFGCoverage = host.CFG.CoverageString()
	sum.FunctionAcceessList = host.CFG.AccessList()

	sum.saveToFile(
		fmt.Sprintf("Stage 2 Funcs Sequence", "Contract Owner: %s\nDeploy at: %s\nAttacker: %s\n", host.OwnerAddress.Hex(), host.DeployAt.Hex(), host.Attackers[0].Hex()),
		fmt.Sprintln(sum.string()),
		fmt.Sprintln(host.CFG.String()),
		fmt.Sprintln(host.Oracle.HumanReport()),
		fmt.Sprintln(host.Mutator.String()),
	)

	return nil
}
