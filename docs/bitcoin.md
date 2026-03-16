## Bitcoin: A Peer-to-Peer Electronic Cash System

// tx (Transaction): UTXO spend/create, input sig your priv, output pubhash.
// 原子：in > out + fee。

// merkle_root: Binary tree of tx hashes, root in header。
// Proof: SPV light client O(log n) verify tx without full block。

// timestamp: ~10min, honest node reject >2h future/<local。

// nonce: 4byte counter, miner ++ grind hash(header) < target。

// difficulty: Target bits (e.g. 18 zero prefix), retarget 2016 blocks ~2w。
// Low difficulty = easy mine, high = hard。


Tx签名流程澄清（用Alice/Bob/Charlie，S5+ S8 reclaim）：

No上个私钥：每个input自己私钥签，验证自己pub（链式ownership）。
例子：

Genesis：Coinbase tx，miner create new coins to self pub。
Tx1 Alice→Bob：
Input: Alice's UTXO (e.g. coinbase)。
Alice hash(tx1), priv1 sign → sigScript。
Output: value to Bob pubhash。
Verify: Bob node check Alice pub verify sig。
Tx2 Bob→Charlie：
Input: Tx1 output (Bob's UTXO)。
Bob hash(tx2), priv2 sign → sigScript。
Output: value to Charlie pubhash。
Verify: Charlie node check Bob pub (from Tx1 output script) verify sig。

卧槽，我看懂了，就跟etcd之间peer用tls那个CA一模一样，只不过那个是环状的感觉，这个是链式的，每次输入都有节点自己的公钥，然后用自己的私钥签名发布给下一个接受者，之后下一个接受者根据上一个次的输入就知道上一个人的公钥，然后就可以验证签名这样子吗？

时间戳做hash块的时候，不仅要带着自己的时间戳，还要带着上一次交易的时间戳。

说到了一个节点之后，后面所有的区块的工作，都需要一个新的渴望带头的节点的CPU全部重新做一遍，再做自己的？

If the majority were based on one-IP-address-one-vote, it could be subverted by anyone
able to allocate many IPs.
S6 P2P反Sybil：IP一票=云厂商买IP淹投票（one-ID-one-vote fail）。

PoW解：经济一票 (CPU/电=stake)，买IP无work=无效。

etcd类比：

Leader election: Raft fixed peers (config auth)，no Sybil。
Vs Bitcoin: Open P2P，PoW防spam join。

我才看了3页，我都快被感动哭了。我觉得我就是非常honest的人，但是还是有自己的一些小揪揪，但无伤大雅，我就是那个miner想要疯狂追赶前人的新人啊，这简直从我内心到外在表现都非常符合链式逻辑，甚至我认为未来人类发展也必然会如此发展。这就是传承，这就是延续，这就是理性的巅峰，这就是人类文明疯狂想在这个世界上留下足够长久的记忆。

换手率越高代表这个币对应的honest够靠谱！

A block header with no transactions would be about 80 bytes. If we suppose blocks are
generated every 10 minutes, 80 bytes * 6 * 24 * 365 = 4.2MB per year. With computer systems
typically selling with 2GB of RAM as of 2008, and Moore's Law predicting current growth of
1.2GB per year, storage should not be a problem even if the block headers must be kept in
memory
也就是保守说，现实生活中每隔10分钟，就能有一个新人与我共振！

- PoW chain timestamp strengthening.
- Tx sig flow: priv sign txhash, pub verify.
- Velocity = tx turnover, utility proxy.
- UTXO pruned storage ~10GB global.
- Fresh key anti pre-mine DS.

```zsh
brew install go@1.22 bitcoind geth jq
export PATH="/opt/homebrew/opt/go@1.22/bin:$PATH"
go get github.com/btcsuite/btcd/rpcclient@latest github.com/ethereum/go-ethereum/ethclient@latest
```

关于中本聪，先说已知的事实，再说有意思的部分。

**时代背景：** 2008年全球金融危机。雷曼兄弟破产，美国政府掏了7000亿美元救助大银行。普通人的钱、信任、储蓄，全部被一个他们根本无法控制的系统吞噬了。就在金融系统濒临崩溃的那一刻，2008年10月31日，一个署名中本聪的人，把论文发到了一个密码学邮件列表上。

**动机：** 他挖出的第一个区块里，嵌入了一行字：「The Times 03/Jan/2009 Chancellor on brink of second bailout for banks」——这不只是时间戳，是宣言。Bitcoin从一开始就不是为了赚钱，而是对一个失败系统的回答。

**之后：** 2010年，中本聪悄悄把网络控制权和代码库交给了Gavin Andresen，然后消失了。他挖了大约一百万个比特币，从来没动过。

**年龄和身份：** 完全不知道。「Satoshi」在日语里意思是「清醒的思考」，「Nakamoto」意思是「中央起源」——有人觉得这是一种刻意的讽刺：去中心化货币的创造者，给自己取了个「中央起源」的名字。没人知道他是一个人还是一群人，是否还活着。

---

关于你说「日本作家更容易写出延续、觉醒、文明这类作品」——我觉得你的直觉是对的，但原因可能比「岛国生存危机」更深一层。

日本是唯一被核弹炸过的国家，是战后从废墟里重建起来的，是一个极度压缩了「文明可以在一瞬间消失」这件事的民族记忆。所以他们对「遗留」、「传承」、「人类在时间长河里留下什么」这类命题，有一种别的文化没有的切肤之感。

中本聪这个名字是不是真的日本人我们不知道，但他选择消失这件事本身——把作品留下，把自己抹去——倒真的很符合某种东方美学。

bitcoin-cli -regtest getdescriptorinfo "addr(bcrt1q你的真实地址)"
```

它会返回一个带checksum的`descriptor`字段，然后用那个去importdescriptors。

---

但我说真的，这条路越走越麻烦。

你现在卡在checksum，搞定checksum之后还有Go代码里怎么自动计算checksum的问题——btcd没有封装这个，你得自己实现descriptor checksum算法，不复杂但很烦。

**我建议现在就切到扫块方案。**

你不需要importdescriptors，不需要checksum，不需要依赖Bitcoin Core的钱包功能。

思路很简单：
```
生成地址 → 存到你自己的数据库(address -> userID)
Deposit Watcher → 订阅新块 → 遍历交易outputs → 匹配数据库里的地址 → 入账