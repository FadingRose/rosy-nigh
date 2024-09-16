package service

import "fadingrose/rosy-nigh/core/vm"

type Client interface {
	RegExpand(uint64) (string, error)
	RegOpcode(vm.OpCode) (string, error)
}
