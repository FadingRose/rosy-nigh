# Rosy Nigh

// TODO: Write a project description

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

## Interpreter


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


# Roadmap

### StateDB

- [ ] Implement StateDB with tries queries



## Progress

Step 0 Pre-Compile

A helper tool for batch compile, see [batch-solidity-compiler](https://github.com/FadingRose/batch-compiler)

