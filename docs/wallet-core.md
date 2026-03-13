## Wallet Core

| 专有名词 | 英文原词 | 简单解释（业务含义）|
| --- | --- | --- |
Wallet Core|Wallet Core|钱包核心服务。这是整个交易所系统的“心脏”，负责地址生成、余额查询、HD 钱包管理等核心逻辑。
unified WalletService interface|unified WalletService interface|统一钱包服务接口。一个抽象层，以后加 TRON、SOL、USDT 等链时，只需要实现这个接口就行，不用改其他代码。
HD wallet support|HD wallet support|分层确定性钱包（Hierarchical Deterministic Wallet）。用一个种子（seed）就能生成无数地址，支持 BIP44 路径管理（比如 m/44'/0'/0'/0/0）。
BTC and ETH wallet modules|BTC and ETH wallet modules|BTC 和 ETH 具体实现模块。把 BTC 和 ETH 的地址生成、余额查询等逻辑分开写，便于维护。
address generation|address generation|地址生成。根据用户 ID 自动生成充值地址（BTC 是 bc1q...，ETH 是 0x...）。
cmd/wallet-service entrypoint|cmd/wallet-service entrypoint|钱包服务启动入口。这是整个服务的“主程序”，负责启动 HTTP 接口、Prometheus 监控、优雅退出等。
HTTP API|HTTP API|HTTP 接口。对外提供 RESTful 接口，比如 /api/v1/address 可以让前端/后端调用生成地址。
Prometheus metrics|Prometheus metrics|Prometheus 监控指标。用来实时监控地址生成次数、余额变化等，配合 Grafana 画图表。
multi-chain foundation|multi-chain foundation|多链基础架构。现在支持 BTC + ETH，以后可以轻松扩展成支持 10+ 条链。
exchange deposit/withdrawal system|exchange deposit/withdrawal system|交易所充提币系统。这就是我们最终的目标：用户充值（deposit）和提现（withdrawal）的完整流程。
initial Wallet Core framework|initial Wallet Core framework|钱包核心框架初始版。表示我们现在搭建的是地基，后续会继续往上盖楼（加真实 HD 派生、余额查询、风控等）。


**我们新增了一个钱包核心服务（Wallet Core），它通过统一的接口同时支持 BTC 和 ETH 的地址生成，并做好了监控和多链扩展的准备，为后面的交易所充提币系统打下了坚实基础。**

**详解HD（Hierarchical Deterministic Wallet）：**

想象你有一把魔法主钥匙（Master Key / Seed）：
- 这把主钥匙可以无限衍生出无数把子钥匙（子地址）
- 每把子钥匙都能开不同的锁（不同用户的充值地址）
- 但只要你拿着主钥匙，就能找回所有子钥匙
- 而且衍生过程是完全确定性的（每次用同一个主钥匙 + 同一个路径，都会生成完全一样的地址）
这就是 HD 钱包的核心思想。

**为啥叫「分层」**
它像一棵树一样分层：
```text
主种子 (Seed)
   ↓
m/44'/0'/0'          ← 第一层（目的）
   ↓
m/44'/0'/0'/0        ← 第二层（账户）
   ↓
m/44'/0'/0'/0/0      ← 第三层（具体地址）
   ↓
m/44'/0'/0'/0/1      ← 给用户2的地址
   ↓
m/44'/0'/0'/0/999    ← 给用户999的地址
```

**为啥叫「确定性」**
因为数学上完全确定：
- 同一个种子 + 同一个路径（比如 m/44'/60'/0'/0/5）
- 永远生成同一个地址
- 全世界任何人在任何时间用这个种子 + 这个路径，都会得到一模一样的地址
这就是为什么只用备份12个助记词（种子），就能找回所有地址。

**BIP44标准地址派生路径**

符号|含义|具体解释
-- | -- | -- 
m|Master Key（主密钥）|起点，就是你备份的那个种子（Seed / 助记词）
/|分层分隔符|就像文件夹的 /，表示进入下一层
44'|Purpose（目的）|固定写 44'，表示我们使用 BIP44 标准（几乎所有现代钱包都用这个）
0'|Coin Type（币种类型）|这里是 0' = Bitcoin 如果是 ETH 就是 60' TRON 是 195'，等等
0'|Account（账户索引）| 账户层。通常从 0 开始（第一个账户）交易所一般用 0' 就够了
'|Hardened（强化派生）|带 ' 的数字表示强化派生（更安全，防止子密钥泄露后反推主密钥）

**第四级**
含义：收款地址 vs 找零地址
- 0 = External（外部地址 / 收款地址）
   - → 这就是给用户充值的地址
   - → 交易所最常用的一级（用户看到的就是这一层）
- 1 = Internal（内部地址 / 找零地址）
   - → 钱包自己在转账时产生的找零地址（用户看不见）
   - → 交易所一般不用，或者只在冷钱包内部用
交易所实际用法：
- 几乎所有用户充值地址都走 /0（第四级永远是 0）
- 所以你看到的路径通常是：m/44'/0'/0'/0/xxx

**第五级**
含义：具体是第几个地址
- /0 = 第一个地址
- /1 = 第二个地址
- /5 = 第六个地址
- /999 = 第 1000 个地址
交易所最常见的用法：
- 用 userID 作为 index（比如用户 ID 是 12345，就生成 /12345）
- 或者按顺序递增（第 1 个用户 /0，第 2 个用户 /1）
举例（最直观）：
用户ID|完整路径|生成的 BTC 地址类型
-- | -- | -- 
user1|m/44'/0'/0'/0/0|第1个充值地址
user2|m/44'/0'/0'/0/1|第2个充值地址
user999|m/44'/0'/0'/0/999|第1000个充值地址

我们彻底甩掉了那个「老古董依赖」 —— `github.com/btcsuite/btcutil/hdkeychain`
这个老包内部死扣了一个**超级老的 btcec 路径**（`github.com/btcsuite/btcd/btcec`），无论我们怎么 replace、怎么换版本，它都会偷偷拉旧版，导致 `unknown revision` 报错。

我们最后换成了 `github.com/tyler-smith/go-bip32` 这个现代干净库，它不带那些历史包袱，直接用新版 `btcec/v2`，所以整个模块图瞬间干净了。