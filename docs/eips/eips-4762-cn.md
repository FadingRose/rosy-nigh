### 概要

本EIP引入了燃气费用的变更，以反映创建见证者的成本。它要求客户端更新其数据库布局以匹配这一变更，从而避免潜在的DoS攻击。

### 动机

将Verkle树引入以太坊需要根本性的变化，作为准备工作，本EIP针对在Verkle树分叉之前的分叉，以激励Dapp开发者采用新的存储模型，并给予充足的时间进行调整。同时，它也激励客户端开发者在其数据库格式迁移到Verkle分叉之前进行迁移。

### 规范

#### 辅助函数

```python
def get_storage_slot_tree_keys(storage_key: int) -> [int, int]:
    if storage_key < (CODE_OFFSET - HEADER_STORAGE_OFFSET):
        pos = HEADER_STORAGE_OFFSET + storage_key
    else:
        pos = MAIN_STORAGE_OFFSET + storage_key
    return (
        pos // 256,
        pos % 256
    )
```

#### 访问事件

每当读取状态时，会发生一个或多个形式的访问事件`(地址, 子键, 叶键)`，确定正在访问哪些数据。我们定义访问事件如下：

##### 账户头的访问事件

当：

1. 非预编译地址是`*CALL`、`SELFDESTRUCT`、`EXTCODESIZE`或`EXTCODECOPY`操作码的目标，
2. 非预编译地址是合约创建的目标地址，其初始代码开始执行，
3. 任何地址是`BALANCE`操作码的目标，
4. 已部署的合约调用`CODECOPY`

处理此访问事件：

```
(地址, 0, BASIC_DATA_LEAF_KEY)
```

注意：非价值转移的`SELFDESTRUCT`或`*CALL`，目标为预编译，不会导致`BASIC_DATA_LEAF_KEY`被添加到见证中。

如果`*CALL`或`SELFDESTRUCT`是价值转移的（即转移非零wei），无论`callee`是否为预编译，处理此附加访问事件：

```
(调用者, 0, BASIC_DATA_LEAF_KEY)
```

注意：当检查`callee`的存在时，通过验证相应stem处存在扩展和后缀树来进行存在性检查，而不依赖于`CODEHASH_LEAF_KEY`。

当对非预编译目标调用`EXTCODEHASH`时，处理访问事件：

```
(地址, 0, CODEHASH_LEAF_KEY)
```

注意预编译被排除在外，因为它们的哈希对客户端是已知的。

当创建合约时，处理这些访问事件：

```
(合约地址, 0, BASIC_DATA_LEAF_KEY)
(合约地址, 0, CODEHASH_LEAF_KEY)
```

##### 存储的访问事件

具有给定地址和键的`SLOAD`和`SSTORE`操作码处理形式的访问事件

```
(地址, 树键, 子键)
```

其中`树键`和`子键`计算为`树键, 子键 = get_storage_slot_tree_keys(地址, 键)`

##### 代码的访问事件

在以下条件下，“访问chunk_id块”理解为形式的访问事件

```
(地址, (chunk_id + 128) // 256, (chunk_id + 128) % 256)
```

* 在EVM执行的每一步，如果且仅当`PC < len(code)`，访问被调用者的chunk `PC // CHUNK_SIZE`（其中`PC`是当前程序计数器）。特别注意以下边界情况：
    * `JUMP`（或正向评估的`JUMPI`）的目标被认为是访问的，即使目标不是跳转目的地或在推送数据内
    * `JUMPI`的目标在跳转条件为`false`时不被认为是访问的。
    * 如果执行到达跳转操作码但没有足够的gas支付执行`JUMP`操作码的gas成本（包括如果`JUMP`是尚未访问的块中的第一个操作码的块访问成本），跳转的目标不被认为是访问的
    * 如果跳转的目标超出代码（`destination >= len(code)`），跳转的目标不被认为是访问的
    * 如果代码通过走过代码末尾停止执行，`PC = len(code)`不被认为是访问的
* 如果EVM执行的当前步骤是`PUSH{n}`，访问被调用者的所有块`(PC // CHUNK_SIZE) <= chunk_index <= ((PC + n) // CHUNK_SIZE)`。
* 如果非零读取大小的`CODECOPY`或`EXTCODECOPY`读取字节`x...y`（包括），访问的合约的所有块`(x // CHUNK_SIZE) <= chunk_index <= (min(y, code_size - 1) // CHUNK_SIZE)`被访问。
    * 示例1：对于起始位置100，读取大小50，`code_size = 200`的`CODECOPY`，`x = 100`和`y = 149`
    * 示例2：对于起始位置600，读取大小0的`CODECOPY`，没有块被访问
    * 示例3：对于起始位置1500，读取大小2000，`code_size = 3100`的`CODECOPY`，`x = 1500`和`y = 3099`
* `CODESIZE`、`EXTCODESIZE`和`EXTCODEHASH`不访问任何块。
   当创建合约时，访问块`0 ... (len(code)+30)//31`

### 写事件

我们定义**写事件**如下。注意，当发生写操作时，也会发生访问事件（因此下面的定义应该是访问事件定义的子集）。写事件的形式为`(地址, 子键, 叶键)`，确定正在写入哪些数据。

#### 账户头的写事件

当具有给定发送者和接收者的非零余额发送的`*CALL`或`SELFDESTRUCT`发生时，处理这些写事件：

```
(调用者, 0, BASIC_DATA_LEAF_KEY)
(被调用者, 0, BASIC_DATA_LEAF_KEY)
```

如果`callee_address`处没有账户存在，也处理：

```
(被调用者, 0, CODEHASH_LEAF_KEY)
```

当合约创建被初始化时，处理这些写事件：

```
(合约地址, 0, BASIC_DATA_LEAF_KEY)
```

当合约被创建时，处理这些写事件：

```
(合约地址, 0, CODEHASH_LEAF_KEY)
```

#### 存储的写事件

具有给定`地址`和`键`的`SSTORE`操作码处理形式的写事件

```
(地址, 树键, 子键)
```

其中`树键`和`子键`计算为`树键, 子键 = get_storage_slot_tree_keys(地址, 键)`

#### 代码的写事件

当创建合约时，处理写事件：

```python
(
    地址,
    (CODE_OFFSET + i) // VERKLE_NODE_WIDTH,
    (CODE_OFFSET + i) % VERKLE_NODE_WIDTH
)
```

对于`i`在`0 ... (len(code)+30)//31`。

注意：由于在此EIP之前代码不存在访问列表，因此不对代码访问收取暖费用。

### 交易

#### 访问事件

对于交易，进行这些访问事件：

```
(tx.origin, 0, BASIC_DATA_LEAF_KEY)
(tx.origin, 0, CODEHASH_LEAF_KEY)
(tx.target, 0, BASIC_DATA_LEAF_KEY)
(tx.target, 0, CODEHASH_LEAF_KEY)
```

#### 写事件

```
(tx.origin, 0, BASIC_DATA_LEAF_KEY)
```

如果`value`非零：

```
(tx.target, 0, BASIC_DATA_LEAF_KEY)
```

### 见证燃气成本

移除以下燃气成本：

* 如果`CALL`是非零价值发送，增加的燃气成本
* [EIP-2200](./eip-2200.md) `SSTORE`燃气成本，除了`SLOAD_GAS`
* 每字节合约代码成本200

减少燃气成本：

* `CREATE`/`CREATE2`到1000

|常量 |值|
|-|-|
|`WITNESS_BRANCH_COST`|1900|
|`WITNESS_CHUNK_COST`	|200|
|`SUBTREE_EDIT_COST`	|3000|
|`CHUNK_EDIT_COST`    |500|
|`CHUNK_FILL_COST`    |6200|

在执行交易时，维护四个集合：

* `accessed_subtrees: Set[Tuple[address, int]]`
* `accessed_leaves: Set[Tuple[address, int, int]]`
* `edited_subtrees: Set[Tuple[address, int]]`
* `edited_leaves: Set[Tuple[address, int, int]]`

当发生`(地址, 子键, 叶键)`形式的**访问**事件时，执行以下检查：

* 除非事件是_交易访问事件_，否则执行以下步骤；
* 如果`(地址, 子键)`不在`accessed_subtrees`中，收取`WITNESS_BRANCH_COST`燃气并添加该元组到`accessed_subtrees`。
* 如果`叶键`不是`None`且`(地址, 子键, 叶键)`不在`accessed_leaves`中，收取`WITNESS_CHUNK_COST`燃气并将其添加到`accessed_leaves`

当发生`(地址, 子键, 叶键)`形式的**写**事件时，执行以下检查：

* 如果事件是_交易写事件_，跳过以下步骤。
* 如果`(地址, 子键)`不在`edited_subtrees`中，收取`SUBTREE_EDIT_COST`燃气并添加该元组到`edited_subtrees`。
* 如果`叶键`不是`None`且`(地址, 子键, 叶键)`不在`edited_leaves`中，收取`CHUNK_EDIT_COST`燃气并添加到`edited_leaves`
    * 此外，如果`(地址, 子键, 叶键)`处没有存储值（即状态在该位置持有`None`），收取`CHUNK_FILL_COST`

注意，树键不能再被清空：只有值`0...2**256-1`可以写入树键，并且0与`None`不同。一旦树键从`None`变为非`None`，它就永远不能再回到`None`。

注意，只有在有足够的燃气覆盖其关联事件成本的情况下，才应将值添加到见证中。

`CREATE*`和`*CALL`在嵌套执行之前保留1/64的燃气。为了使这种收费行为与分叉前访问列表的行为相匹配：

* 在进行`CALL`、`CODECALL`、`DELEGATECALL`或`STATICCALL`时，在收取见证成本**之后**检查这个最低1/64的燃气预留
* 在进行`CREATE`或`CREATE2`时，在收取见证成本**之前**减去这个1/64的燃气

### 块级操作

以下各项在交易开始时都不是暖的：

* 在系统调用期间访问的预编译账户、系统合约账户和系统合约的槽，
* 币基账户
* 提款账户

注意：当（且仅当）通过系统调用调用系统合约时，代码块和账户不应出现在见证中。

### 账户抽象

TODO：仍在等待7702和3074之间的最终决定

## 基本原理

### 燃气改革

存储和代码读取的燃气成本进行了改革，以更紧密地反映在新Verkle树设计下的燃气成本。`WITNESS_CHUNK_COST`设置为每字节块收取6.25燃气，`WITNESS_BRANCH_COST`设置为平均每字节块收取约13.2燃气（假设144字节块长度），在最坏情况下，如果攻击者故意计算最大化证明长度的键来填充树，每字节块收取约2.5燃气。

与柏林燃气成本的主要区别是：

* 每31字节代码块收取200燃气。这已被估计会增加平均燃气使用量约6-12%，表明在每块350燃气水平下会增加10-20%的燃气使用量。
* 访问相邻存储槽（`key1 // 256 == key2 // 256`）的成本从2100降低到200，包括组中的第一个槽之后所有槽，
* 访问存储槽0…63的成本从2100降低到200，包括第一个存储槽。这可能会显著提高许多现有合约的性能，这些合约使用这些存储槽作为单个持久变量。

尚未分析从后两个属性获得的收益，但可能会显著抵消第一个属性的损失。一旦编译器适应这些规则，效率可能会进一步提高。

访问事件发生的精确规范，构成了燃气重新定价的大部分复杂性，是必要的，以清楚地指定何时需要将数据保存到周期1树。

## 向后兼容性

本EIP需要硬分叉，因为它修改了共识规则。

主要的向后兼容性破坏变化是代码块访问的燃气成本使某些应用在经济上不可行。可以通过在实施本EIP的同时增加燃气限制来缓解，减少应用由于交易燃气使用量上升超过块燃气限制而不再工作的风险。

## 安全考虑

本EIP将意味着某些操作，主要是读取和写入同一后缀树中的多个元素，变得更便宜。如果客户端保留与现在相同的数据库结构，这将导致DOS向量。

因此，需要对数据库进行一些适应以使其工作：

* 在所有可能的未来中，将承诺方案与数据存储逻辑分离是很重要的。特别是，找到任何给定状态元素不需要遍历承诺方案树
* 为了使对同一stem的访问便宜，如本EIP所要求的，最好的方法可能是在数据库中相同位置存储每个stem。基本上，256个32字节的叶子将存储在一个8kB的BLOB中。读取/写入此BLOB的开销很小，因为磁盘访问的大部分成本是寻道而不是传输的量。

## 版权

通过[CC0](../LICENSE.md)放弃版权及相关权利。
