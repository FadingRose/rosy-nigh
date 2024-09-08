package fuzz

import "fadingrose/rosy-nigh/abi"

type FucsSequence struct {
	fucs []FucsImpl
}

type FucsImpl struct {
	methood    abi.Method
	arguments  []interface{}
	accessList []SlotAccess
}
