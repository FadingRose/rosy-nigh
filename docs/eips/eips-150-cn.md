```markdown
---
eip: 150
标题: 针对I/O密集型操作的Gas成本调整
作者: Vitalik Buterin (@vbuterin)
类型: 标准轨道
类别: 核心
状态: 最终
创建时间: 2016-09-24
---

### 元参考

[Tangerine Whistle](./eip-608.md).

### 参数

|   FORK_BLKNUM   |  CHAIN_ID  | CHAIN_NAME  |
|-----------------|------------|-------------|
|    2,463,000    |     1      | 主网       |

### 规范

如果 `block.number >= FORK_BLKNUM`，则：
- 将EXTCODESIZE的Gas成本增加到700（原为20）。
- 将EXTCODECOPY的基础Gas成本增加到700（原为20）。
- 将BALANCE的Gas成本增加到400（原为20）。
- 将SLOAD的Gas成本增加到200（原为50）。
- 将CALL、DELEGATECALL、CALLCODE的Gas成本增加到700（原为40）。
- 将SELFDESTRUCT的Gas成本增加到5000（原为0）。
- 如果SELFDESTRUCT命中一个新创建的账户，则触发额外的25000 Gas成本（类似于CALL）。
- 将推荐的Gas上限目标增加到550万。
- 定义“除一个64分之一外的所有”`N`为`N - floor(N / 64)`。
- 如果一个调用请求的Gas超过了最大允许量（即在父级中减去调用和内存扩展的Gas成本后剩余的总量），则不返回OOG错误；相反，如果一个调用请求的Gas超过最大允许量的除一个64分之一外的所有，则使用最大允许量的除一个64分之一外的所有Gas进行调用（这相当于EIP-90<sup>[1](https://github.com/ethereum/EIPs/issues/90)</sup>加上EIP-114<sup>[2](https://github.com/ethereum/EIPs/issues/114)</sup>的版本）。CREATE仅向子调用提供父级Gas的除一个64分之一外的所有。

即，替换：

```
        extra_gas = (not ext.account_exists(to)) * opcodes.GCALLNEWACCOUNT + \
            (value > 0) * opcodes.GCALLVALUETRANSFER
        if compustate.gas < gas + extra_gas:
            return vm_exception('OUT OF GAS', needed=gas+extra_gas)
        submsg_gas = gas + opcodes.GSTIPEND * (value > 0)
```

为：

```
        def max_call_gas(gas):
          return gas - (gas // 64)

        extra_gas = (not ext.account_exists(to)) * opcodes.GCALLNEWACCOUNT + \
            (value > 0) * opcodes.GCALLVALUETRANSFER
        if compustate.gas < extra_gas:
            return vm_exception('OUT OF GAS', needed=extra_gas)
        if compustate.gas < gas + extra_gas:
            gas = min(gas, max_call_gas(compustate.gas - extra_gas))
        submsg_gas = gas + opcodes.GSTIPEND * (value > 0)
```

### 基本原理

最近的拒绝服务攻击表明，相对于其他操作码，读取状态树的操作码定价过低。已经进行了一些软件更改，正在进行的更改以及可以进行的更改以缓解这种情况；然而，事实仍然是，这些操作码将是最容易通过交易垃圾邮件来降低网络性能的已知机制。这种担忧源于从磁盘读取数据需要很长时间，并且对未来的分片提案也构成风险，因为迄今为止最成功地降低网络性能的“攻击交易”还需要数十兆字节的数据来提供Merkle证明。此EIP增加了存储读取操作码的成本，以解决这一问题。成本是根据用于生成1.0 Gas成本的计算表的更新版本得出的：https://docs.google.com/spreadsheets/d/15wghZr-Z6sRSMdmRmhls9dVXTOpxKy8Y64oy9MvDZEQ/edit#gid=0；这些规则旨在将需要读取的数据量限制在8 MB以内，并包括对SLOAD的Merkle证明的500字节和账户的1000字节的估计。

此EIP旨在简单，并在根据此表计算的成本之上增加300 Gas的固定惩罚，以考虑加载代码的成本（在最坏情况下约为17-21 kb）。

引入EIP 90 Gas机制是因为如果没有它，所有当前进行调用的合约将停止工作，因为它们使用类似`msg.gas - 40`的表达式来确定进行调用时使用的Gas量，依赖于调用的Gas成本为40。此外，引入EIP 114是因为，鉴于我们正在提高调用的成本并使其更不可预测，我们有机会在不增加当前可用保证的成本的情况下实现这一点，因此我们还实现了用基于Gas的“更软”限制替换调用栈深度限制的好处，从而消除了调用栈深度攻击作为合约开发者必须担心的攻击类别，从而提高了合约编程的安全性。请注意，在给定的参数下，事实上的最大调用栈深度限制为~340（从~1024下降），从而减轻了任何进一步潜在的依赖于调用的二次复杂度DoS攻击的危害。

建议增加Gas上限，以保持系统对平均合约的每秒交易处理能力。

## 参考文献

1. EIP-90, https://github.com/ethereum/EIPs/issues/90
2. EIP-114, https://github.com/ethereum/EIPs/issues/114
```
