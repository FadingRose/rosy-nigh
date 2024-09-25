# Rosy Nigh

- [ ] 实现对于 Slot 和 Memory 的追踪与求解

For EVM execution details, see [EVM Execution Model](./evm-execution-model.md)

## Fuzz Host

A fuzz host holds these components and middleware:
- EVM Runtime Environment
- Function Sequences & Corpus
- Z3 Solver Adapter
- Basic Block Coverage Tracer & CFG
- Symbolic Interpreter, Shadow Memory Storage, and StateDB
- Test Oracle
- Online Fuzzing Adapter

## EVM Runtime Environment

> Ethereum Runtime Environment: (aka ERE) The environment which is provided to an Autonomous Object executing in the EVM. Includes the EVM but also the structure of the world state on which the EVM relies for certain I/O instructions including CALL & CREATE.

So the ERE means EVM + World State.

### StateDB, the implementation of World State

The world state is a mapping from addresses to account states, where an account state is a tuple of four items:
1. **nonce:** a counter used to make sure each transaction can only be processed once.
2. **balance:** the amount of ether(Wei) owned by this account.
3. **storageRoot:** the root of the trie that encodes the storage contents of this account.
4. **codeHash:** the hash of the EVM code of this account -- this is the code that gets executed should this account receive a message call.

> See [StateDB interface](../core/vm/interface.go)

Rosy-Nigh supports on-chain fuzzing by supports StateDB with cache.

> See [On-Chain DB](../onchain/onchain.go)

## Z3 Solver Adapter

For any `JUMPI`, tree-expand all instructions (referred to as `relies`) with the following characteristics to be processed:

1. Bound to a specific actual argument / MagicNumber?
2. Dependent on a specific Memory? Slot? Location?

For 1., SMT can solve it directly. For 2., consider the following discussions:

1. Does the sequence contain a write to this location? If so, attempt to solve it.
2. Does the sequence contain a read from this location? If so, attempt to solve it.

We aim to transform all 2. problems into 1. problems.
# Roadmap

### StateDB

- [ ] Implement StateDB with tries queries



## Progress

Step 0 Pre-Compile

A helper tool for batch compile, see [batch-solidity-compiler](https://github.com/FadingRose/batch-compiler)

Step 1 Single Function Fuzzing

A pre fuzzing step, using Contract Creator as message sender

Step 2 Multi-Function Fuzzing

Real fuzzing, using a external attacker as message sender

