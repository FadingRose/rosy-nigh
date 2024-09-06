# Argument Index

为了在多个组件中确定同一个参数位置, 引入一个元组

```go
type ArgIndex struct {
  contract string
  method string
  name string
  offset uint64
  size uint64
  val interface{}
}
```

理想执行过程如下:
1. 每个 FuzzLoop 中, 从 `GenerateArgs` 中得到 `params`, 从中构造 `ArgIndex`;
2. 展开每一个尚未完全覆盖的 `JUMPI`, 检测是否存在 `MLOAD` 绑定？
    2.a 如何确定 `MLOAD` 存在绑定?
        2.a.1 `Reg.Val == ArgIndex.Val`
        2.a.2 `Reg.L.Val == ArgIndex.offset`
        2.a.3 `Reg.op == MLOAD`
        3.a.4 (optional) `ArgIndex == 32`
    2.b 需要注意，可能存在其他读入完成参数绑定，即读入大小不为 32 (`MLOAD` 默认读入 32 字节)
3. 如果存在，SendToSMT
