package mutator

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/rand"
)

func bitwidth[SV SeedValue](val SV) int {
	switch any(val).(type) {
	case uint8, int8:
		return 8
	case uint16, int16:
		return 16
	case uint32, int32:
		return 32
	case uint64, int64:
		return 64
	default:
		return 0
	}
}

// Helpers for SeedValue
// TODO is there any better implementation?
func parse[SV SeedValue](val string) (SV, error) {
	var _zero SV

	zero := reflect.New(reflect.TypeOf(_zero)).Elem()

	switch any(_zero).(type) {
	case uint8, uint16, uint32, uint64:
		bw := bitwidth(_zero)
		_val, err := strconv.ParseUint(val, 10, bw)
		if err != nil {
			return _zero, fmt.Errorf("panic parsing %s __size_ptr: %d, val: %s", zero.Type(), bw, val)
		}

		zero.SetUint(_val)

	case int8, int16, int32, int64:
		bw := bitwidth(_zero)
		_val, err := strconv.ParseInt(val, 10, bw)
		if err != nil {
			return _zero, err
		}
		zero.SetInt(_val)

	case [1]byte, [2]byte, [3]byte, [4]byte, [5]byte, [6]byte, [7]byte, [8]byte, [9]byte, [10]byte, [11]byte, [12]byte, [13]byte, [14]byte, [15]byte, [16]byte, [17]byte, [18]byte, [19]byte, [20]byte, [21]byte, [22]byte, [23]byte, [24]byte, [25]byte, [26]byte, [27]byte, [28]byte, [29]byte, [30]byte, [31]byte, [32]byte:
		_val, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return _zero, err
		}
		_val_bytes := []byte{byte(_val)}

		__size_ptr := reflect.TypeOf(zero).Size()
		bits := int(__size_ptr)

		_bytes := make([]byte, bits)
		for i := 0; i < bits; i++ {
			_bytes[i] = _val_bytes[i]
		}

		zero.SetBytes(_bytes)

	case string:
		zero.SetString(val)

	case []byte:
		zero.SetBytes([]byte(val))
	case common.Address:
		addr := common.HexToAddress(val)
		zero.Set(reflect.ValueOf(addr))

	case *big.Int:
		bigint := new(big.Int)
		bigint.SetString(val, 10)
		zero.Set(reflect.ValueOf(bigint))

	case bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return _zero, err
		}
		zero.SetBool(b)

	default:
		panic("invalid SeedValue type")
	}

	_zero = zero.Interface().(SV)
	return _zero, nil
}

// TODO convert this by pointer
func random[SV SeedValue]() SV {
	var _zero SV

	zero := reflect.New(reflect.TypeOf(_zero)).Elem()

	switch any(_zero).(type) {
	case uint8, uint16, uint32, uint64:
		__size_ptr := reflect.TypeOf(zero).Size()
		_val := rand.Intn(int(math.Pow(2, float64(__size_ptr))))
		zero.SetUint(uint64(_val))

	case int8, int16, int32, int64:
		__size_ptr := reflect.TypeOf(zero).Size()
		_val := rand.Intn(int(math.Pow(2, float64(__size_ptr))))
		zero.SetInt(int64(_val))

	case [1]byte, [2]byte, [3]byte, [4]byte, [5]byte, [6]byte, [7]byte, [8]byte, [9]byte, [10]byte, [11]byte, [12]byte, [13]byte, [14]byte, [15]byte, [16]byte, [17]byte, [18]byte, [19]byte, [20]byte, [21]byte, [22]byte, [23]byte, [24]byte, [25]byte, [26]byte, [27]byte, [28]byte, [29]byte, [30]byte, [31]byte, [32]byte:
		__size_ptr := reflect.TypeOf(zero).Size()
		bits := int(__size_ptr / 8)

		_bytes := make([]byte, bits)

		for i := 0; i < bits; i++ {
			_bytes[i] = byte(rand.Intn(256))
		}

		for i := 0; i < bits; i++ {
			zero.Index(i).Set(reflect.ValueOf(_bytes[i]))
		}
		// zero.SetBytes(_bytes)

	case string:
		// randomize string length
		_len := rand.Intn(32)
		zero.SetString(string(randStringBytes(_len)))

	case []byte:
		_len := rand.Intn(32)
		zero.SetBytes(randStringBytes(_len))

	case common.Address:
		addr := common.BytesToAddress(randBytes(20))

		// 20 means AddressLength
		for i := 0; i < 20; i++ {
			zero.Index(i).Set(reflect.ValueOf(addr[i]))
		}
		// zero.Set(reflect.ValueOf(addr))

	case *big.Int:
		bigint := big.NewInt(rand.Int63())
		zero.Set(reflect.ValueOf(bigint))

	case bool:
		b := rand.Intn(2) == 1
		zero.SetBool(b)

	default:
		panic("invalid SeedValue type : " + reflect.TypeOf(zero).Name())
	}

	_zero = zero.Interface().(SV)
	return _zero
}

func hash[SV SeedValue](val SV) []byte {
	_val := reflect.ValueOf(val)
	if _val.Kind() == reflect.Ptr {
		// *big.Int
		if _val.Type().Elem().String() == "big.Int" {
			return _val.Interface().(*big.Int).Bytes()
		}
		panic("hasher: unhandle pointer type")
	}

	var zero SV
	switch any(zero).(type) {
	case *big.Int:
		return _val.Interface().(*big.Int).Bytes()
	case common.Address:
		return _val.Interface().(common.Address).Bytes()
	case string:
		return []byte(_val.String())
	case uint8, uint16, uint32, uint64:
		return []byte(strconv.FormatUint(_val.Uint(), 10))
	case int8, int16, int32, int64:
		return []byte(strconv.FormatInt(_val.Int(), 10))
	default:
		return _val.Bytes()
	}
}

func toBigInt[SV SeedValue](val SV) *big.Int {
	var _zero SV

	var ret *big.Int

	switch any(_zero).(type) {
	case uint8, uint16, uint32, uint64:
		ret = big.NewInt(int64(reflect.ValueOf(val).Uint()))
	case int8, int16, int32, int64:
		ret = big.NewInt(reflect.ValueOf(val).Int())
	case [1]byte, [2]byte, [3]byte, [4]byte, [5]byte, [6]byte, [7]byte, [8]byte, [9]byte, [10]byte, [11]byte, [12]byte, [13]byte, [14]byte, [15]byte, [16]byte, [17]byte, [18]byte, [19]byte, [20]byte, [21]byte, [22]byte, [23]byte, [24]byte, [25]byte, [26]byte, [27]byte, [28]byte, [29]byte, [30]byte, [31]byte, [32]byte:
		ret = new(big.Int).SetBytes(reflect.ValueOf(val).Bytes())
	case string:
		ret = new(big.Int).SetBytes([]byte(reflect.ValueOf(val).String()))
	case common.Address:
		addr := reflect.ValueOf(val).Interface().(common.Address)
		ret = new(big.Int).SetBytes(addr.Bytes())
		// ret = new(big.Int).SetBytes(reflect.ValueOf(val).Bytes())
	case *big.Int:
		ret = new(big.Int).Set(reflect.ValueOf(val).Interface().(*big.Int))
	case bool:
		ret = new(big.Int).SetInt64(reflect.ValueOf(val).Int())
	case []byte:
		ret = new(big.Int).SetBytes(reflect.ValueOf(val).Bytes())
	default:
		panic("invalid SeedValue type: " + reflect.TypeOf(val).Name())
	}

	return ret
}

func randStringBytes(n int) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return b
}
