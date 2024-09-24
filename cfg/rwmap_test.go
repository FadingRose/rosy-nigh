package cfg

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/holiman/uint256"
)

func loadRWMap() map[string][]SlotAccess {
	// read file rwmap.log line by line
	// parse the line and return the map
	f, err := os.Open("rw.log")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	rwmap := make(map[string][]SlotAccess)
	var (
		this  string
		tmps  = make([]SlotAccess, 0)
		first = true
	)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, ":") {
			line = strings.TrimSuffix(line, ":")
			if first {
				first = false
			} else {
				rwmap[this] = tmps
				tmps = make([]SlotAccess, 0)
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
			tp    AccessType
			key   uint256.Int
			value uint256.Int
		)

		if strings.Contains(parts[0], "[R]") {
			tp = Read
		}

		if strings.Contains(parts[0], "[W]") {
			tp = Write
		}

		if tp == Unknown {
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

		tmps = append(tmps, SlotAccess{
			AccessType: tp,
			Key:        key,
			Value:      value,
		})
	}

	rwmap[this] = tmps

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return rwmap
}

func TestRWMap(t *testing.T) {
	raw := loadRWMap()
	rwm := NewRWMap(raw)
	fmt.Println(rwm.String())
}

func TestEntriesFind(t *testing.T) {
	raw := loadRWMap()
	rwm := NewRWMap(raw)
	fmt.Println(rwm.String())
	entries := rwm.entries()
	for _, e := range entries {
		fmt.Println(e.name)
	}
}

func TestVisit(t *testing.T) {
	raw := loadRWMap()
	rwm := NewRWMap(raw)
	for i := 0; i < 10; i++ {
		fmt.Println(rwm.Visit(10))
	}
}
