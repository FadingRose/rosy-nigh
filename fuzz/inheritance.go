package fuzz

import "fadingrose/rosy-nigh/abi"

// Find Inheritance from ABIs
// Consider contract A and B:
// IF A is-a B, B is-not-a A:
// 1. A has all B's functions and events
// 2. A has more functions and events

// IF A is-a B , B is-a A:
// 1. A has all B's functions and events
// 2. B has all A's functions and events

// Find which is the tartget contract to fuzz
type Inheritance int

const (
	IS_A Inheritance = iota
	EQ
	None
)

func (i Inheritance) String() string {
	switch i {
	case IS_A:
		return "is-a"
	case EQ:
		return "eq"
	case None:
		return "none"
	}
	return ""
}

type InheritMap = map[string][]InheritPair

type Inheritancer struct {
	NameMap    map[*abi.ABI]string
	ABIs       []*abi.ABI
	InheritMap InheritMap
	// map[contractA] -> [contractB, Inheritance]
}

type InheritPair struct {
	A           string
	B           string
	Inheritance Inheritance
}

func NewInheritancer(contracts []*Contract) *Inheritancer {
	return &Inheritancer{
		NameMap: func() map[*abi.ABI]string {
			res := make(map[*abi.ABI]string)
			for _, c := range contracts {
				res[&c.ABI] = c.Name
			}
			return res
		}(),
		ABIs: func() []*abi.ABI {
			var res []*abi.ABI
			for _, c := range contracts {
				res = append(res, &c.ABI)
			}
			return res
		}(),
		InheritMap: make(InheritMap),
	}
}

// FindInheritance find the inheritance relationship between contracts
// Returns a list of contracts that MOST inherit from others
func (i *Inheritancer) FindInheritance() []string {
	// construct a 2-dimensional map
	for _, a := range i.ABIs {
		for _, b := range i.ABIs {
			if a.IsSame(*b) {
				i.InheritMap[i.NameMap[a]] = append(i.InheritMap[i.NameMap[a]], InheritPair{
					A:           i.NameMap[a],
					B:           i.NameMap[b],
					Inheritance: EQ,
				})
			} else if i.isA(a, b) {
				i.InheritMap[i.NameMap[a]] = append(i.InheritMap[i.NameMap[a]], InheritPair{
					A:           i.NameMap[a],
					B:           i.NameMap[b],
					Inheritance: IS_A,
				})
			} else {
				i.InheritMap[i.NameMap[a]] = append(i.InheritMap[i.NameMap[a]], InheritPair{
					A:           i.NameMap[a],
					B:           i.NameMap[b],
					Inheritance: None,
				})
			}
		}
	}

	// the final contract X follow the rule:
	// 1. For each of all other contract Y, inheritMap[Y] -> [X] == none OR eq
	res := collectVals[*abi.ABI, string](i.NameMap)

	for _, v := range i.InheritMap {
		for _, pair := range v {
			// if pair.Inheritance == is-a, then remove pair.B
			if pair.Inheritance == IS_A {
				res = removeAny(res, pair.B, func(a, b string) bool {
					return a == b
				})
			}
		}
	}

	return res
}

func (i *Inheritancer) isA(a, b *abi.ABI) bool {
	methodSame := func(a, b abi.Method) bool {
		return a.IsSame(b)
	}
	eventSame := func(a, b abi.Event) bool {
		return a.IsSame(b)
	}
	errSame := func(a, b abi.Error) bool {
		return a.IsSame(b)
	}
	// a has all b's functions and events
	for _, bFunc := range b.ListMethods() {
		if !containsAny(a.ListMethods(), bFunc, methodSame) {
			return false
		}
	}
	for _, bEvent := range b.ListEvents() {
		if !containsAny(a.ListEvents(), bEvent, eventSame) {
			return false
		}
	}
	for _, bErr := range b.ListErrors() {
		if !containsAny(a.ListErrors(), bErr, errSame) {
			return false
		}
	}
	return true
}

func (i *Inheritancer) Format() string {
	res := ""
	for k, v := range i.InheritMap {
		res += k + ":\n"
		for _, pair := range v {
			res += "\t" + pair.B + " " + pair.Inheritance.String() + "\n"
		}
	}
	return res
}
