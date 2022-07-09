# 一致性hash简单理解

首先介绍一下，为啥会用到这个一致性hash，在`MapReduce`框架下，有个重要的环节   --`shuffle`过程，如下图所示：

![shuffle过程](D:\go学习笔记\笔记图片\shuffle过程.jpg)

现阶段在单机版本中已经介绍到map过程结束，现在往下一步介绍即`shuffle`过程，

## shuffle过程

Map-shuffle过程如下图所示：

![mapshuff](D:\go学习笔记\笔记图片\mapshuff.jpg)

用到一致性hash过程就是上图中partition步骤--对于map输出的每一个键值对，系统都会给定一个partition，partition值默认是通过计算key的hash值后对Reduce task的数量取模获得。如果一个键值对的partition值为1，意味着这个键值对会交给第一个Reducer处理。因此分配的是键值对--这是我一直不太理解的地方

其余的shuffle过程等在单机版本再一一介绍

## 一致性hash概述

这里存在一种场景, 当一个服务由多个服务器组共同提供时, key应该路由到哪一个服务.这里假如采用最通用的方式key%N(N为服务器数目), 这里乍一看没什么问题, 但是当服务器数目发送增加或减少时, 分配方式则变为key%(N+1)或key%(N-1).这里将会有大量的key失效迁移,如果后端key对应的是有状态的存储数据,那么毫无疑问,这种做法将导致服务器间大量的数据迁移,从而照成服务的不稳定. 为了解决类问题,一致性hash算法应运而生，详细原理见：[Consistent hashing - Wikipedia](https://en.wikipedia.org/wiki/Consistent_hashing)

### 一致性hash的特点

- 均衡性(Balance)

均衡性主要指,通过算法分配, 集群中各节点应该要尽可能均衡.

- 单调性(Monotonicity)

单调性主要指当集群发生变化时, 已经分配到老节点的key, 尽可能的任然分配到之前节点,以防止大量数据迁移, 这里一般的hash取模就很难满足这点,而一致性hash算法能够将发生迁移的key数量控制在较低的水平.

- 分散性(Spread)

分散性主要针对同一个key, 当在不同客户端操作时,可能存在客户端获取到的缓存集群的数量不一致,从而导致将key映射到不同节点的问题,这会引发数据的不一致性.好的hash算法应该要尽可能避免分散性.

- 负载(Load)

负载主要是针对一个缓存而言, 同一缓存有可能会被用户映射到不同的key上,从而导致该缓存的状态不一致.

从原理来看,一致性hash算法针对以上问题均有一个合理的解决

### 一致性hash和普通hash比较

1、普通的hash算法：

采用：`key % N` ，这种方法能保证相同的key值得到的hash值是一致的，下面给出在`MapReduce`框架下使用普通hash算法计算的代码：

```go
func ihash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32() & 0x7fffffff)
}
```

 但是没有考虑节点数量变化的场景。假设，移除了其中一台节点，只剩下 9 个，那么之前 `hash(key) % 10` 变成了 `hash(key) % 9`，也就意味着几乎缓存值对应的节点都发生了改变。即几乎所有的缓存值都失效了。节点在接收到对应的请求时，均需要重新去数据源获取数据，容易引起 `缓存雪崩`

**缓存雪崩**：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。常因为缓存服务器宕机，或缓存设置了相同的过期时间引起

2、一致性hash算法 

**2.1原理**：

一致性哈希算法将 key 映射到 2^32 的空间中，将这个数字首尾相连，形成一个环

- 计算节点/机器(通常使用节点的名称、编号、`IP `地址)的哈希值，放置在环上
- 计算 key 的哈希值，放置在环上，顺时针寻找到的第一个节点，就是应选取的节点/机器

![consiste hash](D:\go学习笔记\笔记图片\consiste hash.jpg)

如上左图所示：按照原理，现在有三个节点， 按照顺时针方向看，`key27`，`key11`，`key2`映射到peer2，`key23` 映射到 peer4，这是新增节点在`key27`，`key11`之间，那么将会影响`key27`的映射关系，即`key27`映射到peer8上，其余的映射关系均不发生变化。也就是说，一致性哈希算法，在新增/删除节点时，只需要重新定位该节点附近的一小部分数据，而不需要重新定位所有的节点，这就解决了普通hash算法的问题。

**2.2 数据倾斜问题**

什么叫数据倾斜？-- 即上图所示，左边`key27`，`key11`，`key2`映射到peer2，`key23` 映射到 peer4，而peer6没有数据映射，这是因为节点过少，导致环的下半部分key全部映射到到peer2，key过度倾向peer2，使得节点之间负载不均衡

为解决这一问题，引入虚拟节点概念，一个真实节点对应多个虚拟节点

假设 1 个真实节点对应 3 个虚拟节点，那么 peer1 对应的虚拟节点是 peer1-1、 peer1-2、 peer1-3（通常以添加编号的方式实现），其余节点也以相同的方式操作。

- 第一步，计算虚拟节点的 Hash 值，放置在环上。
- 第二步，计算 key 的 Hash 值，在环上顺时针寻找到应选取的虚拟节点，例如是 peer2-1，那么就对应真实节点 peer2。

虚拟节点扩充了节点的数量，解决了节点较少的情况下数据容易倾斜的问题。而且代价非常小，只需要增加一个字典(map)维护真实节点与虚拟节点的映射关系即可

下面给出实现一致性hash算法的代码，最后分析每一块对应的原理

```go
package main

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	hash     Hash
	replicas int
	keys     []int // Sorted
	hashMap  map[int]string
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}

func ihash(key []byte) uint32 {
	i, _ := strconv.Atoi(string(key))
	return uint32(i)
}

func main() {
	hash := New(3, ihash)
	hash.Add("6", "4", "2")
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		if hash.Get(k) != v {
			fmt.Printf("Asking for %s, should have yielded %s", k, v)
		}
	}
	fmt.Println(hash.hashMap)
	hash.Add("8")

	testCases["27"] = "8"

	for k, v := range testCases {
		if hash.Get(k) != v {
			fmt.Printf("Asking for %s, should have yielded %s", k, v)
		}
	}
	fmt.Println(hash.hashMap)
}

```

输出结果：

```go
map[2:2 4:4 6:6 12:2 14:4 16:6 22:2 24:4 26:6]
map[2:2 4:4 6:6 8:8 12:2 14:4 16:6 18:8 22:2 24:4 26:6 28:8]
```

代码解释：

1. 首先定义了一个函数类型`Hash`， 这个有点类似于资源池中定义的工厂函数，采取依赖注入的方式，允许用于替换成自定义的 Hash 函数，
2. `Map` 是一致性哈希算法的主数据结构，包含 4 个成员变量：Hash 函数 `hash`；虚拟节点倍数 `replicas`；哈希环 `keys`；虚拟节点与真实节点的映射表 `hashMap`，键是虚拟节点的哈希值，值是真实节点的名称。
3. 构造函数 `New()` 允许自定义虚拟节点倍数和 Hash 函数。
4. `Add` 函数允许传入 0 或 多个真实节点的名称。
5. 对每一个真实节点 `key`，对应创建 `m.replicas` 个虚拟节点，虚拟节点的名称是：`strconv.Itoa(i) + key`，即通过添加编号的方式区分不同虚拟节点。
6. 使用 `m.hash()` 计算虚拟节点的哈希值，使用 `append(m.keys, hash)` 添加到环上。
7. 在 `hashMap` 中增加虚拟节点和真实节点的映射关系。
8. 最后一步，环上的哈希值排序。
9. 选择节点就非常简单了，第一步，计算 key 的哈希值。
10. 第二步，顺时针找到第一个匹配的虚拟节点的下标 `idx`，从 m.keys 中获取到对应的哈希值。如果 `idx == len(m.keys)`，说明应选择 `m.keys[0]`，因为 `m.keys` 是一个环状结构，所以用取余数的方式来处理这种情况。
11. 第三步，通过 `hashMap` 映射得到真实的节点。

至此，整个一致性哈希算法就实现完成了。

上述即为简单的一致性hash的实现

## 下一步计划

在单机MapReduce框架下，在shuffle过程将传统的hash算法替换成一致性hash算法
