package mutator

import (
	"sort"
	"strings"
)

type (
	SeedMap     = map[HashSeed]Seed
	SeedHistory = map[HashSeed]uint64
)

type Vault struct {
	Seeds    []Seed
	baseSeed Seed

	SeedMap
	SeedHistory

	// if the Vault stores argument, the name is the Unique Name
	// of a contract level namespace
	// <MethodName>:<Type>_<ArgumentName>
	// if <MethodName> == "", means the `constructor`
	name string
}

// To bind any BaseSeed to the Vault, please use
//
//		var vt Vault
//		_seed := NewSeedImpl[...](&vt)
//		vt.IntiValut(_seed)
//	 ** MAKE SURE BaseSeed's *vault point's to RIGHT Vault' **
func (v *Vault) InitVault(_base Seed, _name string) {
	if _base == nil {
		panic("Vault.InitVault: BaseSeed is nil")
	}

	v.Seeds = make([]Seed, 0)
	v.SeedMap = make(SeedMap)
	v.SeedHistory = make(SeedHistory)

	v.baseSeed = _base
	v.name = _name
}

func (v *Vault) Inherit(val string) {
	// Ensure the seed is unique by checking its hash against the selectHistory map
	if _seed, err := v.baseSeed.Parse(val); err != nil {
	} else {
		v.addSeedWithHashCheck(_seed)
	}
}

// Add random seeds
func (v *Vault) Randomize(n int) {
	for n > 0 {
		n--
		_seed := v.baseSeed.Random()
		v.addSeedWithHashCheck(_seed)
	}
}

func (v *Vault) GetSeed() Seed {
	if len(v.Seeds) == 0 {
		v.Randomize(1)
		// return v.baseSeed
	}

	// ignores all the priority == -1000

	// Sort by priority
	sort.Slice(v.Seeds, func(i, j int) bool {
		return v.Seeds[i].Priority() > v.Seeds[j].Priority()
	})

	return v.Seeds[0]
}

func (v *Vault) addSeedWithHashCheck(seed Seed) {
	hash := seed.Hash()

	if _seed, ok := v.SeedMap[hash]; ok {
		_seed.IncreasePriority()
	} else {
		v.Seeds = append(v.Seeds, seed)
		v.SeedMap[hash] = seed
		v.SeedHistory[hash] = 0
	}
}

func (v *Vault) Name() string {
	return v.name
}

func (v *Vault) Size() int {
	return len(v.Seeds)
}

func (v *Vault) Format() string {
	var ret strings.Builder
	for _, s := range v.Seeds {
		ret.WriteString(s.Format() + "\n")
	}
	return ret.String()
}

func (v *Vault) String() string {
	if len(v.Seeds) == 0 {
		return v.baseSeed.Format()
	}

	// ignores all the priority == -1000

	// Sort by priority
	sort.Slice(v.Seeds, func(i, j int) bool {
		return v.Seeds[i].Priority() > v.Seeds[j].Priority()
	})

	var ret strings.Builder
	cutoff := 5
	cutoffStr := "..."
	if len(v.Seeds) < cutoff {
		cutoffStr = ""
	}

	for _, s := range v.Seeds {
		ret.WriteString(s.Format() + "\n")
	}

	ret.WriteString(cutoffStr)

	return ret.String()
}
