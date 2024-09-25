package fuzz

import (
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/cfg"
	"fadingrose/rosy-nigh/core"
	"fadingrose/rosy-nigh/core/state"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fadingrose/rosy-nigh/onchain"
	"fmt"
	"math/big"
	"os"
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

	// timeout := time.Duration(5) * time.Second
	// ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel()

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

	epoch1 := 1000

	wg.Add(1)
	fmt.Println("[1/3] Start fuzzing each function")
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

				_, funcCover, funcTotal, err := host.FuzzOnce(f)

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

	// NOTE: Stage 2 Fuzzing with FuncSequence
	fmt.Println("[2/3] Start fuzzing FuncSequence with no Retreecy")
	epoch2 := 1000
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
					_, funcCover, funcTotal, err := host.FuzzOnce(f)
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
		fmt.Sprintf("Contract Owner: %s\nDeploy at: %s\nAttacker: %s\n", host.OwnerAddress.Hex(), host.DeployAt.Hex(), host.Attackers[0].Hex()),
		fmt.Sprintln(sum.string()),
		fmt.Sprintln(host.CFG.String()),
		fmt.Sprintln(host.Oracle.HumanReport()),
		fmt.Sprintln(host.Mutator.String()),
	)

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

func (s summary) saveToFile(contents ...string) {
	fileName := fmt.Sprintf("summary_%s.log", time.Now().Format("20060102150405"))
	file, err := os.Create(fileName)
	if err != nil {
		log.Error("Failed to create summary file: ", err)
		return
	}
	defer file.Close()

	for _, content := range contents {
		_, err := file.WriteString(content)
		if err != nil {
			log.Error("Failed to write to summary file: ", err)
			return
		}
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
