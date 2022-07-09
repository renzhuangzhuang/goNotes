# go 实现资源池及个人理解

## go by example案例分析

```go
package main

import (
	"fmt"
	"time"
)

func worker(id int, jobs <-chan int, result chan<- int) {
	for j := range jobs {
		fmt.Println("worker", id, "started job", j)
		time.Sleep(time.Second)
		fmt.Println("worker", id, "finished job", j)
		result <- j * 2
	}
}

func main() {

	const numJobs = 5
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results)
	}

	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	for a := 1; a <= numJobs; a++ {
		<-results
	}
}
```

### 代码解读

go 官方提供了一个简单的工作池实现思路，利用goroutinue和channel来实现

在worker函数中，传递了两种channel类型，jobs 为只读，result 为只写，

如何理解这个只读只写-- 下面给出一个小案例

```go
package main

import (
	"fmt"
	"time"
)

func testRead(jobs <-chan int) {
	for j := range jobs {
		fmt.Println("接收任务号: ", j)
		time.Sleep(time.Second)
	}
}

func main() {

	channel_read := make(chan int, 5)

	go testRead(channel_read)

	for j := 1; j <= 10; j++ {
		channel_read <- j
		time.Sleep(2 * time.Second)
	}
	close(channel_read)

}
```

 结果图：

![image-20220703224002475](C:\Users\njupttest\AppData\Roaming\Typora\typora-user-images\image-20220703224002475.png)

回到上面代码， 创建两个即可读又可写的channel， 利用goroutine模拟三个工人去do work， 通过往jobs channel中发送数据模拟分发任务的场景，来实现一个简单的资源池，这边给我们提供一个思路，我们可以利用channel来保存资源来达到实现资源池的功能

## 借助channel实现资源池

```go
package main
 
import (
	"errors"
	"io"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)
 
//定义一个结构体,这个实体类型可以作为整体单元被复制,可以作为参数或返回值,或被存储到数组
type Pool struct {
	//定义成员,互斥锁类型
	m sync.Mutex
	//定义成员,通道类型,通道传递的是io.Closer类型
	resources chan io.Closer
	//定义工厂成员,类型是func()(io.Closer,error)
	//error是预定义类型,实际上是个interface接口类型
	factory func() (io.Closer, error)
	closed  bool
}
 
//定义变量,函数返回的是error类型
var ErrPoolClosed = errors.New("池已经关闭了")
 
//定义New方法,创建一个池,返回的是Pool类型的指针
//传入的参数是个函数类型func(io.Closer,error)和池的大小
func New(fn func() (io.Closer, error), size uint) (*Pool, error) {
	//使用结构体字面值给结构体成员赋值
	myPool := Pool{
		factory:   fn,
		resources: make(chan io.Closer, size),
	}
	//返回两个返回值
	return &myPool, nil
}
 
//从池中请求获取一个资源,给Pool类型定义的方法
//返回的值是io.Closer类型
func (p *Pool) Acquire() (io.Closer, error) {
	//基于select的多路复用
	//select会等待case中有能够执行的,才会去执行,等待其中一个能执行就执行
	//default分支会在所有case没法执行时,默认执行,也叫轮询channel
	select {
	case r, _ := <-p.resources:
		log.Printf("请求资源:来自通道 %d", r.(*dbConn).ID)
		return r, nil
	//如果缓冲通道中没有了,就会执行这里
	default:
		log.Printf("请求资源:创建新资源")
		return p.factory()
	}
}
 
//将一个使用后的资源放回池
//传入的参数是io.Closer类型
func (p *Pool) Release(r io.Closer) {
	//使用mutex互斥锁
	p.m.Lock()
	//解锁
	defer p.m.Unlock()
	//如果池都关闭了
	if p.closed {
		//关掉资源
		r.Close()
		return
	}
	//select多路选择
	//如果放回通道的时候满了,就关闭这个资源
	select {
	case p.resources <- r:
		log.Printf("释放资源:放入通道 %d", r.(*dbConn).ID)
	default:
		log.Printf("释放资源:关闭资源%d", r.(*dbConn).ID)
		r.Close()
	}
}
 
//关闭资源池,关闭通道,将通道中的资源关掉
func (p *Pool) Close() {
	p.m.Lock()
	defer p.m.Unlock()
	p.closed = true
	//先关闭通道再清空资源
	close(p.resources)
	//清空并关闭资源
	for r := range p.resources {
		r.Close()
	}
}
 
//定义全局常量
const (
	maxGoroutines = 20 //使用25个goroutine模拟同时的连接请求
	poolSize      = 2  //资源池中的大小
)
 
//定义结构体,模拟要共享的资源
type dbConn struct {
	//定义成员
	ID int32
}
 
//dbConn实现io.Closer接口
func (db *dbConn) Close() error {
	return nil
}
 
var idCounter int32 //定义一个全局的共享的变量,更新时用原子函数锁住
//定义方法,创建dbConn实例
//返回的是io.Closer类型和error类型
func createConn() (io.Closer, error) {
	//原子函数锁住,更新加1
	id := atomic.AddInt32(&idCounter, 1)
	log.Printf("创建新资源: %d", id)
	return &dbConn{id}, nil
}
func main() {
	//计数信号量
	var wg sync.WaitGroup
	//同时并发的数量
	wg.Add(maxGoroutines)
	myPool, _ := New(createConn, poolSize)
	//开25个goroutine同时查询
	for i := 0; i < maxGoroutines; i++ {
		//模拟请求
		time.Sleep(time.Duration(rand.Intn(2)) * time.Second)
		go func(gid int) {
			execQuery(gid, myPool)
			wg.Done()
		}(i)
	}
	//等待上面开的goroutine结束
	wg.Wait()
	myPool.Close()
}
 
//定义一个查询方法,参数是当前gorotineId和资源池
func execQuery(goroutineId int, pool *Pool) {
	//从池里请求资源,第一次肯定是没有的,就会创建一个dbConn实例
	conn, _ := pool.Acquire()
	//将创建的dbConn实例放入了资源池的缓冲通道里
	defer pool.Release(conn)
	//睡眠一下,模拟查询过程
	time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	log.Printf("执行查询...协程ID [%d] 资源ID [%d]", goroutineId, conn.(*dbConn).ID)
}
```

上述代码为模拟数据库请求连接，

### 代码理解

资源池的含义，开始时候我认为应该是可以控制开启goroutinue的数量，应为在我做的mapreduce项目中，采用并发方式生成key-value形式，但是对goroutinue的数量却是不可控制的，这就可能导致死锁、资源浪费、运行速度降低等等问题，但是经过多次查阅资料，资源池的意思指我将需要处理的任务（函数）作为资源，而非goroutinue，但是这并不表示goroutinue的数量不需要管理。

上述代码中，构建了一个Pool struct， 其中成员有 互斥锁（Mutex是一种用于多线程编程中，防止两条线程同时对同一公共资源（比如全局变量）进行读写的机制），存放io.Closer类型的channel，对于io.Closer类型需要多说一点，因为这是我原来没有接触过的

#### io.Closer

```go
Closer接口：
type Closer interface {
    Close() error
}
```

该接口比较简单，只有一个 Close() 方法，用于关闭数据流。

文件 (os.File)、归档（压缩包）、数据库连接、Socket 等需要手动关闭的资源都实现了 Closer 接口。

回到代码理解部分，在Pool结构体重定义工厂成员,类型是func()(io.Closer,error)，那么问题来了什么叫做工厂成员或者叫工厂函数--我的直观理解就是创建一个先不定义传入参数的函数，如下所示

```go
var example1 func()(x, y int)
```

上面定义了一个example1函数，我给出了返回类型但是没有给出输入参数类型--这就是工厂函数

最后还定义了一个布尔类型的表示Closed

针对Pool结构体定义了几个函数，New()、Acquire()、Release()、Close()分别表示创建资源池、从池中获取资源、资源重新放回到池中、关闭资源池

下面对每个函数做简单的理解

```go
func New(fn func() (io.Closer, error), size uint) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("Size value too small.")
	}

	return &Pool{
		factory:   fn,
		resources: make(chan io.Closer, size),
	}, nil
}
```

这里面还有一个小问题，结构体使用值传递和指针传递的区别，这个在后面在理解现在先给出链接

[Frequently Asked Questions (FAQ) - The Go Programming Language](https://go.dev/doc/faq#methods_on_values_or_pointers)

New函数传入类型有，工厂函数以及资源池的大小，返回值类型为Pool类型指针，初始化主要有初始化工厂函数以及资源通道channel

```go 
func (p *Pool) Acquire() (io.Closer, error) {
	select {
	// 检查是否有空闲的资源
	case r, ok := <-p.resources:
		log.Println("Acquire:", "Shared Resource")
		if !ok {
			return nil, ErrPoolClosed
		}
		return r, nil

	//如果没有空闲资源可用，提供一个新资源
	default:
		log.Println("Acquire:", "New Resource")
		return p.factory()
	}
}
```

其中判断通道是否关闭：

ok为true时，通道没有关闭，可以读到数据

ok为false时，通道关闭，没有数据可读

还有一点就是针对select的理解，基于select的多路复用，select会等待case中有能够执行的,才会去执行,等待其中一个能执行就执行，default分支会在所有case没法执行时,默认执行,也叫轮询channel
```go
//Release将一个使用后的资源放回池里
func (p *Pool) Release(r io.Closer) {
	//加锁，保证本操作和Close操作的安全
	p.m.Lock()
	defer p.m.Unlock()

	//如果池已经关闭，销毁这个资源
	if p.closed {
		r.Close()
		return
	}

	select {
	//将这个资源放入队列
	case p.resources <- r:
		log.Println("Release:", "In Queue")
	// 如果队列已满，则关闭这个资源
	default:
		log.Println("Release:", "Closing")
		r.Close()
	}
}

// Close会让资源池停止工作，并关闭所有现在的资源
func (p *Pool) Close() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return
	}

	//将池关闭
	p.closed = true

	//清空通道里的资源之前，将通道关闭，防止死锁
	close(p.resources)

	//关闭资源
	for r := range p.resources {
		r.Close()
	}
}
```

release函数和close函数 主要是对资源的一个回收工作不过多讲解



## 如何利用这个Pool框架应用到自己的MapReduce项目





