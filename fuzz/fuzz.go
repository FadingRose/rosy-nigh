package fuzz

import (
	"context"
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/cfg"
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

// NOTE: Entry point for a fuzzing task
//  0. set timeout, deploy
//  1. get funcs sequence from scheduler
//  2. for each func in funcs, run fuzz
//  3. summary:
//     3.a function branch coverage
//     3.b total branch coverage and statement coverage
//     3.c error
func execute(contract *Contract, debug bool) error {
	var (
		host      *FuzzHost
		initState int
	)

	defer func() {
		if debug {
			host.Debug()
		}
	}()

	statedb, blockCtx, chainConfig, config := CreateEVMRuntimeEnironment()
	host = NewFuzzHost(contract, statedb, blockCtx, chainConfig, config)

	log.Info("Fuzzing contract: ", "name", contract.Name)
	host.RunForDeployOnchain()

	// take a snapshot for the init state
	initState = host.StateDB.Snapshot()

	log.Info("Init state snapshot: ", "snapshot", initState)

	timeout := time.Duration(5) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sum := newSummary(host.Scheduler.GetSingleFuncList())

	var wg sync.WaitGroup

	var (
		throughputTotal    uint64
		throughputSuccess  uint64
		throughputFail     uint64
		throughputMeanning uint64
	)

	start := time.Now()

	// NOTE: Stage 1 Single Functional Fuzzing
	// This step try to fuzz each function, collect the coverage
	wg.Add(1)
	// TODO: thoughout
	// success call per second
	go func() {
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return
			default:
				func() {
					funcs := host.Scheduler.GetSingleFuncList()
					for _, f := range funcs {
						// NOTE: Loop for each function until timeout or reach full coverage
						cnt := 0

						initState = host.StateDB.RevertToInitState(initState)
						for {

							throughputTotal++

							_, funcCover, funcTotal, err := host.FuzzOnce(f)

							if err != nil {
								sum.Errors = append(sum.Errors, [2]string{"!" + f.Name, err.Error()})
								break
							} else {
								sum.FunctionBranchCoverage[f.Name] = [2]int{funcCover, funcTotal}
							}

							if host.Err != nil {
								sum.Errors = append(sum.Errors, [2]string{f.Name, host.Err.Error()})
								host.Err = nil
								throughputFail++
								break
							}
							// TODO: measure the meaning value
							throughputMeanning++
							throughputSuccess++

							if funcCover == funcTotal {
								break
							}

							cnt++
							if cnt > 100 {
								break
							}
						}
					}
				}()
			}
		}
	}()

	wg.Wait()

	// NOTE: Stage 2 Fuzzing with FuncSequence
	wg.Add(1)
	timeout2 := time.Duration(300) * time.Second
	ctx2, cancel2 := context.WithTimeout(context.Background(), timeout2)
	defer cancel2()

	go func() {
		for {
			select {
			case <-ctx2.Done():
				wg.Done()
				return
			default:
				func() {
					funcs, _ := host.Scheduler.GetFuncsSequence(host.CFG.RWMap())
					fmt.Println("funcs: ", funcs)
					// for a specific func sequence [A->B->C], assume that we have execute A and B, then we try to go through C
					initState = host.StateDB.RevertToInitState(initState)

					for _, f := range funcs {
						retry := 10
						halt := false

						for retry > 0 {
							throughputTotal++
							// WARN: should we reset statedb to the initial state?
							_, funcCover, funcTotal, err := host.FuzzOnce(f)

							// fmt.Println("fuzzing: ", f.Name, funcCover, funcTotal, err, host.Err)

							if err == nil && host.Err == nil {
								sum.FunctionBranchCoverage[f.Name] = [2]int{funcCover, funcTotal}
								halt = false
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
							// DONE: if this funcs sequence is NOT a good one, let Scheduler knows
							// log.Info("Bad Funcs ", "funcs", funcs)
							host.Scheduler.BadFuncs()
							break
						}

						host.Scheduler.GoodFuncs()
					}
				}()
			}
		}
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

	fmt.Printf("Contract Owner: %s\nDeploy at: %s\nAttacker: %s\n", host.OwnerAddress.Hex(), host.DeployAt.Hex(), host.Attackers[0].Hex())

	fmt.Println(sum.string())

	fmt.Println(host.CFG.String())

	fmt.Println(host.Oracle.HumanReport())

	fmt.Println(host.Mutator.String())

	return nil
}

type throughput struct {
	total    uint64 // total calls
	success  uint64 // calls with Return
	fail     uint64 // calls without Return, but Revert error
	meanning uint64 // calls with Return, and reach the target

	duration time.Duration
}

type summary struct {
	FunctionBranchCoverage map[string][2]int
	FunctionAcceessList    map[string][]cfg.SlotAccess
	CFGCoverage            string
	Errors                 [][2]string
	Throughput             throughput
}

func newSummary(funcs []abi.Method) summary {
	fbc := func() map[string][2]int {
		ret := make(map[string][2]int)
		for _, f := range funcs {
			ret[f.Name] = [2]int{-1, 0}
		}
		return ret
	}()

	return summary{
		FunctionBranchCoverage: fbc,
		FunctionAcceessList:    make(map[string][]cfg.SlotAccess),
		CFGCoverage:            "",
		Errors:                 make([][2]string, 0),
		Throughput: throughput{
			total:    0,
			success:  0,
			fail:     0,
			meanning: 0,
			duration: 0,
		},
	}
}

func (s summary) string() string {
	errStr := ""
	for _, parts := range s.Errors {
		errStr += "| " + parts[0] + "->" + parts[1] + "\n"
	}
	funcCoverageStr := ""
	for name, coverage := range s.FunctionBranchCoverage {
		funcCoverageStr += fmt.Sprintf("|->%s: %d/%d\n", name, coverage[0], coverage[1])
	}
	funcSlotAccessStr := ""
	for name, accessList := range s.FunctionAcceessList {
		funcSlotAccessStr += "|->" + name + ":\n"
		// readlistStr := ""
		// writeListStr := ""
		last := ""
		for _, access := range accessList {
			if last == access.String() {
				continue
			}
			last = access.String()
			funcSlotAccessStr += fmt.Sprintf("|->%s\n", access.String())
			// if access.AccessType == cfg.Read {
			// 	readlistStr += fmt.Sprintf("|->%s\n", access.String())
			// } else {
			// 	writeListStr += fmt.Sprintf("|->%s\n", access.String())
			// }
		}
	}
	throughputStr := ""
	if s.Throughput.total > 0 {
		totalQps := float64(s.Throughput.total) / s.Throughput.duration.Seconds()
		sucQps := float64(s.Throughput.success) / s.Throughput.duration.Seconds()
		meanQps := float64(s.Throughput.meanning) / s.Throughput.duration.Seconds()
		throughputStr = fmt.Sprintf("|->Total: %d, Success: %d, Fail: %d, Meanning: %d\n|->QPS: %.2f, SuccessQPS: %.2f, MeanningQPS: %.2f\n", s.Throughput.total, s.Throughput.success, s.Throughput.fail, s.Throughput.meanning, totalQps, sucQps, meanQps)
	}
	return fmt.Sprintf("> Throughput:\n%s\n> FunctionBranchCoverage:\n%v\n> CFGCoverage: %s> FunctionSlotAccessList:\n%s\n> Errors:\n%v", throughputStr, funcCoverageStr, s.CFGCoverage, funcSlotAccessStr, errStr)
}
