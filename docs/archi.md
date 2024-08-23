# Rosy Nigh

// TODO: Write a project description

## Fuzz Host

A fuzz host holds these components and middleware:
- Function Sequences & Corpus
- Z3 Solver Adapter
- Basic Block Coverage Tracer & CFG
- Symbolic Interpreter, Shadow Memory Storage, and StateDB
- Test Oracle
- Online Fuzzing Adapter

## EVM Runtime Environment

### StateDB 

`StateDB` holds all the account states and storage states of the current EVM runtime environment.

see [StateDB interface](../core/vm/interface.go)


## Progress

Step 0 Pre-Compile

A helper tool for batch compile, see [batch-solidity-compiler](https://github.com/FadingRose/batch-compiler)

// TODO accompile this
