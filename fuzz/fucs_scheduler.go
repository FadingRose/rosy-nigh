package fuzz

import (
	"fadingrose/rosy-nigh/abi"

	"golang.org/x/exp/rand"
)

type FuncsScheduler struct {
	methods map[string]abi.Method
}

func NewScheduler(abi abi.ABI) *FuncsScheduler {
	return &FuncsScheduler{
		methods: abi.Methods,
	}
}

func (scheduler *FuncsScheduler) GetFucsSequence() []abi.Method {
	// HACK: only return govWithdrawEther
	return []abi.Method{scheduler.methods["govWithdrawEther"]}

	// TODO: use funcs scheduler algorithm replace this
	n := rand.Intn(len(scheduler.methods))

	methods := func() []abi.Method {
		var ret []abi.Method
		for _, method := range scheduler.methods {
			if method.Name == "" {
				continue
			}
			ret = append(ret, method)
		}
		return ret
	}()

	var funcs []abi.Method
	// random pick n funcs
	for i := 0; i < n; i++ {
		index := rand.Intn(len(scheduler.methods))

		funcs = append(funcs, methods[index])
	}
	return funcs
}
