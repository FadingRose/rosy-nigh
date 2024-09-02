package mutator

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// TODO: Provies a 'WRiteBack' Method for SMT solver
// Vault holds a set of seeds can be use
type Seed interface {
	IncreasePriority()
	Priority() int
	DecreasePriority()

	Val() interface{}

	Parse(val string) (Seed, error)
	Random() Seed
	Hash() HashSeed
	Drop() (string, *big.Int)
	Format() string
}

// These three interfaces contains the types that SeedValue can be
type Integer interface {
	uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64
}

type FixedBytes interface {
	[1]byte | [2]byte | [3]byte | [4]byte | [5]byte | [6]byte | [7]byte | [8]byte | [9]byte | [10]byte | [11]byte | [12]byte | [13]byte | [14]byte | [15]byte | [16]byte | [17]byte | [18]byte | [19]byte | [20]byte | [21]byte | [22]byte | [23]byte | [24]byte | [25]byte | [26]byte | [27]byte | [28]byte | [29]byte | [30]byte | [31]byte | [32]byte
}

type OtherSeedValue interface {
	string | []byte | common.Address | *big.Int | bool
}

type SeedValue interface {
	Integer | FixedBytes | OtherSeedValue
}

// type SeedValue[T INTS|string|[]byte|common.Address|FIXEDBYTES]

type HashSeed = [32]byte

type SeedImpl[SV SeedValue] struct {
	val      SV
	priority int

	fixed int // -1 means not fixed-lenwgth

	// parser func(string) (Seed, error)
	// random func() Seed
	// hasher func(V) HashSeed
	vault *Vault
}

func NewSeedImpl[SV SeedValue](vault *Vault) *SeedImpl[SV] {
	if vault == nil {
		panic("vault is nil when creating new seed")
	}

	_seed := &SeedImpl[SV]{
		val:      random[SV](),
		priority: 0,

		fixed: 0,

		vault: vault,
	}

	return _seed
}

func (si *SeedImpl[SV]) Val() interface{} {
	return si.val
}

func (si *SeedImpl[SV]) IncreasePriority() {
	si.priority++
}

func (si *SeedImpl[SV]) Priority() int {
	return si.priority
}

func (si *SeedImpl[SV]) DecreasePriority() {
	si.priority--
}

func (si *SeedImpl[SV]) Instance() *SeedImpl[SV] {
	return si
}

func (si *SeedImpl[SV]) Parse(_val string) (Seed, error) {
	if sv, err := parse[SV](_val); err != nil {
		return nil, err
	} else {
		return &SeedImpl[SV]{
			val:      sv,
			priority: 0,
			fixed:    0,
			vault:    si.vault,
		}, nil
	}
}

func (si *SeedImpl[SV]) Random() Seed {
	sv := random[SV]()
	return &SeedImpl[SV]{
		val:      sv,
		priority: 0,
		fixed:    0,
		vault:    si.vault,
	}
}

// VaultFullName + Val -> Hash
func (si *SeedImpl[SV]) Hash() HashSeed {
	hasher := sha256.New()

	if si.vault == nil {
		panic(" * Seed's Vault is nil, make sure the BaseSeed and Inherit Seed are correctly set.")
	}
	hasher.Write([]byte(si.vault.Name()))

	_bin := hash[SV](si.val)
	hasher.Write(_bin)

	hashBytes := hasher.Sum(nil)
	var hashSeed HashSeed
	copy(hashSeed[:], hashBytes)
	return hashSeed
}

// Drop() returns fullname and value to *big.Int
// See z3.go Exclusion struct
func (si *SeedImpl[SV]) Drop() (string, *big.Int) {
	if si.priority > -1000 {
		si.priority = -1000
	} else {
		si.priority--
	}
	name := si.vault.Name()
	ret := toBigInt[SV](si.val)
	return name, ret
}

func (si *SeedImpl[SV]) Format() string {
	return fmt.Sprintf("[%d] %s -> %v", si.priority, si.vault.name, si.val)
}
