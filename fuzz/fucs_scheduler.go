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
	// TODO: use funcs scheduler algorithm replace this
	n := rand.Intn(len(scheduler.methods))

	methods := func() []abi.Method {
		var ret []abi.Method
		for _, method := range scheduler.methods {
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
