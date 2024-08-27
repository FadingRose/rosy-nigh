# EIP-4844: 分片Blob交易

## 摘要

引入一种新的“携带Blob的交易”格式，其中包含大量数据，这些数据无法被EVM执行访问，但其承诺可以被访问。该格式旨在与完整分片中将使用的格式完全兼容。

## 动机

在短期内，Rollups是Ethereum的唯一可信扩展解决方案，并且可能在长期内也是如此。L1上的交易费用已经非常高昂数月，因此迫切需要采取任何必要措施来帮助促进整个生态系统向Rollups的迁移。Rollups显著降低了Ethereum用户的费用：Optimism和Arbitrum经常提供比Ethereum基础层本身低3-8倍的费用，而ZK Rollups由于更好的数据压缩并可以避免包含签名，其费用比基础层低40-100倍。

然而，即使这些费用对于许多用户来说仍然过高。Rollups本身长期不足的长期解决方案一直是数据分片，这将使Rollups可以使用的链上专用数据空间增加约每区块16 MB。然而，数据分片仍然需要相当长的时间来完成实施和部署。

本EIP通过实现分片中将使用的交易格式，但在不实际分片这些交易的情况下，提供了一个过渡解决方案。相反，这种交易格式的数据仅作为信标链的一部分，并由所有共识节点完全下载（但可以在相对较短的延迟后删除）。与完整数据分片相比，本EIP对可以包含的这些交易的数量设定了降低的限制，对应于每区块目标约0.375 MB和限制约0.75 MB。

## 规范

### 参数

| 常量 | 值 |
| - | - |
| `BLOB_TX_TYPE` | `Bytes1(0x03)` |
| `BYTES_PER_FIELD_ELEMENT` | `32` |
| `FIELD_ELEMENTS_PER_BLOB` | `4096` |
| `BLS_MODULUS` | `52435875175126190479447740508185965837690552500527637822603658699938581184513` |
| `VERSIONED_HASH_VERSION_KZG` | `Bytes1(0x01)` |
| `POINT_EVALUATION_PRECOMPILE_ADDRESS` | `Bytes20(0x0A)` |
| `POINT_EVALUATION_PRECOMPILE_GAS` | `50000` |
| `MAX_BLOB_GAS_PER_BLOCK` | `786432` |
| `TARGET_BLOB_GAS_PER_BLOCK` | `393216` |
| `MIN_BASE_FEE_PER_BLOB_GAS` | `1` |
| `BLOB_BASE_FEE_UPDATE_FRACTION` | `3338477` |
| `GAS_PER_BLOB` | `2**17` |
| `HASH_OPCODE_BYTE` | `Bytes1(0x49)` |
| `HASH_OPCODE_GAS` | `3` |
| [`MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS`](https://github.com/ethereum/consensus-specs/blob/4de1d156c78b555421b72d6067c73b614ab55584/configs/mainnet.yaml#L148) | `4096` |

### 类型别名

| 类型 | 基础类型 | 额外检查 |
| - | - | - |
| `Blob` | `ByteVector[BYTES_PER_FIELD_ELEMENT * FIELD_ELEMENTS_PER_BLOB]` | |
| `VersionedHash` | `Bytes32` | |
| `KZGCommitment` | `Bytes48` | 执行IETF BLS签名“KeyValidate”检查，但不允许身份点 |
| `KZGProof` | `Bytes48` | 与`KZGCommitment`相同 |

### 加密助手

在本提案中，我们使用了[共识4844规范](https://github.com/ethereum/consensus-specs/blob/86fb82b221474cc89387fa6436806507b3849d88/specs/deneb)中定义的加密方法和类。

具体来说，我们使用了[`polynomial-commitments.md`](https://github.com/ethereum/consensus-specs/blob/86fb82b221474cc89387fa6436806507b3849d88/specs/deneb/polynomial-commitments.md)中的以下方法：

- [`verify_kzg_proof()`](https://github.com/ethereum/consensus-specs/blob/86fb82b221474cc89387fa6436806507b3849d88/specs/deneb/polynomial-commitments.md#verify_kzg_proof)
- [`verify_blob_kzg_proof_batch()`](https://github.com/ethereum/consensus-specs/blob/86fb82b221474cc89387fa6436806507b3849d88/specs/deneb/polynomial-commitments.md#verify_blob_kzg_proof_batch)

### 助手

```python
def kzg_to_versioned_hash(commitment: KZGCommitment) -> VersionedHash:
    return VERSIONED_HASH_VERSION_KZG + sha256(commitment)[1:]
```

使用泰勒展开近似`factor * e ** (numerator / denominator)`：

```python
def fake_exponential(factor: int, numerator: int, denominator: int) -> int:
    i = 1
    output = 0
    numerator_accum = factor * denominator
    while numerator_accum > 0:
        output += numerator_accum
        numerator_accum = (numerator_accum * numerator) // (denominator * i)
        i += 1
    return output // denominator
```

### Blob交易

我们引入了一种新的[EIP-2718](./eip-2718.md)交易类型，“blob交易”，其中`TransactionType`为`BLOB_TX_TYPE`，`TransactionPayload`为以下`TransactionPayloadBody`的RLP序列化：

```
[chain_id, nonce, max_priority_fee_per_gas, max_fee_per_gas, gas_limit, to, value, data, access_list, max_fee_per_blob_gas, blob_versioned_hashes, y_parity, r, s]
```

字段`chain_id`、`nonce`、`max_priority_fee_per_gas`、`max_fee_per_gas`、`gas_limit`、`value`、`data`和`access_list`遵循与[EIP-1559](./eip-1559.md)相同的语义。

字段`to`略有不同，除了它必须始终表示一个20字节的地址，不能为`nil`。这意味着blob交易不能具有创建交易的格式。

字段`max_fee_per_blob_gas`是一个`uint256`，字段`blob_versioned_hashes`表示从`kzg_to_versioned_hash`输出的哈希列表。

此交易的[EIP-2718](./eip-2718.md) `ReceiptPayload`为`rlp([status, cumulative_transaction_gas_used, logs_bloom, logs])`。

#### 签名

签名值`y_parity`、`r`和`s`通过在以下摘要上构造一个secp256k1签名来计算：

`keccak256(BLOB_TX_TYPE || rlp([chain_id, nonce, max_priority_fee_per_gas, max_fee_per_gas, gas_limit, to, value, data, access_list, max_fee_per_blob_gas, blob_versioned_hashes]))`。

### 头部扩展

当前头部编码扩展了两个新的64位无符号整数字段：

- `blob_gas_used`是区块内交易消耗的总blob gas量。
- `excess_blob_gas`是超出目标的blob gas消耗的累计总量，前一个区块。高于目标的区块增加此值，低于目标的区块减少此值（以0为界）。

因此，头部的RLP编码结果为：

```
rlp([
    parent_hash,
    0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347, # ommers hash
    coinbase,
    state_root,
    txs_root,
    receipts_root,
    logs_bloom,
    0, # difficulty
    number,
    gas_limit,
    gas_used,
    timestamp,
    extradata,
    prev_randao,
    0x0000000000000000, # nonce
    base_fee_per_gas,
    withdrawals_root,
    blob_gas_used,
    excess_blob_gas,
])
```

`excess_blob_gas`的值可以使用父头部计算。

```python
def calc_excess_blob_gas(parent: Header) -> int:
    if parent.excess_blob_gas + parent.blob_gas_used < TARGET_BLOB_GAS_PER_BLOCK:
        return 0
    else:
        return parent.excess_blob_gas + parent.blob_gas_used - TARGET_BLOB_GAS_PER_BLOCK
```

对于分叉后的第一个区块，`parent.blob_gas_used`和`parent.excess_blob_gas`都被视为`0`。

### 气体会计

我们引入了一种新的气体类型——blob gas。它独立于普通气体，并遵循自己的目标规则，类似于EIP-1559。我们使用`excess_blob_gas`头部字段来存储计算blob gas基础费用所需的持久数据。目前，只有blobs以blob gas计价。

```python
def calc_blob_fee(header: Header, tx: Transaction) -> int:
    return get_total_blob_gas(tx) * get_base_fee_per_blob_gas(header)

def get_total_blob_gas(tx: Transaction) -> int:
    return GAS_PER_BLOB * len(tx.blob_versioned_hashes)

def get_base_fee_per_blob_gas(header: Header) -> int:
    return fake_exponential(
        MIN_BASE_FEE_PER_BLOB_GAS,
        header.excess_blob_gas,
        BLOB_BASE_FEE_UPDATE_FRACTION
    )
```

区块有效性条件被修改以包括blob gas检查（参见下面的[执行层验证](#execution-layer-validation)部分）。

通过`calc_blob_fee`计算的实际`blob_fee`在交易执行前从发送者余额中扣除并销毁，并且在交易失败时不退还。

### 获取版本化哈希的Opcode

我们添加了一个指令`BLOBHASH`（带有操作码`HASH_OPCODE_BYTE`），该指令从堆栈顶部读取`index`作为大端`uint256`，并用`tx.blob_versioned_hashes[index]`替换它，如果`index < len(tx.blob_versioned_hashes)`，否则用零值`bytes32`替换。该操作码的气体成本为`HASH_OPCODE_GAS`。

### 点评估预编译

在`POINT_EVALUATION_PRECOMPILE_ADDRESS`处添加一个预编译，用于验证声称blob（由承诺表示）在给定点评估为给定值的KZG证明。

预编译成本为`POINT_EVALUATION_PRECOMPILE_GAS`，执行以下逻辑：

```python
def point_evaluation_precompile(input: Bytes) -> Bytes:
    """
    验证p(z) = y给定承诺对应于多项式p(x)和一个KZG证明。
    还要验证提供的承诺与提供的版本化哈希匹配。
    """
    # 数据编码如下：versioned_hash | z | y | commitment | proof | 其中z和y是填充的32字节大端值
    assert len(input) == 192
    versioned_hash = input[:32]
    z = input[32:64]
    y = input[64:96]
    commitment = input[96:144]
    proof = input[144:192]

    # 验证承诺与版本化哈希匹配
    assert kzg_to_versioned_hash(commitment) == versioned_hash

    # 验证KZG证明，其中z和y是大端格式
    assert verify_kzg_proof(commitment, z, y, proof)

    # 返回FIELD_ELEMENTS_PER_BLOB和BLS_MODULUS作为填充的32字节大端值
    return Bytes(U256(FIELD_ELEMENTS_PER_BLOB).to_be_bytes32() + U256(BLS_MODULUS).to_be_bytes32())
```

预编译必须拒绝非规范字段元素（即提供的字段元素必须严格小于`BLS_MODULUS`）。

### 共识层验证

在共识层，blobs在信标块体中被引用，但未完全编码。相反，blobs作为“侧载车”单独传播。

这种“侧载车”设计为数据增加提供了向前兼容性，通过黑盒化`is_data_available()`：在完整分片中，`is_data_available()`可以被数据可用性采样（DAS）替代，从而避免所有信标节点在网络上下载所有blobs。

请注意，共识层负责为数据可用性持久化blobs，执行层则不负责。

`ethereum/consensus-specs`仓库定义了本EIP涉及的以下共识层更改：

- 信标链：处理更新的信标块并确保blobs可用。
- P2P网络：传播和同步更新的信标块类型和新blob侧载车。
- 诚实验证者：生成带有blobs的信标块；签名并发布相关的blob侧载车。

### 执行层验证

在执行层，区块有效性条件被扩展如下：

```python
def validate_block(block: Block) -> None:
    ...

    # 检查excess_blob_gas是否正确更新
    assert block.header.excess_blob_gas == calc_excess_blob_gas(block.parent.header)

    blob_gas_used = 0

    for tx in block.transactions:
        ...

        # 修改足够的余额检查
        max_total_fee = tx.gas * tx.max_fee_per_gas
        if get_tx_type(tx) == BLOB_TX_TYPE:
            max_total_fee += get_total_blob_gas(tx) * tx.max_fee_per_blob_gas
        assert signer(tx).balance >= max_total_fee

        ...

        # 添加特定于blob tx的有效性逻辑
        if get_tx_type(tx) == BLOB_TX_TYPE:
            # 必须至少有一个blob
            assert len(tx.blob_versioned_hashes) > 0

            # 所有版本化blob哈希必须以VERSIONED_HASH_VERSION_KZG开头
            for h in tx.blob_versioned_hashes:
                assert h[0] == VERSIONED_HASH_VERSION_KZG

            # 确保用户愿意至少支付当前的blob基础费用
            assert tx.max_fee_per_blob_gas >= get_base_fee_per_blob_gas(block.header)

            # 跟踪区块中消耗的总blob gas
            blob_gas_used += get_total_blob_gas(tx)

    # 确保总blob gas消耗最多等于限制
    assert blob_gas_used <= MAX_BLOB_GAS_PER_BLOCK

    # 确保blob_gas_used与头部匹配
    assert block.header.blob_gas_used == blob_gas_used

```

### 网络

Blob交易有两种网络表示。在交易广播响应（`PooledTransactions`）中，blob交易的EIP-2718 `TransactionPayload`被包装为：

```
rlp([tx_payload_body, blobs, commitments, proofs])
```

这些元素的定义如下：

- `tx_payload_body` - 是标准EIP-2718 [blob交易](#blob-transaction)的`TransactionPayloadBody`
- `blobs` - `Blob`项的列表
- `commitments` - 对应`blobs`的`KZGCommitment`列表
- `proofs` - 对应`blobs`和`commitments`的`KZGProof`列表

节点必须验证`tx_payload_body`并根据其验证包装数据。为此，确保：

- `tx_payload_body.blob_versioned_hashes`、`blobs`、`commitments`和`proofs`的数量相等。
- KZG `commitments`哈希到版本化哈希，即`kzg_to_versioned_hash(commitments[i]) == tx_payload_body.blob_versioned_hashes[i]`
- KZG `commitments`与对应的`blobs`和`proofs`匹配。（注意：这可以使用`verify_blob_kzg_proof_batch`进行优化，每个blob提供一个在从承诺和blob数据导出的点上的随机评估的证明）

对于块体检索响应（`BlockBodies`），使用标准EIP-2718 blob交易的`TransactionPayload`。

节点不得自动向其对等方广播blob交易。相反，这些交易仅使用`NewPooledTransactionHashes`消息进行宣布，然后可以通过`GetPooledTransactions`手动请求。

## 基本原理

### 通往分片的道路

本EIP引入了blob交易，其格式与最终分片规范中预期的格式相同。这为Rollups提供了一个临时的但显著的扩展缓解，允许它们最初扩展到每槽0.375 MB，并有一个单独的费用市场，允许在系统使用有限时费用非常低。

Rollup扩展停顿的核心目标是提供临时扩展缓解，而不对Rollups施加额外的开发负担以利用这种缓解。今天，Rollups使用calldata。在未来，Rollups将别无选择，只能使用分片数据（也称为“blobs”），因为分片数据将便宜得多。因此，Rollups无法避免在途中至少进行一次大规模升级。但我们能做的是确保Rollups只需要升级一次。这立即意味着停顿有两种可能性：（i）降低现有calldata的gas成本，（ii）提前使用分片数据将使用的格式，但不实际分片它。以前的EIP都是类别（i）的解决方案；本EIP是类别（ii）的解决方案。

在设计本EIP时的主要权衡是现在实施更多与以后实施更多的权衡：我们是实施通往完整分片的25%的工作，还是50%，还是75%？

本EIP已经完成的工作包括：

- 一种新的交易类型，其格式与“完整分片”中将存在的格式完全相同
- 完整分片所需的所有执行层逻辑
- 完整分片所需的所有执行/共识交叉验证逻辑
- `BeaconBlock`验证和数据可用性采样blobs之间的层分离
- 完整分片所需的大部分`BeaconBlock`逻辑
- 一个自我调整的独立blob基础费用

通往完整分片所需的工作包括：

- 共识层中`commitments`的低次扩展，以允许2D采样
- 数据可用性采样的实际实现
- PBS（提议者/构建者分离），以避免要求单个验证者在单个槽中处理32 MB的数据
- 证明保管或类似的协议内要求，要求每个验证者在每个区块中验证特定部分的分片数据

本EIP还为长期协议清理奠定了基础。例如，其（更清洁的）gas基础费用更新规则可以应用于主要基础费用计算。

### Rollups如何运作

Rollups将期望Rollup块提交者将数据放入blobs中，而不是将Rollup块数据放入交易calldata中。这保证了可用性（这是Rollups需要的），但会比calldata便宜得多。Rollups需要数据可用一次，足够长的时间以确保诚实行为者可以构建Rollup状态，但不需要永远。

乐观Rollups只需要在提交欺诈证明时实际提供底层数据。欺诈证明可以分小步骤验证转换，每次最多加载几个blob值通过calldata。对于每个值，它将提供一个KZG证明并使用点评估预编译来验证该值与之前提交的版本化哈希一致，然后对数据执行与当前欺诈证明验证相同的操作。

ZK Rollups将提供两个承诺给它们的交易或状态增量数据：blob承诺（协议确保指向可用数据）和ZK Rollup使用其内部证明系统的承诺。它们将使用等价证明协议，使用点评估预编译，证明两个承诺指向相同的数据。

### 版本化哈希和预编译返回数据

我们在执行层使用版本化哈希（而不是承诺）作为blobs的引用，以确保向前兼容未来的更改。例如，如果我们需要切换到Merkle树+STARKs以实现量子安全，那么我们将添加一个新版本，允许点评估预编译与新格式一起工作。Rollups不需要在EVM级别上进行任何更改；序列器只需在适当的时候切换到使用新交易类型。

然而，点评估发生在有限域内，并且只有在域模数已知时才有明确定义。智能合约可以包含一个将承诺版本映射到模数的表，但这不允许智能合约考虑到未来未知的模数升级。通过允许在EVM内访问模数，智能合约可以构建为可以使用未来的承诺和证明，而无需任何升级。

为了不添加另一个预编译，我们直接从点评估预编译返回模数和多项式度数。然后调用者可以使用它。这也是“免费”的，因为调用者可以忽略返回值的这一部分而不产生额外成本——在可预见的未来保持可升级性的系统可能会选择这条路线。

### Blob gas基础费用更新规则

Blob gas基础费用更新规则旨在近似公式`base_fee_per_blob_gas = MIN_BASE_FEE_PER_BLOB_GAS * e**(excess_blob_gas / BLOB_BASE_FEE_UPDATE_FRACTION)`，其中`excess_blob_gas`是链相对于“目标”数量（每区块`TARGET_BLOB_GAS_PER_BLOCK`）消耗的“额外”blob gas总量。与EIP-1559一样，它是一个自我纠正的公式：随着超额增加，`base_fee_per_blob_gas`呈指数增长，减少使用并最终迫使超额下降。

逐块行为大致如下。如果块`N`消耗了`X` blob gas，那么在块`N+1`中`excess_blob_gas`增加了`X - TARGET_BLOB_GAS_PER_BLOCK`，因此块`N+1`的`base_fee_per_blob_gas`增加了`e**((X - TARGET_BLOB_GAS_PER_BLOCK) / BLOB_BASE_FEE_UPDATE_FRACTION)`的因子。因此，它与现有的EIP-1559有类似的效果，但在某种意义上更“稳定”，因为它对相同总使用的响应方式相同，无论其分布如何。

参数`BLOB_BASE_FEE_UPDATE_FRACTION`控制blob gas基础费用的最大变化率。它旨在实现每块最大变化率`e**(TARGET_BLOB_GAS_PER_BLOCK / BLOB_BASE_FEE_UPDATE_FRACTION) ≈ 1.125`。

### 吞吐量

`TARGET_BLOB_GAS_PER_BLOCK`和`MAX_BLOB_GAS_PER_BLOCK`的值被选择为对应于每区块目标3个blobs（0.375 MB）和最大6个blobs（0.75 MB）。这些小的初始限制旨在最大限度地减少本EIP对网络造成的压力，并预计在网络在更大区块下表现出可靠性后，在未来的升级中增加。

## 向后兼容性

### Blob不可访问性

本EIP引入了一种交易类型，具有独特的内存池版本和执行负载版本，只有两者之间的一向转换性。blobs在网络表示中，而不是在共识表示中；相反，它们与信标块耦合。这意味着现在有一部分交易无法从web3 API访问。

### 内存池问题

Blob交易在内存池层具有大量数据大小，这构成了内存池DoS风险，尽管这不是前所未有的，因为这也适用于具有大量calldata的交易。

通过仅广播blob交易的公告，接收节点将对哪些和多少交易进行控制，允许它们将吞吐量限制在可接受的水平。[EIP-5793](./eip-5793.md)将通过扩展`NewPooledTransactionHashes`公告消息以包括交易类型和大小，为节点提供更细粒度的控制。

此外，我们建议在内存池交易替换规则中包括1.1x blob gas基础费用增加要求。

## 测试案例

本EIP的执行层测试案例可以在`ethereum/execution-spec-tests`仓库的[`eip4844_blobs`](https://github.com/ethereum/execution-spec-tests/tree/1983444bbe1a471886ef7c0e82253ffe2a4053e1/tests/cancun/eip4844_blobs)中找到。共识层测试案例可以在[这里](https://github.com/ethereum/consensus-specs/tree/2297c09b7e457a13f7b2261a28cb45777be82f83/tests/core/pyspec/eth2spec/test/deneb)找到。

## 安全考虑

本EIP将每个信标块的最大带宽要求增加了~0.75 MB。这比今天理论上的最大块大小（30M gas / 16 gas per calldata byte = 1.875M bytes）大40%，因此不会大大增加最坏情况下的带宽。合并后，块时间静态而不是不可预测的泊松分布，为大块传播提供了保证的时间段。

本EIP的持续负载比减少calldata成本的替代方案要低得多，即使calldata有限，因为blobs不需要像执行负载那样长时间存储。这使得可以实施这些blobs必须至少保存一定时间的策略。所选值为`MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS` epochs，大约为18天，与执行负载历史记录提出的（但尚未实施）一年轮换时间相比，延迟要短得多。

## 版权

通过[CC0](../LICENSE.md)放弃版权及相关权利。
