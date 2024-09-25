package fuzz

import (
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/cfg"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"os"
	"time"
)

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

func (s summary) saveToFile(title string, contents ...string) {
	fileName := fmt.Sprintf("%s-summary_%s.log", title, time.Now().Format("20060102150405"))
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
