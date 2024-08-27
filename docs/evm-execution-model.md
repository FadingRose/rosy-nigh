# EVM Execution Model

In general, the procedure of an execution is like this:
1. Prepare `Message`
2. Prepare `EVM` entity to execute the `Message`
3. Execute the `EVM.ApplyMessage()`
4. Get the result from `EVM`

## Transaction and Message

`Transaction` is a message **without** a `Signer`, we can consider it as a raw message.

> `Method` and `Input` means the contract's method and input data.

1. `Method` and `Input` -> `Transaction`
2. `Transaction` with `Signer`, `Basefee` -> `Message`

### Transaction Types

> Further, there re two subtypes of transactions: those which result in message calls and those which result in the creation of new accounts with associated code (known informally as contract creation). 
>
> *Ethereum Yellow Paper*

See [Transaction](../core/types/transaction.go)

## EVM :: ApplyMessage

`ApplyMessage` will execute the message and return the result.
```go
func ApplyMessage(evm *vm.EVM, msg *Message, gp *GasPool) (*ExecutionResult, error) {
	return NewStateTransition(evm, msg, gp).TransitionDb()
}
```

`TransitionDb()` is the core function, it's behavior is like this:
1. Safe Check
2. Buy Gas
3. Burn Gas for the Blockchain
4. Execute Transaction
5. Refund Gas

Details are in the following sections.

### ApplyMessage :: TransitionDb :: Pre EVM Execution

1. **Pre Check**
  1.a Check transaction `Nonce` and `EOA`, see [Tx Validity](../core/state_transition.go:199)
  1.b Make sure that transaction gasFeeCap is greater than the baseFee (post london), see [Gas Fee Cap](../core/state_transition.go:219)
  1.c Check the blob version validity, see [Blob Version](../core/state_transition.go:244)
  1.d Check that the user is paying at least the current blob fee, see [Blob Fee](../core/state_transition.go:260)

2. **Buy Gas** sender's balance must cover the `cost`, see [Buy Gas](../core/state_transition.go:283)
  - `cost` = `msg.GasLimit` * `msg.GasPrice` + `&blobFee`
  - `&blobFee` = `&blobGasUsed` * `BlobBaseFee`
  - `&blobGaseUsed` = `len(msg.BlobHashes)` * `BlobTxBlobGasPerBlob`

3. **Subtract the Intrinsic Gas** some action takes intrinsic gas, see [Subtract Intrinsic Gas](../core/state_transition.go:328)
  3.a **Contract Creation and Homestead Fork Check**:
    - If the transaction is a contract creation and the Homestead fork is active, it starts with `params.TxGasContractCreation = 53000`.
    - Otherwise, it starts with `params.TxGas = 21000`.

  3.b. **Transaction Data**:
    - The length of the data (`dataLen`) is calculated.
    - For each non-zero byte in the data, it adds `params.TxDataNonZeroGasFrontier = 68` or `params.TxDataNonZeroGasEIP2028 = 16` (if EIP-2028 is active).
   - For each zero byte in the data, it adds `params.TxDataZeroGas = 4`.
  3.c **Contract Creation under EIP-3860**:
    - If the transaction is a contract creation and EIP-3860 is active, it adds `rams.InitCodeWordGas = 2` for each word (32 bytes) in the data.
  3.d **Access List**:
    - If an access list is provided, it adds `params.TxAccessListAddressGas = 2400` for each address in the access list.
    - It also adds `params.TxAccessListStorageKeyGas = 1900` for each storage key in the access list.

4. **EIP-4762 Activation Handling**, see [EIP-4762](../docs/eips/eips-4762.md)
    - Adds transaction origin and destination to the access list if EIP-4762 is active.

5. **Overflow Check**, see [Overflow Check](../core/state_transition.go:161)
    - `msg.Value` overflow?
6. **Initialization Code Size Check**
    - Verifies that the init code size does not exceed the maximum allowed size if the Shanghai rules are active and it's a contract creation.
7. **Preparatory Steps for the State Transition**
   - Prepares the state for transition, including `msg.AccessList` preparation and `Transition Storage` reset.

### ApplyMessage :: TransitionDb :: Runtime EVM Execution

EVM distinguish `Create` and `Call`.

See [EVM Interpreter Execution With Symbolic Tracing](./evm-interpreter-execution-with-symbolic-tracing.md)
### ApplyMessage :: TransitionDb :: Post EVM Execution

After EVM execution, refund the remaining gas, see [Refund Gas](../core/state_transition.go:220)

1. **Gas Refund Calculation**:
   - Before EIP-3529, `refund = gasUsed / 2`
   - After EIP-3529, `refund = gasUsed / 5`

2. **Effective Tip Calculation**:
  - If the London rules are in effect, `&EffectiveTip = min(gasTipCap, gasUsed() * gasPrice)`.
  - otherwise, `&EffectiveTip = gasPrice`.

3. **Effective Tip Fee Payment or Skip**:
  3.a Skip payment when `evm.Config.NoBaseFee && msg.GasFeeCap.Sign() == 0 && msg.GasTipCap.Sign() == 0` 
  3.b.1 Otherwise, `fee = gasUsed() * &EffectiveTip`, then `st.evm.Context.Coinbase.AddBalance(fee)`
  3.b.2 Access List Update:
    - If EIP-4762 rules are in effect and the fee is greater than zero, the coinbase is added to the access list for balance gas events.

