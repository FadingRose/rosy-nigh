package oracle

import (
	"errors"
	"fmt"
	"strings"
)

type OracleError error

var ErrDivideByZero OracleError = errors.New("divide by zero")

type OracleHost struct {
	nonce  int
	report map[string][]interface{}
}

func NewOracleHost() *OracleHost {
	return &OracleHost{
		report: make(map[string][]interface{}),
	}
}

func (o *OracleHost) HumanReport() string {
	var report string
	for k, v := range o.report {
		report += fmt.Sprintf("===================================\n%s\n-----------------------------------\n%s\n\n", k, v)
	}
	return report
}

func (o *OracleHost) DivideZeroCheck(model string) {
	// e.g. SMT may give a divide zero report, means there may has potential divide by zero error
	// userList:uint32, -> 14253863
	// changeRate:uint8,_rate -> 212
	// isVip:address,account -> 654328426503321873227290820396529531474655385980
	// transferOwnership:address,newOwner -> 654328426503321873227290820396529531474655385980
	// withdraw:uint256,_amount -> 0
	// div0 -> {
	//   0
	// }
	// mod0 -> {
	//   0
	// }

	patterns := []string{"div0 -> {\n", "mod0 -> {\n"}
	for _, p := range patterns {
		if strings.Contains(model, p) {
			o.assert(ErrDivideByZero, model)
		}
	}
}

func (o *OracleHost) assert(err OracleError, ctx ...interface{}) {
	errNonce := fmt.Errorf("%w-%v", err, o.nonce)
	o.report[errNonce.Error()] = ctx
}
