package fuzz

import (
	"bufio"
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/cfg"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/holiman/uint256"
)

func loadRWMap() *cfg.RWMap {
	// read file rwmap.log line by line
	// parse the line and return the map
	f, err := os.Open("rw.log")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	acclist := make(map[string][]cfg.SlotAccess)
	var (
		this  string
		tmps  = make([]cfg.SlotAccess, 0)
		first = true
	)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, ":") {
			line = strings.TrimSuffix(line, ":")
			if first {
				first = false
			} else {
				acclist[this] = tmps
				tmps = make([]cfg.SlotAccess, 0)
			}
			this = strings.TrimPrefix(line, "|->")
			continue
		}

		line = strings.TrimPrefix(line, "|->")
		parts := strings.Split(line, " ")
		if len(parts) != 4 {
			continue
		}

		var (
			tp    cfg.AccessType
			key   uint256.Int
			value uint256.Int
		)

		if strings.Contains(parts[0], "[R]") {
			tp = cfg.Read
		}

		if strings.Contains(parts[0], "[W]") {
			tp = cfg.Write
		}

		if tp == cfg.Unknown {
			panic(fmt.Sprintf("unknown access type: %v", parts))
		}

		err = key.SetFromHex(parts[1])
		if err != nil {
			log.Fatal(err)
		}
		err = value.SetFromHex(parts[3])
		if err != nil {
			log.Fatal(err)
		}

		tmps = append(tmps, cfg.SlotAccess{
			AccessType: tp,
			Key:        key,
			Value:      value,
		})
	}

	acclist[this] = tmps

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	// return acclist
	return cfg.NewRWMap(acclist)
}

func loadABI() abi.ABI {
	f, err := os.Open("abi.log")
	if err != nil {
		log.Fatal(err)
	}
	abi, err := abi.JSON(f)
	if err != nil {
		log.Fatal(err)
	}
	return abi
}

func printMethods(ms []abi.Method) {
	s := strings.Repeat("*", len(ms))
	// for _, m := range ms {
	// 	// s += m.Name + " "
	// 	s += "*"
	// }
	s += "\n"
	fmt.Printf("%s", s)
}

func TestFuncSchedulerMuptiSequence(t *testing.T) {
	rwmap := loadRWMap()
	abi := loadABI()
	fs := NewScheduler(abi)

	for i := 0; i < 200; i++ {
		ms, _ := fs.GetFuncsSequence(rwmap)
		if i%2 == 0 {
			fs.GoodFuncs()
		}
		printMethods(ms)
	}
}
