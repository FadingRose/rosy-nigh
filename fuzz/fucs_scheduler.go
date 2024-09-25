package fuzz

import (
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/cfg"

	"golang.org/x/exp/rand"
)

type prefix []string

func (p prefix) fork(s string) prefix {
	ret := make(prefix, len(p))
	copy(ret, p)
	ret = append(ret, s)
	return ret
}

type FuncsScheduler struct {
	methods map[string]abi.Method

	length int // visit length

	lives      []prefix // the good funcs sequences for now
	diries     []prefix // use this for init prefix
	lastPrefix prefix
}

func NewScheduler(abi abi.ABI) *FuncsScheduler {
	return &FuncsScheduler{
		methods:    abi.Methods,
		length:     1,
		lives:      make([]prefix, 0),
		diries:     make([]prefix, 0),
		lastPrefix: make(prefix, 0),
	}
}

func (fs *FuncsScheduler) GetFuncsSequence(rwmap *cfg.RWMap) ([]abi.Method, int) {
	if len(fs.diries) == 0 {
		if fs.length == 1 {
			fs.diries = func() []prefix {
				ns := rwmap.Nodes()
				var ret []prefix
				for _, n := range ns {
					nn := []string{n}
					ret = append(ret, prefix(nn))
				}
				return ret
			}()
		} else {
			for _, l := range fs.lives {
				// fmt.Printf(" * length: %d, length of live: %d\n", fs.length, len(l))
				if len(l) < fs.length-1 {
					continue
				}
				for _, n := range rwmap.Nodes() {
					np := l.fork(n)
					fs.diries = append(fs.diries, np)
				}
			}
		}
		fs.length++
	}

	// if lives is empty, return empty
	if len(fs.diries) == 0 {
		return make([]abi.Method, 0), fs.length
	}

	fs.lastPrefix = fs.diries[0]
	fs.diries = fs.diries[1:]
	return fs.prefixToMethods(fs.lastPrefix), fs.length
}

func (fs *FuncsScheduler) getFuncsSequence() []abi.Method {
	// TODO: use funcs scheduler algorithm replace this
	n := rand.Intn(len(fs.methods))

	methods := func() []abi.Method {
		var ret []abi.Method
		for _, method := range fs.methods {
			if method.Name == "" || method.StateMutability == "view" {
				continue
			}

			ret = append(ret, method)
		}
		return ret
	}()

	var funcs []abi.Method
	// random pick n funcs
	for i := 0; i < n-1; i++ {
		index := rand.Intn(len(methods) - 1)
		funcs = append(funcs, methods[index])
	}
	return funcs
}

func (scheduler *FuncsScheduler) GetSingleFuncList() []abi.Method {
	var ret []abi.Method
	for _, method := range scheduler.methods {
		if method.Name == "" || method.StateMutability == "view" {
			continue
		}
		ret = append(ret, method)
	}
	return ret
}

func (scheduler *FuncsScheduler) BadFuncs() {
}

func (scheduler *FuncsScheduler) GoodFuncs() {
	// fmt.Printf(" * GoodFuncs %v\n", scheduler.lastPrefix)
	scheduler.lives = append(scheduler.lives, scheduler.lastPrefix)
}

func (fs *FuncsScheduler) prefixToMethods(prefix prefix) []abi.Method {
	var ret []abi.Method
	for _, name := range prefix {
		if m, ok := fs.methods[name]; !ok {
			panic("unknown method: " + name)
		} else {
			ret = append(ret, m)
		}
	}
	return ret
}
