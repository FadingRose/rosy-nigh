package mutator

import (
	"fadingrose/rosy-nigh/abi"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/rand"
)

type RandomMethodPicker struct {
	MethodsList []string
}
type RandomArgsGenerator struct {
	Methods map[string]abi.Method
}

func NewRandomArgsGenerator(abi abi.ABI) *RandomArgsGenerator {
	return &RandomArgsGenerator{
		Methods: abi.Methods,
	}
}

func NewRandomMethodPicker(abi abi.ABI) *RandomMethodPicker {
	methods := make([]string, 0, len(abi.Methods))
	for name := range abi.Methods {
		if name == "" {
			continue // avoid constructor
		}
		methods = append(methods, name)
	}

	return &RandomMethodPicker{
		MethodsList: methods,
	}
}

func (r *RandomMethodPicker) Pick() (name string) {
	name = r.MethodsList[rand.Intn(len(r.MethodsList))]
	return
}

// Type enumerator
// const (
//
//	IntTy byte = iota
//	UintTy
//	BoolTy
//	StringTy
//	SliceTy
//	ArrayTy
//	TupleTy
//	AddressTy
//	FixedBytesTy
//	BytesTy
//	HashTy
//	FixedPointTy
//	FunctionTy
//
// )

// []interface{} -> data
// abi.Arguments -> format
func (r *RandomArgsGenerator) Generate(method abi.Method) ([]interface{}, abi.Arguments) {
	inputs := method.Inputs
	args := make([]interface{}, 0, len(inputs))

	for _, input := range inputs {
		tp := input.Type.T // SolidityType
		size := input.Type.Size
		var value interface{}

		switch tp {
		case abi.IntTy:
			value = randInt(false, size)

		case abi.UintTy:
			value = randInt(true, size)

		case abi.BoolTy:
			value = rand.Intn(2) == 1

		case abi.StringTy:
			value = randString(10)

		case abi.SliceTy:
			value = randBytes(10)

		case abi.ArrayTy:
			value = randBytes(10)

		case abi.TupleTy:
			value = randBytes(10)

		case abi.AddressTy:
			value = randAddress()

		case abi.FixedBytesTy:
			value = randFixedBytes(size)

		case abi.BytesTy:
			value = randBytes(10)

		case abi.HashTy:
			value = randBytes(10)

		case abi.FixedPointTy:
			value = randBytes(10)

		case abi.FunctionTy:
			value = randBytes(10)

		default:
			panic("Unknown solidity type from abi.method")
		}
		args = append(args, value)
	}
	return args, inputs
}

func randerInt[T Integer](unsigned bool, size int) func(...int) T {
	return func(...int) T {
		return randInt(unsigned, size).(T)
	}
}

func randerBigInt() func(...int) *big.Int {
	return func(...int) *big.Int {
		ret := randInt(false, 64).(int64)
		return big.NewInt(ret)
	}
}

func randerBool() func(...int) uint8 {
	return func(...int) uint8 {
		if rand.Intn(2) == 1 {
			return 1
		}
		return 0
	}
}

func randerString() func(...int) string {
	return func(b ...int) string {
		if len(b) > 0 {
			return string(b[0])
		}
		return randString(10)
	}
}

func randerBytes() func(...int) []byte {
	return func(...int) []byte {
		return randBytes(10)
	}
}

func randerAddress() func(...int) common.Address {
	return func(...int) common.Address {
		return randAddress()
	}
}

func randAddress() common.Address {
	// HACK return only zero
	// return common.BytesToAddress(zeroBytes(20))
	return common.BytesToAddress(randBytes(20))
}

func randerFixedBytes[T FixedBytes](n int) func(...int) T {
	return func(fixed ...int) T {
		if len(fixed) <= 0 {
			panic("FixedBytes size is not specified")
		}

		n := fixed[0]

		if n <= 0 || n > 32 {
			panic("FixedBytes size is not valid")
		}

		switch n {
		case 1:
			return randFixedBytes(1).(T)
		}

		panic("FixedBytes size is not valid")
	}
}

func randInt(unsigned bool, size int) interface{} {
	if unsigned {
		switch size {
		case 8:
			return uint8(rand.Intn(256))
		case 16:
			return uint16(rand.Intn(65536))
		case 32:
			return uint32(rand.Uint32())
		case 64:
			return uint64(rand.Uint64())
		}
	}

	switch size {
	case 8:
		return int8(rand.Intn(256))
	case 16:
		return int16(rand.Intn(65536))
	case 32:
		return int32(rand.Int31())
	case 64:
		return int64(rand.Int63())
	}

	return big.NewInt(rand.Int63())
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// byte1 ... byte32
// byte is a alias for byte1
func randFixedBytes(n int) interface{} {
	arrType := reflect.ArrayOf(n, reflect.TypeOf(byte(0)))
	arrVal := reflect.New(arrType).Elem()
	for i := 0; i < n; i++ {
		arrVal.Index(i).SetUint(uint64(rand.Intn(256)))
	}

	return arrVal.Interface()
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return b
}

func zeroBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(0)
	}
	return b
}
