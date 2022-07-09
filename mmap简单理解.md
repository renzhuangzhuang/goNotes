[TOC]

# mmap简单理解

![mmap](D:\go学习笔记\笔记图片\mmap.jpg)

如图所示，mmap是一种将**文件/设备映射到内存**的方法，实现文件的磁盘地址和进程虚拟地址空间中的一段虚拟地址的一一映射关系。也就是说，可以在某个进程中通过操作这一段映射的内存，实现对文件的读写等操作。修改了这一段内存的内容，文件对应位置的内容也会同步修改，而读取这一段内存的内容，相当于读取文件对应位置的内容。mmap还有一个重要的特性：**减少内存的拷贝次数**。

在 linux 系统中，文件的读写操作通常通过 read 和 write 这两个系统调用来实现，这个过程会产生频繁的内存拷贝。比如 read 函数就涉及了 2 次内存拷贝：

1. 操作系统读取磁盘文件到页缓存；

2. 从页缓存将数据拷贝到 read 传递的 buf 中(例如进程中创建的byte数组)。

mmap 只需要一次拷贝。即操作系统读取磁盘文件到页缓存，进程内部直接通过指针方式修改映射的内存。因此 mmap 特别适合读写频繁的场景，既减少了内存拷贝次数，提高效率，又简化了操作。KV数据库 [bbolt](https://github.com/etcd-io/bbolt) 就使用了这个方法持久化数据。

具体的mmap原理，参考https://blog.csdn.net/ITer_ZC/article/details/44308729

## 标准库mmap

下面是结合网上给出案例，代码如下

```go
package main

import (
	"fmt"
	"log"
	"mmap"
)

func main() {
	at, _ := mmap.Open("file.txt")
	buff := make([]byte, 5)
	n, err := at.ReadAt(buff, 5)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n)
	_ = at.Close()
	fmt.Println(string(buff))
}
```

输出结果：

```go
5
 worl
```

分析： 首先读取本地文件，得到`mmap.ReaderAt`结构体的地址，也就是at里面存放的是比特数据流， 这是定义一个buff缓存（比特切片类型），`ReadAt`方法 第一个参数就是将at的数据copy给的切片，第二参数则是将at数据从哪一步复制的起始位置。

`file.txt`文件第一行为：

```txt
hello world hello golang
```

`ReadAt`从第六位置（5）开始读，因此读取的是一个空格，而buff大小为5因此结果为： worl

下面给出官方mmap实现的源码：

```go
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mmap provides a way to memory-map a file.
package mmap

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// debug is whether to print debugging messages for manual testing.
//
// The runtime.SetFinalizer documentation says that, "The finalizer for x is
// scheduled to run at some arbitrary time after x becomes unreachable. There
// is no guarantee that finalizers will run before a program exits", so we
// cannot automatically test that the finalizer runs. Instead, set this to true
// when running the manual test.
const debug = false

// ReaderAt reads a memory-mapped file.
//
// Like any io.ReaderAt, clients can execute parallel ReadAt calls, but it is
// not safe to call Close and reading methods concurrently.
type ReaderAt struct {
	data []byte
}

// Close closes the reader.
func (r *ReaderAt) Close() error {
	if r.data == nil {
		return nil
	}
	data := r.data
	r.data = nil
	if debug {
		var p *byte
		if len(data) != 0 {
			p = &data[0]
		}
		println("munmap", r, p)
	}
	runtime.SetFinalizer(r, nil)
	return syscall.UnmapViewOfFile(uintptr(unsafe.Pointer(&data[0])))
}

// Len returns the length of the underlying memory-mapped file.
func (r *ReaderAt) Len() int {
	return len(r.data)
}

// At returns the byte at index i.
func (r *ReaderAt) At(i int) byte {
	return r.data[i]
}

// ReadAt implements the io.ReaderAt interface.
func (r *ReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if r.data == nil {
		return 0, errors.New("mmap: closed")
	}
	if off < 0 || int64(len(r.data)) < off {
		return 0, fmt.Errorf("mmap: invalid ReadAt offset %d", off)
	}
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Open memory-maps the named file for reading.
func Open(filename string) (*ReaderAt, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := fi.Size()
	if size == 0 {
		return &ReaderAt{}, nil
	}
	if size < 0 {
		return nil, fmt.Errorf("mmap: file %q has negative size", filename)
	}
	if size != int64(int(size)) {
		return nil, fmt.Errorf("mmap: file %q is too large", filename)
	}

	low, high := uint32(size), uint32(size>>32)
	fmap, err := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, syscall.PAGE_READONLY, high, low, nil)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(fmap)
	ptr, err := syscall.MapViewOfFile(fmap, syscall.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil {
		return nil, err
	}
	data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)

	r := &ReaderAt{data: data}
	if debug {
		var p *byte
		if len(data) != 0 {
			p = &data[0]
		}
		println("mmap", r, p)
	}
	runtime.SetFinalizer(r, (*ReaderAt).Close)
	return r, nil
}
```

完成内存映射部分，需要通过两个步骤完成， 分别为`CreateFileMapping` 和 `MapViewOfFile` 两步才能完成内存映射。`MapViewOfFile` 返回映射成功的内存地址，因此可以直接将该地址转换成 byte 数组。

## 将`MapRedece`的文件处理换成mmap形式

先回顾一下，在我的框架下，读取文件采用的是块读取+计算块数量二者结合方法完成的，具体内容`MapReduce解决词频统计问题--单机版`文档。

为了对比给出块读取+计算块数量二者结合方法的代码：

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
)

const chunksize = 1 << (10) //二进制赋值

func main() {
	filename := "file.txt"

	fi, err := os.Stat(filename) //使用fi.size得到文件大小
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(float64(fi.Size())) 查看大小
	file_num := math.Ceil(float64(fi.Size()) / float64(chunksize)) // 得到文件的分块数
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	b1 := bufio.NewReader(file)

	for i := 0; i < int(file_num); i++ {
		p := make([]byte, chunksize)
		n1, err := b1.Read(p)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("......................")
		fmt.Println(string(p[:n1]))
	}
}
```

思考： 采用mmap读取文件，两个关键步骤，每次设置的缓存大小和复制时开始位置，确定这两个量之后就可以进行替换了。

下面是实现代码：

```go
package main

import (
	"fmt"
	"log"
	"math"
	"os"
)

const chunksize = 1 << (10)

func main() {
	filename := "file.txt"
	fi, err := os.Stat(filename) //使用fi.size得到文件大小
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(float64(fi.Size())) 查看大小
	file_num := math.Ceil(float64(fi.Size()) / float64(chunksize)) // 得到文件的分块数
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for i := 0; i < int(file_num); i++ {
		file_size := i * chunksize
		buff := make([]byte, chunksize)
		_, _ = file.ReadAt(buff, int64(file_size))
		fmt.Println(".................")
		fmt.Println(string(buff))
	}
}
```

输出结果：

```go
.................
hello world hello golang
你好
War and Peace

WAR AND PEACE

VOLUME 1

CHAPTER I
"Well, Prince, so Genoa and Lucca are now just family estates of the Buonapartes. But I warn you, if you don't tell me that this means war, if you still try to defend the infamies and horrors perpetrated by that Antichrist- I really believe he is Antichrist- I will have nothing more to do with you and you are no longer my friend, no longer my 'faithful slave,' as you call yourself! But how do you do? I see I have frightened you- sit down and tell me all the news."
It was in July, 1805, and the speaker was the well-known Anna Pavlovna Scherer, maid of honor and favorite of the Empress Marya Fedorovna. With these words she greeted Prince Vasili Kuragin, a man of high rank and importance, who was the first to arrive at her reception. Anna Pavlovna had had a cough for some days. She was, as she said, suffering from la grippe; grippe being then a new word in St. Petersburg, used only by the elite.
All her invitations witho
.................
ut exception, written in French, and delivered by a scarlet-liveried footman that morning, ran as follows:
"If you have nothing better to do, Count (or Prince), and if the prospect of spending an evening with a poor invalid is not too terrible, I shall be very charmed to see you tonight between 7 and 10- Annette Scherer."
"Heavens! what a virulent attack!" replied the prince, not in the least disconcerted by this reception. He had just entered, wearing an embroidered court uniform, knee breeches, and shoes, and had stars on his breast and a serene expression on his flat face. He spoke in that refined French in which our grandfathers not only spoke but thought, and with the gentle, patronizing intonation natural to a man of importance who had grown old in society and at court. He went up to Anna Pavlovna, kissed her hand, presenting to her his bald, scented, and shining head, and complacently seated himself on the sofa.
"First of all, dear friend, tell me how you are. Set your friend's mind at rest," said h
.................
e without altering his tone, beneath the politeness and affected sympathy of which indifference and even irony could be discerned.
"Can one be well while suffering morally? Can one be calm in times like these if one has any feeling?" said Anna Pavlovna. "You are staying the whole evening, I hope?"
"And the fete at the English ambassador's? Today is Wednesday. I must put in an appearance there," said the prince. "My daughter is coming for me to take me there."
"I thought today's fete had been canceled. I confess all these festivities and fireworks are becoming wearisome."
"If they had known that you wished it, the entertainment would have been put off," said the prince, who, like a wound-up clock, by force of habit said things he did not even wish to be believed.
"Don't tease! Well, and what has been decided about Novosiltsev's dispatch? You know everything."
"What can one say about it?" replied the prince in a cold, listless tone. "What has been decided? They have decided that Buonaparte has burnt his b
.................
oats, and I believe that we are ready to burn ours."
Prince Vasili always spoke languidly, like an actor repeating a stale part. Anna Pavlovna Scherer on the contrary, despite her forty years, overflowed with animation and impulsiveness. To be an enthusiast had become her social vocation and, sometimes even when she did not feel like it, she became enthusiastic in order not to disappoint the expectations of those who knew her. The subdued smile which, though it did not suit her faded features, always played round her lips expressed, as in a spoiled child, a continual consciousness of her charming defect, which she neither wished, nor could, nor considered it necessary, to correct.
In the midst of a conversation on political matters Anna Pavlovna burst out:
"Oh, don't speak to me of Austria. Perhaps I don't understand things, but Austria never has wished, and does not wish, for war. She is betraying us! Russia alone must save Europe. Our gracious sovereign recognizes his high vocation and will be true to it
.................
. That is the one thing I have faith in! Our good and wonderful sovereign has to perform the noblest role on earth, and he is so virtuous and noble that God will not forsake him. He will fulfill his vocation and crush the hydra of revolution, which has become more terrible than ever in the person of this murderer and villain! We alone must avenge the blood of the just one.... Whom, I ask you, can we rely on?... England with her commercial spirit will not and cannot understand the Emperor Alexander's loftiness of soul. She has refused to evacuate Malta. She wanted to find, and still seeks, some secret motive in our actions. What answer did Novosiltsev get? None. The English have not understood and cannot understand the self-abnegation of our Emperor who wants nothing for himself, but only desires the good of mankind. And what have they promised? Nothing! And what little they have promised they will not perform! Prussia has always declared that Buonaparte is invincible, and that all Europe is powerless before h
.................
im.... And I don't believe a word that Hardenburg says, or Haugwitz either. This famous Prussian neutrality is just a trap. I have faith only in God and the lofty destiny of our adored monarch. He will save Europe!"
She suddenly paused, smiling at her own impetuosity.
"I think," said the prince with a smile, "that if you had been sent instead of our dear Wintzingerode you would have captured the King of Prussia's consent by assault. You are so eloquent. Will you give me a cup of tea?"
"In a moment. A propos," she added, becoming calm again, "I am expecting two very interesting men tonight, le Vicomte de Mortemart, who is connected with the Montmorencys through the Rohans, one of the best French families. He is one of the genuine emigres, the good ones. And also the Abbe Morio. Do you know that profound thinker? He has been received by the Emperor. Had you heard?"
"I shall be delighted to meet them," said the prince. "But tell me," he added with studied carelessness as if it had only just occurred to him,
.................
though the question he was about to ask was the chief motive of his visit, "is it true that the Dowager Empress wants Baron Funke to be appointed first secretary at Vienna? The baron by all accounts is a poor creature."
Prince Vasili wished to obtain this post for his son, but others were trying through the Dowager Empress Marya Fedorovna to secure it for the baron.
Anna Pavlovna almost closed her eyes to indicate that neither she nor anyone else had a right to criticize what the Empress desired or was pleased with.
"Baron Funke has been recommended to the Dowager Empress by her sister," was all she said, in a dry and mournful tone.
As she named the Empress, Anna Pavlovna's face suddenly assumed an expression of profound and sincere devotion and respect mingled with sadness, and this occurred every time she mentioned her illustrious patroness. She added that Her Majesty had deigned to show Baron Funke beaucoup d'estime, and again her face clouded over with sadness.
The prince was silent and looked indiff
.................
erent. But, with the womanly and courtierlike quickness and tact habitual to her, Anna Pavlovna wished both to rebuke him (for daring to speak he had done of a man recommended to the Empress) and at the same time to console him, so she said:
"Now about your family. Do you know that since your daughter came out everyone has been enraptured by her? They say she is amazingly beautiful."
The prince bowed to signify his respect and gratitude.
"I often think," she continued after a short pause, drawing nearer to the prince and smiling amiably at him as if to show that political and social topics were ended and the time had come for intimate conversation- "I often think how unfairly sometimes the joys of life are distributed. Why has fate given you two such splendid children? I don't speak of Anatole, your youngest. I don't like him," she added in a tone admitting of no rejoinder and raising her eyebrows. "Two such charming children. And really you appreciate them less than anyone, and so you don't deserve to hav
.................
e them."
And she smiled her ecstatic smile.
"I can't help it," said the prince. "Lavater would have said I lack the bump of paternity."
"Don't joke; I mean to have a serious talk with you. Do you know I am dissatisfied with your younger son? Between ourselves" (and her face assumed its melancholy expression), "he was mentioned at Her Majesty's and you were pitied...."
The prince answered nothing, but she looked at him significantly, awaiting a reply. He frowned.
"What would you have me do?" he said at last. "You know I did all a father could for their education, and they have both turned out fools. Hippolyte is at least a quiet fool, but Anatole is an active one. That is the only difference between them." He said this smiling in a way more natural and animated than usual, so that the wrinkles round his mouth very clearly revealed something unexpectedly coarse and unpleasant.
"And why are children born to such men as you? If you were not a father there would be nothing I could reproach you with," said An
.................
na Pavlovna, looking up pensively.
"I am your faithful slave and to you alone I can confess that my children are the bane of my life. It is the cross I have to bear. That is how I explain it to myself. It can't be helped!"
He said no more, but expressed his resignation to cruel fate by a gesture. Anna Pavlovna meditated.
"Have you never thought of marrying your prodigal son Anatole?" she asked. "They say old maids have a mania for matchmaking, and though I don't feel that weakness in myself as yet,I know a little person who is very unhappy with her father. She is a relation of yours, Princess Mary Bolkonskaya."
Prince Vasili did not reply, though, with the quickness of memory and perception befitting a man of the world, he indicated by a movement of the head that he was considering this information.
"Do you know," he said at last, evidently unable to check the sad current of his thoughts, "that Anatole is costing me forty thousand rubles a year? And," he went on after a pause, "what will it be in five ye
.................
ars, if he goes on like this?" Presently he added: "That's what we fathers have to put up with.... Is this princess of yours rich?"
"Her father is very rich and stingy. He lives in the country. He is the well-known Prince Bolkonski who had to retire from the army under the late Emperor, and was nicknamed 'the King of Prussia.' He is very clever but eccentric, and a bore. The poor girl is very unhappy. She has a brother; I think you know him, he married Lise Meinen lately. He is an aide-de-camp of Kutuzov's and will be here tonight."
"Listen, dear Annette," said the prince, suddenly taking Anna Pavlovna's hand and for some reason drawing it downwards. "Arrange that affair for me and I shall always be your most devoted slave- slafe wigh an f, as a village elder of mine writes in his reports. She is rich and of good family and that's all I want."        
And with the familiarity and easy grace peculiar to him, he raised the maid of honor's hand to his lips, kissed it, and swung it to and fro as he lay back in his arm
.................
chair, looking in another direction.
"Attendez," said Anna Pavlovna, reflecting, "I'll speak to Lise, young Bolkonski's wife, this very evening, and perhaps the thing can be arranged. It shall be on your family's behalf that I'll start my apprenticeship as old maid."
```

最终结果和上面方法一致。

## 总结

在遇到读取大文件时候，已知的可以采用方法有两种：

1. 使用`bufio.NewReader` + 计算分块大小
2. 使用mmap内存映射方式，只需确定文件块数和复制的buff切片大小