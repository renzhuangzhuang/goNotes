# MapReduce解决词频统计问题--单机版

## 一、文件处理

第一步讲文件处理是因为我们的目标是对单词进行统计次数，在得到一个大文件时，我们可以**逐行** 或者**分块** 读取文件，特别的在分块读取时候我们可以借助**goroutine** 读取每一块分好的文件中的数据并将其转化成**key-value** 格式，平时在做go相关工作时候，不可避免的要进行读文件操作，下面将整理几种常见的读取文件的方法，以及在词频统计中运用的方法展示。

常见的文件方式有：**读取整个文件**、**逐行读取文件**、**逐字读取文件**、**分块读取文件**

### 读取整个文件

一般这种方式可以读取较小的文件，利用os包中的**ReadFile()**函数

```go
package main

import (
    "fmt"
    "log"
    "os"
)

func main() {
    content, err := os.ReadFile("file.txt")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(content))
}
```

输出：

```go
hello world hello golang
```

### 逐行读取文件

要逐行读取文件，我们可以使用比较方便的`bufio.Scanner`结构。它的构造函数`NewScanner()`接受一个打开的文件（记住在操作完成后关闭文件，例如通过 `defer`语句），并让您通过`Scan()`和`Text()`方法读取后续行。使用`Err()`方法，您可以检查文件读取过程中遇到的错误。

使用逐行读取文件

```go
package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
)

func main() {
    // open file
    f, err := os.Open("file.txt")
    if err != nil {
        log.Fatal(err)
    }
    // remember to close the file at the end of the program
    defer f.Close()

    // read the file line by line using scanner
    scanner := bufio.NewScanner(f)

    for scanner.Scan() {
        // do something with a line
        fmt.Printf("line: %s\n", scanner.Text())
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
}
```

输出：

```go
line: hello world hello golang
line: 你好
```

**逐行读取，可以减少内存的使用，但是会耗费大量的时间在IO**

### 逐字读取文件

逐字读取文件与逐行读取几乎相同。您只需要将`Scanner`的`split`功能从**默认的`ScanLines()`**函数更改为`ScanWords()`即可。

```go
package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
)

func main() {
    // open file
    f, err := os.Open("file.txt")
    if err != nil {
        log.Fatal(err)
    }
    // remember to close the file at the end of the program
    defer f.Close()

    // read the file word by word using scanner
    scanner := bufio.NewScanner(f)
    scanner.Split(bufio.ScanWords)

    for scanner.Scan() {
        // do something with a word
        fmt.Println(scanner.Text())
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
}
```

输出：

```go
hello
world
hello
golang
你好
```

### 分块读取文件

**第一种：**

当你有一个非常大的文件或不想将整个[文件存储](https://cloud.tencent.com/product/cfs?from=10680)在内存中时，您可以通过`固定大小的块`读取文件。在这种情况下，您需要创建一个指定大小`chunkSize`的byte切片作为缓冲区，用于存储后续读取的字节。使用`Read()`方法加载文件数据的下一个块。当发生`io.EOF`错误，指示文件结束，读取循环结束。

```go
package main

import (
    "fmt"
    "io"
    "log"
    "os"
)

const chunkSize = 10

func main() {
    // open file
    f, err := os.Open("file.txt")
    if err != nil {
        log.Fatal(err)
    }
    // remember to close the file at the end of the program
    defer f.Close()

    buf := make([]byte, chunkSize)

    for {
        n, err := f.Read(buf)
        if err != nil && err != io.EOF {
            log.Fatal(err)
        }

        if err == io.EOF {
            break
        }

        fmt.Println(string(buf[:n]))
    }
}
```

输出：

```go
hello worl
d hello go
lang
你�
��
```

**第二种：**

借助bufio包，直接上代码

```go
package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	filename := "file.txt"
	file, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	b1 := bufio.NewReader(file)
	p := make([]byte, 1024)
	n1, err := b1.Read(p)
	fmt.Println(n1)
	fmt.Println(string(p[:n1]))
}

```

输出：

```go
1024
hello world hello golang
你好
War and Peace

WAR AND PEACE

VOLUME 1

CHAPTER I
"Well, Prince, so Genoa and Lucca are now just family estates of the Buonapartes. But I warn you, if you don't tell me that this means war, if you still try to defend the infamies and horrors perpetrated by that Antichrist- I really believe he is Antichrist- I will have nothing more to do with you and you are no longer my friend, no longer my 'faithful slave,' as you call yourself! But how do you do? I see I have frightened you- sit down and tell me all the news."    
It was in July, 1805, and the speaker was the well-known Anna Pavlovna Scherer, maid of honor and favorite of the Empress Marya Fedorovna. With these words she greeted Prince Vasili Kuragin, a man of high rank and importance, who was the first to arrive at her reception. Anna Pavlovna had had a cough for some days. She was, as she said, suffering from la grippe; grippe being then a new word in St. Petersburg, used only by the elite.
All her invitations witho
```

**第三种：**

原来自己的mapreduce中采用的文件切块方法， 直接上代码(部分):

```go
package main

import (
    "os"
    "math"
)

func main() {
	// 文件划分大小
	file_size := 500 * 1024 //定义字节数
	txt_name := "1.txt"
	fi, err := os.Stat(txt_name) //使用fi.size得到文件大小
	if err != nil {
		fmt.Println(err)
	}
	file_num := math.Ceil(float64(fi.Size()) / float64(file_size)) // 得到文件的分块数

	//fmt.Println(fi.Size()) 得到字节数
	inputFile, err := os.OpenFile(txt_name, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("read file error =", err)
		return
	}
	defer inputFile.Close()
}
```

**第四种：**

安块读取文件时，我们可以结合二三两种方法，设定好块大小，并且计算出分块数量，借助goroutinue即可实现读取大规模文件。思路参考[MapReduce/main.go at main · leoneyar/MapReduce (github.com)](https://github.com/leoneyar/MapReduce/blob/main/project03/main.go)

话不多说，上代码

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

通过观察最后的打印效果可以达到我们一开始的要求，看结果：

```go
......................
hello world hello golang
你好
War and Peace

WAR AND PEACE

VOLUME 1

CHAPTER I
"Well, Prince, so Genoa and Lucca are now just family estates of the Buonapartes. But I warn you, if you don't tell me that this means war, if you still try to defend the infamies and horrors perpetrated by that Antichrist- I really believe he is Antichrist- I will have nothing more to do with you and you are no longer my friend, no longer my 'faithful slave,' as you call yourself! But how do you do? I see I have frightened you- sit down and tell me all the news."      
It was in July, 1805, and the speaker was the well-known Anna Pavlovna Scherer, maid of honor and favorite of the Empress Marya Fedorovna. With these words she greeted Prince Vasili Kuragin, a man of high rank and importance, who was the first to arrive at her reception. Anna Pavlovna had had a cough for some days. She was, as she said, suffering from la grippe; grippe being then a new word in St. Petersburg, used only by the elite.
All her invitations witho
......................
ut exception, written in French, and delivered by a scarlet-liveried footman that morning, ran as follows:
"If you have nothing better to do, Count (or Prince), and if the prospect of spending an evening with a poor invalid is not too terrible, I shall be very charmed to see you tonight between 7 and 10- Annette Scherer."
"Heavens! what a virulent attack!" replied the prince, not in the least disconcerted by this reception. He had just entered, wearing an embroidered court uniform, knee breeches, and shoes, and had stars on his breast and a serene expression on his flat face. He spoke in that refined French in which our grandfathers not only spoke but thought, and with the gentle, patronizing intonation natural to a man of importance who had grown old in society and at court. He went up to Anna Pavlovna, kissed her hand, presenting to her his bald, scented, and shining head, and complacently seated himself on the sofa.
"First of all, dear friend, tell me how you are. Set your friend's mind at rest," said h
......................
e without altering his tone, beneath the politeness and affected sympathy of which indifference and even irony could be discerned.
"Can one be well while suffering morally? Can one be calm in times like these if one has any feeling?" said Anna Pavlovna. "You are staying the whole evening, I hope?"
"And the fete at the English ambassador's? Today is Wednesday. I must put in an appearance there," said the prince. "My daughter is coming for me to take me there."
"I thought today's fete had been canceled. I confess all these festivities and fireworks are becoming wearisome."
"If they had known that you wished it, the entertainment would have been put off," said the prince, who, like a wound-up clock, by force of habit said things he did not even wish to be believed.
"Don't tease! Well, and what has been decided about Novosiltsev's dispatch? You know everything."
"What can one say about it?" replied the prince in a cold, listless tone. "What has been decided? They have decided that Buonaparte has burnt his b
......................
oats, and I believe that we are ready to burn ours."
Prince Vasili always spoke languidly, like an actor repeating a stale part. Anna Pavlovna Scherer on the contrary, despite her forty years, overflowed with animation and impulsiveness. To be an enthusiast had become her social vocation and, sometimes even when she did not feel like it, she became enthusiastic in order not to disappoint the expectations of those who knew her. The subdued smile which, though it did not suit her faded features, always played round her lips expressed, as in a spoiled child, a continual consciousness of her charming defect, which she neither wished, nor could, nor considered it necessary, to correct.
In the midst of a conversation on political matters Anna Pavlovna burst out:
"Oh, don't speak to me of Austria. Perhaps I don't understand things, but Austria never has wished, and does not wish, for war. She is betraying us! Russia alone must save Europe. Our gracious sovereign recognizes his high vocation and will be true to it
......................
. That is the one thing I have faith in! Our good and wonderful sovereign has to perform the noblest role on earth, and he is so virtuous and noble that God will not forsake him. He will fulfill his vocation and crush the hydra of revolution, which has become more terrible than ever in the person of this murderer and villain! We alone must avenge the blood of the just one.... Whom, I ask you, can we rely on?... England with her commercial spirit will not and cannot understand the Emperor Alexander's loftiness of soul. She has refused to evacuate Malta. She wanted to find, and still seeks, some secret motive in our actions. What answer did Novosiltsev get? None. The English have not understood and cannot understand the self-abnegation of our Emperor who wants nothing for himself, but only desires the good of mankind. And what have they promised? Nothing! And what little they have promised they will not perform! Prussia has always declared that Buonaparte is invincible, and that all Europe is powerless before h
......................
im.... And I don't believe a word that Hardenburg says, or Haugwitz either. This famous Prussian neutrality is just a trap. I have faith only in God and the lofty destiny of our adored monarch. He will save Europe!"
She suddenly paused, smiling at her own impetuosity.
"I think," said the prince with a smile, "that if you had been sent instead of our dear Wintzingerode you would have captured the King of Prussia's consent by assault. You are so eloquent. Will you give me a cup of tea?"
"In a moment. A propos," she added, becoming calm again, "I am expecting two very interesting men tonight, le Vicomte de Mortemart, who is connected with the Montmorencys through the Rohans, one of the best French families. He is one of the genuine emigres, the good ones. And also the Abbe Morio. Do you know that profound thinker? He has been received by the Emperor. Had you heard?"
"I shall be delighted to meet them," said the prince. "But tell me," he added with studied carelessness as if it had only just occurred to him,
......................
though the question he was about to ask was the chief motive of his visit, "is it true that the Dowager Empress wants Baron Funke to be appointed first secretary at Vienna? The baron by all accounts is a poor creature."
Prince Vasili wished to obtain this post for his son, but others were trying through the Dowager Empress Marya Fedorovna to secure it for the baron.
Anna Pavlovna almost closed her eyes to indicate that neither she nor anyone else had a right to criticize what the Empress desired or was pleased with.      
"Baron Funke has been recommended to the Dowager Empress by her sister," was all she said, in a dry and mournful tone.
As she named the Empress, Anna Pavlovna's face suddenly assumed an expression of profound and sincere devotion and respect mingled with sadness, and this occurred every time she mentioned her illustrious patroness. She added that Her Majesty had deigned to show Baron Funke beaucoup d'estime, and again her face clouded over with sadness.
The prince was silent and looked indiff
......................
erent. But, with the womanly and courtierlike quickness and tact habitual to her, Anna Pavlovna wished both to rebuke him (for daring to speak he had done of a man recommended to the Empress) and at the same time to console him, so she said:
"Now about your family. Do you know that since your daughter came out everyone has been enraptured by her? They say she is amazingly beautiful."
The prince bowed to signify his respect and gratitude.
"I often think," she continued after a short pause, drawing nearer to the prince and smiling amiably at him as if to show that political and social topics were ended and the time had come for intimate conversation- "I often think how unfairly sometimes the joys of life are distributed. Why has fate given you two such splendid children? I don't speak of Anatole, your youngest. I don't like him," she added in a tone admitting of no rejoinder and raising her eyebrows. "Two such charming children. And really you appreciate them less than anyone, and so you don't deserve to hav
......................
e them."
And she smiled her ecstatic smile.
"I can't help it," said the prince. "Lavater would have said I lack the bump of paternity."
"Don't joke; I mean to have a serious talk with you. Do you know I am dissatisfied with your younger son? Between ourselves" (and her face assumed its melancholy expression), "he was mentioned at Her Majesty's and you were pitied...."
The prince answered nothing, but she looked at him significantly, awaiting a reply. He frowned.
"What would you have me do?" he said at last. "You know I did all a father could for their education, and they have both turned out fools. Hippolyte is at least a quiet fool, but Anatole is an active one. That is the only difference between them." He said this smiling in a way more natural and animated than usual, so that the wrinkles round his mouth very clearly revealed something unexpectedly coarse and unpleasant.
"And why are children born to such men as you? If you were not a father there would be nothing I could reproach you with," said An
......................
na Pavlovna, looking up pensively.
"I am your faithful slave and to you alone I can confess that my children are the bane of my life. It is the cross I have to bear. That is how I explain it to myself. It can't be helped!"
He said no more, but expressed his resignation to cruel fate by a gesture. Anna Pavlovna meditated.
"Have you never thought of marrying your prodigal son Anatole?" she asked. "They say old maids have a mania for matchmaking, and though I don't feel that weakness in myself as yet,I know a little person who is very unhappy with her father. She is a relation of yours, Princess Mary Bolkonskaya."
Prince Vasili did not reply, though, with the quickness of memory and perception befitting a man of the world, he indicated by a movement of the head that he was considering this information.
"Do you know," he said at last, evidently unable to check the sad current of his thoughts, "that Anatole is costing me forty thousand rubles a year? And," he went on after a pause, "what will it be in five ye
......................
ars, if he goes on like this?" Presently he added: "That's what we fathers have to put up with.... Is this princess of yours rich?"
"Her father is very rich and stingy. He lives in the country. He is the well-known Prince Bolkonski who had to retire from the army under the late Emperor, and was nicknamed 'the King of Prussia.' He is very clever but eccentric, and a bore. The poor girl is very unhappy. She has a brother; I think you know him, he married Lise Meinen lately. He is an aide-de-camp of Kutuzov's and will be here tonight."
"Listen, dear Annette," said the prince, suddenly taking Anna Pavlovna's hand and for some reason drawing it downwards. "Arrange that affair for me and I shall always be your most devoted slave- slafe wigh an f, as a village elder of mine writes in his reports. She is rich and of good family and that's all I want."
And with the familiarity and easy grace peculiar to him, he raised the maid of honor's hand to his lips, kissed it, and swung it to and fro as he lay back in his arm
......................
chair, looking in another direction.
"Attendez," said Anna Pavlovna, reflecting, "I'll speak to Lise, young Bolkonski's wife, this very evening, and perhaps the thing can be arranged. It shall be on your family's behalf that I'll start my apprenticeship as old maid."
```

文件读取方式已经确定-- 分块读取中的第四种方法。除此以外，还有一种mmap方式读取文件（已经实现，在单独一个文本中有介绍）。那么下面工作就是对读取到的每块数据转化成key-value结构也就是**map**过程

## **二、Map过程**

我们通过文件处理过程将得到一块一块的比特形式的数据，并将其字符化，这时我们拿到的数据就是一块字符串，我们的需求是将每一个字符写成**kv**结构，k--字符名，v--出现次数。下面介绍两种生成kv的方式

**方式一：自带map形式**

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
)

const chunksize = 1 << (10) //二进制赋值

func main() {
	filename := "file.txt"
	ans := make(map[string]int)
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

		ss := strings.Fields(string(p[:n1]))

		for _, v := range ss {

			word := strings.ToLower(v)
			for len(word) > 0 && (word[0] < 'a' || word[0] > 'z') {
				word = word[1:]
			}
			for len(word) > 0 && (word[len(word)-1] < 'a' || word[len(word)-1] > 'z') {
				word = word[:len(word)-1]
			}
			ans[word]++

		}

	}
	fmt.Println(ans)

}
```

输出结果：

```go
map[:6 a:43 abbe:1 about:4 accounts:1 actions:1 active:1 actor:1 added:5 admitting:1 adored:1 affair:1 affected:1 after:2 again:2 aide-de-camp:1 alexander's:1 all:9 almost:1 alone:3 also:1 altering:1 always:4 am:3 amazingly:1 ambassador's:1 amiably:1 an:10 anatole:4 and:83 animated:1 animation:1 anna:12 annette:2 another:1 answer:1 answered:1 antichrist:2 any:1 anyone:2 appearance:1 appointed:1 appreciate:1 apprenticeship:1 are:10 arm:1 army:1 arrange:1 arranged:1 arrive:1 ars:1 as:12 ask:2 asked:1 assault:1 assumed:2 at:13 attack:1 attendez:1 austria:2 avenge:1 awaiting:1 b:1 back:1 bald:1 bane:1 baron:5 be:16 bear:1 beaucoup:1 beautiful:1 became:1 become:2 becoming:2 been:8 befitting:1 before:1 behalf:1 being:1 believe:3 believed:1 beneath:1 best:1 betraying:1 better:1 between:3 blood:1 bolkonskaya:1 bolkonski:1 bolkonski's:1 bore:1 born:1 both:2 bowed:1 breast:1 breeches:1 brother:1 bump:1 buonaparte:2 buonapartes:1 burn:1 burnt:1 burst:1 but:12 by:12 call:1 calm:2 came:1 can:6 can't:2 canceled:1 cannot:2 captured:1 carelessness:1 chair:1 chapter:1 charmed:1 charming:2 check:1 chief:1 child:1 children:4 clearly:1 clever:1 clock:1 closed:1 clouded:1 coarse:1 cold:1 come:1 coming:1 commercial:1 complacently:1 confess:2 connected:1 consciousness:1 consent:1 considered:1 considering:1 console:1 continual:1 continued:1 contrary:1 conversation:2 correct:1 costing:1 cough:1 could:4 count:1 country:1 court:2 courtierlike:1 creature:1 criticize:1 cross:1 cruel:1 crush:1 cup:1 current:1 d'estime:1 daring:1 daughter:2 days:1 de:1 dear:3 decided:3 declared:1 defect:1 defend:1 deigned:1 delighted:1 delivered:1 deserve:1 desired:1 desires:1 despite:1 destiny:1 devoted:1 devotion:1 did:6 difference:1 direction:1 disappoint:1 discerned:1 disconcerted:1 dispatch:1 dissatisfied:1 distributed:1 do:9 does:1 don't:10 done:1 dowager:3 down:1 downwards:1 drawing:2 dry:1 e:2 earth:1 easy:1 eccentric:1 ecstatic:1 education:1 either:1 elder:1 elite:1 eloquent:1 else:1 embroidered:1 emigres:1 emperor:4 empress:7 ended:1 england:1 english:2 enraptured:1 entered:1 entertainment:1 enthusiast:1 enthusiastic:1 erent:1 estates:1 europe:3 evacuate:1 even:3 evening:3 ever:1 every:1 everyone:1 everything:1 evidently:1 exception:1 expectations:1 expecting:1 explain:1 expressed:2 expression:3 eyebrows:1 eyes:1 f:1 face:4 faded:1 faith:2 faithful:2 familiarity:1 families:1 family:3 family's:1 famous:1 fate:2 father:4 fathers:1 favorite:1 features:1 fedorovna:2 feel:2 feeling:1 festivities:1 fete:2 find:1 fireworks:1 first:3 five:1 flat:1 follows:1 fool:1 fools:1 footman:1 for:12 force:1 forsake:1 forty:2 french:3 friend:2 friend's:1 frightened:1 fro:1 from:2 frowned:1 fulfill:1 funke:3 genoa:1 gentle:1 genuine:1 gesture:1 get:1 girl:1 give:1 given:1 god:2 goes:1 golang:1 good:4 grace:1 gracious:1 grandfathers:1 gratitude:1 greeted:1 grippe:2 grown:1 h:2 habit:1 habitual:1 had:16 hand:3 hardenburg:1 has:14 haugwitz:1 hav:1 have:19 he:31 head:2 heard:1 heavens:1 hello:2 help:1 helped:1 her:25 here:1 high:2 him:9 himself:2 hippolyte:1 his:16 honor:1 honor's:1 hope:1 horrors:1 how:4 hydra:1 i:38 i'll:2 if:11 illustrious:1 im:1 impetuosity:1 importance:2 impulsiveness:1 in:27 indicate:1 indicated:1 indiff:1 indifference:1 infamies:1 information:1 instead:1 interesting:1 intimate:1 intonation:1 invalid:1 invincible:1 invitations:1 irony:1 is:30 it:19 its:1 joke:1 joys:1 july:1 just:5 king:2 kissed:2 knee:1 knew:1 know:8 known:1 kuragin:1 kutuzov's:1 la:1 lack:1 languidly:1 last:2 late:1 lately:1 lavater:1 lay:1 le:1 least:2 less:1 life:2 like:6 lips:2 lise:2 listen:1 listless:1 little:2 lives:1 loftiness:1 lofty:1 longer:2 looked:2 looking:2 lucca:1 maid:3 maids:1 majesty:1 majesty's:1 malta:1 man:4 mania:1 mankind:1 married:1 marrying:1 mary:1 marya:2 matchmaking:1 matters:1 me:11 mean:1 means:1 meditated:1 meet:1 meinen:1 melancholy:1 memory:1 men:2 mentioned:2 midst:1 mind:1 mine:1 mingled:1 moment:1 monarch:1 montmorencys:1 morally:1 more:4 morio:1 morning:1 mortemart:1 most:1 motive:2 mournful:1 mouth:1 movement:1 murderer:1 must:3 my:6 myself:2 na:1 named:1 natural:2 nearer:1 necessary:1 neither:2 neutrality:1 never:2 new:1 news:1 nicknamed:1 no:4 noble:1 noblest:1 none:1 nor:3 not:14 nothing:6 novosiltsev:1 novosiltsev's:1 now:2 oats:1 obtain:1 occurred:2 of:45 off:1 often:2 oh:1 old:3 on:10 one:9 ones:1 only:6 or:3 order:1 others:1 our:7 ours:1 ourselves:1 out:3 over:1 overflowed:1 own:1 part:1 paternity:1 patroness:1 patronizing:1 pause:2 paused:1 pavlovna:11 pavlovna's:2 peace:2 peculiar:1 pensively:1 perception:1 perform:2 perhaps:2 perpetrated:1 person:2 petersburg:1 pitied:1 played:1 pleased:1 politeness:1 political:2 poor:3 post:1 powerless:1 presenting:1 presently:1 prince:19 princess:2 prodigal:1 profound:2 promised:2 propos:1 prospect:1 prussia:2 prussia's:1 prussian:1 put:3 question:1 quickness:2 quiet:1 raised:1 raising:1 ran:1 rank:1 ready:1 really:2 reason:1 rebuke:1 received:1 reception:2 recognizes:1 recommended:2 refined:1 reflecting:1 refused:1 rejoinder:1 relation:1 rely:1 repeating:1 replied:2 reply:2 reports:1 reproach:1 resignation:1 respect:2 rest:1 retire:1 revealed:1 revolution:1 rich:3 right:1 rohans:1 role:1 round:2 rubles:1 russia:1 sad:1 sadness:2 said:19 same:1 save:2 say:3 says:1 scarlet-liveried:1 scented:1 scherer:3 seated:1 secret:1 secretary:1 secure:1 see:2 seeks:1 self-abnegation:1 sent:1 serene:1 serious:1 set:1 shall:4 she:26 shining:1 shoes:1 short:1 show:2 significantly:1 signify:1 silent:1 since:1 sincere:1 sister:1 sit:1 slafe:1 slave:3 smile:3 smiled:1 smiling:3 so:6 social:2 society:1 sofa:1 some:3 something:1 sometimes:2 son:3 soul:1 sovereign:2 speak:4 speaker:1 spending:1 spirit:1 splendid:1 spoiled:1 spoke:3 st:1 stale:1 stars:1 start:1 staying:1 still:2 stingy:1 studied:1 subdued:1 such:3 suddenly:3 suffering:2 suit:1 swung:1 sympathy:1 tact:1 take:1 taking:1 talk:1 tea:1 tease:1 tell:4 terrible:2 than:3 that:28 that's:2 the:84 their:1 them:4 then:1 there:3 these:3 they:8 thing:2 things:2 think:4 thinker:1 this:11 those:1 though:4 thought:3 thoughts:1 thousand:1 through:2 time:3 times:1 to:51 today:1 today's:1 tone:4 tonight:3 too:1 topics:1 trap:1 true:2 try:1 trying:1 turned:1 two:3 unable:1 under:1 understand:3 understood:1 unexpectedly:1 unfairly:1 unhappy:2 uniform:1 unpleasant:1 up:3 us:1 used:1 usual:1 ut:1 vasili:4 very:8 vicomte:1 vienna:1 village:1 villain:1 virtuous:1 virulent:1 visit:1 vocation:3 volume:1 want:1 wanted:1 wants:2 war:4 warn:1 was:12 way:1 we:4 weakness:1 wearing:1 wearisome:1 wednesday:1 well:3 well-known:2 went:2 were:4 what:11 when:1 which:5 while:1 who:8 whole:1 whom:1 why:2 wife:1 wigh:1 will:10 wintzingerode:1 wish:2 wished:5 with:20 witho:1 without:1 womanly:1 wonderful:1 word:2 words:1 world:2 would:5 wound-up:1 wrinkles:1 writes:1 written:1 ye:1 year:1 years:1 yet,i:1 you:37 young:1 younger:1 youngest:1 your:9 yours:2 yourself:1]
```

这里面只是简单的 生成kv并对相同的k的值做加一处理

****

**方式二：构建kv形式的结构体**

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"unicode"
)

type KeyValue struct {
	key   string
	value string
}

const chunksize = 1 << (10) //二进制赋值

func mapF(contents string) []KeyValue {
	//debug("Map %v\n", value)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	keys := strings.FieldsFunc(contents, f)
	var res []KeyValue
	for _, key := range keys {
		res = append(res, KeyValue{key, "1"})
	}
	return res
}

func main() {
	var wg sync.WaitGroup
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
	ans := []KeyValue{}
	b1 := bufio.NewReader(file)
	for i := 0; i < int(file_num); i++ {
		p := make([]byte, chunksize)
		b1.Read(p)
		wg.Add(1)
		go func(b []byte) {
			defer wg.Done()
			res1 := mapF(string(b))
			ans = append(ans, res1...)
		}(p)
	}
	wg.Wait()
	fmt.Println(ans)
}
```

第二种方法则是定义一个kv的结构体，并定义一个map函数，返回值类型为kv结构体类型，上述代码结果为：

```go
[{hello 1} {world 1} {hello 1} {golang 1} {你好 1} {War 1} {and 1} {Peace 1} {WAR 1} {AND 1} {PEACE 1} {VOLUME 1} {1 1} {CHAPTER 1} {I 1} {Well 1} {Prince 1} {so 1} {Genoa 1} {and 1} {Lucca 1} {are 1} {now 1} {just 1} {family 1} {estates 1} {of 1} {the 1} {Buonapartes 1} {But 1} {I 1} {warn 1} {you 1} {if 1} {you 1} {don 1} {t 1} {tell 1} {me 1} {that 1} {this 1} {means 1} {war 1} {if 1} {you 1} {still 1} {try 1} {to 1} {defend 1} {the 1} {infamies 1} {and 1} {horrors 1} {perpetrated 1} {by 1} {that 1} {Antichrist 1} {I 1} {really 1} {believe 1} {he 1} {is 1} {Antichrist 1} {I 1} {will 1} {have 1} {nothing 1} {more 1} {to 1} {do 1} {with 1} {you 1} {and 1} {you 1} {are 1} {no 1} {longer 1} {my 1} {friend 1} {no 1} {longer 1} {my 1} {faithful 1} {slave 1} {as 1} {you 1} {call 1} {yourself 1} {But 1} {how 1} {do 1} {you 1} {do 1} {I 1} {see 1} {I 1} {have 1} {frightened 1} {you 1} {sit 1} {down 1} {and 1} {tell 1} {me 1} {all 1} {the 1} {news 1} {It 1} {was 1} {in 1} {July 1} {1805 1} {and 1} {the 1} {speaker 1} {was 1} {the 1} {well 1} {known 1} {Anna 1} {Pavlovna 1} {Scherer 1} {maid 1} {of 1} {honor 1} {and 1} {favorite 1} {of 1} {the 1} {Empress 1} {Marya 1} {Fedorovna 1} {With 1} {these 1} {words 1} {she 1} {greeted 1} {Prince 1} {Vasili 1} {Kuragin 1} {a 1} {man 1} {of 1} {high 1} {rank 1} {and 1} {importance 1} {who 1} {was 1} {the 1} {first 1} {to 1} {arrive 1} {at 1} {her 1} {reception 1} {Anna 1} {Pavlovna 1} {had 1} {had 1} {a 1} {cough 1} {for 1} {some 1} {days 1} {She 1} {was 1} {as 1} {she 1} {said 1} {suffering 1} {from 1} {la 1} {grippe 1} {grippe 1} {being 1} {then 1} {a 1} {new 1} {word 1} {in 1} {St 1} {Petersburg 1} {used 1} {only 1} {by 1} {the 1} {elite 1} {All 1} {her 1} {invitations 1} {witho 1} { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } { } {ut 1} {exception 1} {written 1} {in 1} {French 1} {and 1} {delivered 1} {by 1} {a 1} {scarlet 1} {liveried 1} {footman 1} {that 1} {morning 1} {ran 1} {as 1} {follows 1} {If 1} {you 1} {have 1} {nothing 1} {better 1} {to 1} {do 1} {Count 1} {or 1} {Prince 1} {and 1} {if 1} {the 1} {prospect 1} {of 1} {spending 1} {an 1} {evening 1} {with 1} {a 1} {poor 1} {invalid 1} {is 1} {not 1} {too 1} {terrible 1} {I 1} {shall 1} {be 1} {very 1} {charmed 1} {to 1} {see 1} {you 1} {tonight 1} {between 1} {7 1} {and 1} {10 1} {Annette 1} {Scherer 1} {Heavens 1} {what 1} {a 1} {virulent 1} {attack 1} {replied 1} {the 1} {prince 1} {not 1} {in 1} {the 1} {least 1} {disconcerted 1} {by 1} {this 1} {reception 1} {He 1} {had 1} {just 1} {entered 1} {wearing 1} {an 1} {embroidered 1} {court 1} {uniform 1} {knee 1} {breeches 1} {and 1} {shoes 1} {and 1} {had 1} {stars 1} {on 1} {his 1} {breast 1} {and 1} {a 1} {serene 1} {expression 1} {on 1} {his 1} {flat 1} {face 1} {He 1} {spoke 1} {in 1} {that 1} {refined 1} {French 1} {in 1} {which 1} {our 1} {grandfathers 1} {not 1} {only 1} {spoke 1} {but 1} {thought 1} {and 1} {with 1} {the 1} {gentle 1} {patronizing 1} {intonation 1} {natural 1} {to 1} {a 1} {man 1} {of 1} {importance 1} {who 1} {had 1} {grown 1} {old 1} {in 1} {society 1} {and 1} {though 1} {the 1} {question 1} {he 1} {was 1} {about 1} {to 1} {ask 1} {was 1} {the 1} {chief 1} {motive 1} {of 1} {his 1} {visit 1} {is 1} {it 1} {true 1} {that 1} {the 1} {Dowager 1} {Empress 1} {wants 1} {Baron 1} {Funke 1} {to 1} {be 1} {appointed 1} {first 1} {secretary 1} {at 1} {Vienna 1} {The 1} {baron 1} {by 1} {all 1} {accounts 1} {is 1} {a 1} {poor 1} {creature 1} {Prince 1} {Vasili 1} {wished 1} {to 1} {obtain 1} {this 1} {post 1} {for 1} {his 1} {son 1} {but 1} {others 1} {were 1} {trying 1} {through 1} {the 1} {Dowager 1} {Empress 1} {Marya 1} {Fedorovna 1} {to 1} {secure 1} {it 1} {for 1} {the 1} {baron 1} {Anna 1} {Pavlovna 1} {almost 1} {closed 1} {her 1} {eyes 1} {to 1} {indicate 1} {that 1} {neither 1} {she 1} {nor 1} {anyone 1} {else 1} {had 1} {a 1} {right 1} {to 1} {criticize 1} {what 1} {the 1} {Empress 1} {desired 1} {or 1} {was 1} {pleased 1} {with 1} {Baron 1} {Funke 1} {has 1} {been 1} {recommended 1} {to 1} {the 1} {Dowager 1} {Empress 1} {by 1} {her 1} {sister 1} {was 1} {all 1} {she 1} {said 1} {in 1} {a 1} {dry 1} {and 1} {mournful 1} {tone 1} {As 1} {she 1} {named 1} {the 1} {Empress 1} {Anna 1} {Pavlovna 1} {s 1} {face 1} {suddenly 1} {assumed 1} {an 1} {expression 1} {of 1} {profound 1} {and 1} {sincere 1} {devotion 1} {and 1} {respect 1} {mingled 1} {with 1} {sadness 1} {and 1} {this 1} {occurred 1} {every 1} {time 1} {she 1} {mentioned 1} {her 1} {illustrious 1} {patroness 1} {She 1} {added 1} {that 1} {Her 1} {Majesty 1} {had 1} {deigned 1} {to 1} {show 1} {Baron 1} {Funke 1} {beaucoup 1} {d 1} {estime 1} {and 1} {again 1} {her 1} {face 1} {clouded 1} {over 1} {with 1} {sadness 1} {The 1} {prince 1} {was 1} {silent 1} {and 1} {looked 1} {indiff 1} { } { } { } {erent 1} {But 1} {with 1} {the 1} {womanly 1} {and 1} {courtierlike 1} {quickness 1} {and 1} {tact 1} {habitual 1} {to 1} {her 1} {Anna 1} {Pavlovna 1} {wished 1} {both 1} {to 1} {rebuke 1} {him 1} {for 1} {daring 1} {to 1} {speak 1} {he 1} {had 1} {done 1} {of 1} {a 1} {man 1} {recommended 1} {to 1} {the 1} {Empress 1} {and 1} {at 1} {the 1} {same 1} {time 1} {to 1} {console 1} {him 1} {so 1} {she 1} {said 1} {Now 1} {about 1} {your 1} {family 1} {Do 1} {you 1} {know 1} {that 1} {since 1} {your 1} {daughter 1} {came 1} {out 1} {everyone 1} {has 1} {been 1} {enraptured 1} {by 1} {her 1} {They 1} {say 1} {she 1} {is 1} {amazingly 1} {beautiful 1} {The 1} {prince 1} {bowed 1} {to 1} {signify 1} {his 1} {respect 1} {and 1} {gratitude 1} {I 1} {often 1} {think 1} {she 1} {continued 1} {after 1} {a 1} {short 1} {pause 1} {drawing 1} {nearer 1} {to 1} {the 1} {prince 1} {and 1} {smiling 1} {amiably 1} {at 1} {him 1} {as 1} {if 1} {to 1} {show 1} {that 1} {political 1} {and 1} {social 1} {topics 1} {were 1} {ended 1} {and 1} {the 1} {time 1} {had 1} {come 1} {for 1} {intimate 1} {conversation 1} {I 1} {often 1} {think 1} {how 1} {unfairly 1} {sometimes 1} {the 1} {joys 1} {of 1} {life 1} {are 1} {distributed 1} {Why 1} {has 1} {fate 1} {given 1} {you 1} {two 1} {such 1} {splendid 1} {children 1} {I 1} {don 1} {t 1} {speak 1} {of 1} {Anatole 1} {your 1} {youngest 1} {I 1} {don 1} {t 1} {like 1} {him 1} {she 1} {added 1} {in 1} {a 1} {tone 1} {admitting 1} {of 1} {no 1} {rejoinder 1} {and 1} {raising 1} {her 1} {eyebrows 1} {Two 1} {such 1} {charming 1} {children 1} {And 1} {really 1} {you 1} {appreciate 1} {them 1} {less 1} {than 1} {anyone 1} {and 1} {so 1} {you 1} {don 1} {t 1} {That 1} {is 1} {the 1} {one 1} {thing 1} {I 1} {have 1} {faith 1} {in 1} {Our 1} {good 1} {and 1} {wonderful 1} {sovereign 1} {has 1} {to 1} {perform 1} {the 1} {noblest 1} {role 1} {on 1} {earth 1} {and 1} {he 1} {is 1} {so 1} {virtuous 1} {and 1} {noble 1} {that 1} {God 1} {will 1} {not 1} {forsake 1} {him 1} {He 1} {will 1} {fulfill 1} {his 1} {vocation 1} {and 1} {crush 1} {the 1} {hydra 1} {of 1} {revolution 1} {which 1} {has 1} {become 1} {more 1} {terrible 1} {than 1} {ever 1} {in 1} {the 1} {person 1} {of 1} {this 1} {murderer 1} {and 1} {villain 1} {We 1} {alone 1} {must 1} {avenge 1} {the 1} {blood 1} {of 1} {the 1} {just 1} {one 1} {Whom 1} {I 1} {ask 1} {you 1} {can 1} {we 1} {rely 1} {on 1} {England 1} {with 1} {her 1} {commercial 1} {spirit 1} {will 1} {not 1} {and 1} {cannot 1} {understand 1} {the 1} {Emperor 1} {Alexander 1} {s 1} {loftiness 1} {of 1} {soul 1} {She 1} {has 1} {refused 1} {to 1} {evacuate 1} {Malta 1} {She 1} {wanted 1} {to 1} {find 1} {and 1} {still 1} {seeks 1} {some 1} {secret 1} {motive 1} {in 1} {our 1} {actions 1} {What 1} {answer 1} {did 1} {Novosiltsev 1} {get 1} {None 1} {The 1} {English 1} {have 1} {not 1} {understood 1} {and 1} {cannot 1} {understand 1} {the 1} {self 1} {abnegation 1} {of 1} {our 1} {Emperor 1} {who 1} {wants 1} {nothing 1} {for 1} {himself 1} {but 1} {only 1} {desires 1} {the 1} {good 1} {of 1} {mankind 1} {And 1} {what 1} {have 1} {they 1} {promised 1} {Nothing 1} {And 1} {what 1} {little 1} {they 1} {have 1} {promised 1} {they 1} {will 1} {not 1} {perform 1} {Prussia 1} {has 1} {always 1} {declared 1} {that 1} {Buonaparte 1} {is 1} {invincible 1} {and 1} {that 1} {all 1} {Europe 1} {is 1} {powerless 1} {before 1} {h 1} {oats 1} {and 1} {I 1} {believe 1} {that 1} {we 1} {are 1} {ready 1} {to 1} {burn 1} {ours 1} {Prince 1} {Vasili 1} {always 1} {spoke 1} {languidly 1} {like 1} {an 1} {actor 1} {repeating 1} {a 1} {stale 1} {part 1} {Anna 1} {Pavlovna 1} {Scherer 1} {on 1} {the 1} {contrary 1} {despite 1} {her 1} {forty 1} {years 1} {overflowed 1} {with 1} {animation 1} {and 1} {impulsiveness 1} {To 1} {be 1} {an 1} {enthusiast 1} {had 1} {become 1} {her 1} {social 1} {vocation 1} {and 1} {sometimes 1} {even 1} {when 1} {she 1} {did 1} {not 1} {feel 1} {like 1} {it 1} {she 1} {became 1} {enthusiastic 1} {in 1} {order 1} {not 1} {to 1} {disappoint 1} {the 1} {expectations 1} {of 1} {those 1} {who 1} {knew 1} {her 1} {The 1} {subdued 1} {smile 1} {which 1} {though 1} {it 1} {did 1} {not 1} {suit 1} {her 1} {faded 1} {features 1} {always 1} {played 1} {round 1} {her 1} {lips 1} {expressed 1} {as 1} {in 1} {a 1} {spoiled 1} {child 1} {a 1} {continual 1} {consciousness 1} {of 1} {her 1} {charming 1} {defect 1} {which 1} {she 1} {neither 1} {wished 1} {nor 1} {could 1} {nor 1} {considered 1} {it 1} {necessary 1} {to 1} {correct 1} {In 1} {the 1} {midst 1} {of 1} {a 1} {conversation 1} {on 1} {political 1} {matters 1} {Anna 1} {Pavlovna 1} {burst 1} {out 1} {Oh 1} {don 1} {t 1} {speak 1} {to 1} {me 1} {of 1} {Austria 1} {Perhaps 1} {I 1} {don 1} {t 1} {understand 1} {things 1} {but 1} {Austria 1} {never 1} {has 1} {wished 1} {and 1} {does 1} {not 1} {wish 1} {for 1} {war 1} {She 1} {is 1} {betraying 1} {us 1} {Russia 1} {alone 1} {must 1} {save 1} {Europe 1} {Our 1} {gracious 1} {sovereign 1} {recognizes 1} {his 1} {high 1} {vocation 1} {and 1} {will 1} {be 1} {true 1} {to 1} {it 1} {im 1} {And 1} {I 1} {don 1} {t 1} {believe 1} {a 1} {word 1} {that 1} {Hardenburg 1} {says 1} {or 1} {Haugwitz 1} {either 1} {This 1} {famous 1} {Prussian 1} {neutrality 1} {is 1} {just 1} {a 1} {trap 1} {I 1} {have 1} {faith 1} {only 1} {in 1} {God 1} {and 1} {the 1} {lofty 1} {destiny 1} {of 1} {our 1} {adored 1} {monarch 1} {He 1} {will 1} {save 1} {Europe 1} {She 1} {suddenly 1} {paused 1} {smiling 1} {at 1} {her 1} {own 1} {impetuosity 1} {I 1} {think 1} {said 1} {the 1} {prince 1} {with 1} {a 1} {smile 1} {that 1} {if 1} {you 1} {had 1} {been 1} {sent 1} {instead 1} {of 1} {our 1} {dear 1} {Wintzingerode 1} {you 1} {would 1} {have 1} {captured 1} {the 1} {King 1} {of 1} {Prussia 1} {s 1} {consent 1} {by 1} {assault 1} {You 1} {are 1} {so 1} {eloquent 1} {Will 1} {you 1} {give 1} {me 1} {a 1} {cup 1} {of 1} {tea 1} {In 1} {a 1} {moment 1} {A 1} {propos 1} {she 1} {added 1} {becoming 1} {calm 1} {again 1} {I 1} {am 1} {expecting 1} {two 1} {very 1} {interesting 1} {men 1} {tonight 1} {le 1} {Vicomte 1} {de 1} {Mortemart 1} {who 1} {is 1} {connected 1} {with 1} {the 1} {Montmorencys 1} {through 1} {the 1} {Rohans 1} {one 1} {of 1} {the 1} {best 1} {French 1} {families 1} {He 1} {is 1} {one 1} {of 1} {the 1} {genuine 1} {emigres 1} {the 1} {good 1} {ones 1} {And 1} {also 1} {the 1} {Abbe 1} {Morio 1} {Do 1} {you 1} {know 1} {that 1} {profound 1} {thinker 1} {He 1} {has 1} {been 1} {received 1} {by 1} {the 1} {Emperor 1} {Had 1} {you 1} {heard 1} {I 1} {shall 1} {be 1} {delighted 1} {to 1} {meet 1} {them 1} {said 1} {the 1} {prince 1} {But 1} {tell 1} {me 1} {he 1} {added 1} {with 1} {studied 1} {carelessness 1} {as 1} {if 1} {it 1} {had 1} {only 1} {just 1} {occurred 1} {to 1} {him 1} {ars 1} {if 1} {he 1} {goes 1} {on 1} {like 1} {this 1} {Presently 1} {he 1} {added 1} {That 1} {s 1} {what 1} {we 1} {fathers 1} {have 1} {to 1} {put 1} {up 1} {with 1} {Is 1} {this 1} {princess 1} {of 1} {yours 1} {rich 1} {Her 1} {father 1} {is 1} {very 1} {rich 1} {and 1} {stingy 1} {He 1} {lives 1} {in 1} {the 1} {country 1} {He 1} {is 1} {the 1} {well 1} {known 1} {Prince 1} {Bolkonski 1} {who 1} {had 1} {to 1} {retire 1} {from 1} {the 1} {army 1} {under 1} {the 1} {late 1} {Emperor 1} {and 1} {was 1} {nicknamed 1} {the 1} {King 1} {of 1} {Prussia 1} {He 1} {is 1} {very 1} {clever 1} {but 1} {eccentric 1} {and 1} {a 1} {bore 1} {The 1} {poor 1} {girl 1} {is 1} {very 1} {unhappy 1} {She 1} {has 1} {a 1} {brother 1} {I 1} {think 1} {you 1} {know 1} {him 1} {he 1} {married 1} {Lise 1} {Meinen 1} {lately 1} {He 1} {is 1} {an 1} {aide 1} {de 1} {camp 1} {of 1} {Kutuzov 1} {s 1} {and 1} {will 1} {be 1} {here 1} {tonight 1} {Listen 1} {dear 1} {Annette 1} {said 1} {the 1} {prince 1} {suddenly 1} {taking 1} {Anna 1} {Pavlovna 1} {s 1} {hand 1} {and 1} {for 1} {some 1} {reason 1} {drawing 1} {it 1} {downwards 1} {Arrange 1} {that 1} {affair 1} {for 1} {me 1} {and 1} {I 1} {shall 1} {always 1} {be 1} {your 1} {most 1} {devoted 1} {slave 1} {slafe 1} {wigh 1} {an 1} {f 1} {as 1} {a 1} {village 1} {elder 1} {of 1} {mine 1} {writes 1} {in 1} {his 1} {reports 1} {She 1} {is 1} {rich 1} {and 1} {of 1} {good 1} {family 1} {and 1} {that 1} {s 1} {all 1} {I 1} {want 1} {And 1} {with 1} {the 1} {familiarity 1} {and 1} {easy 1} {grace 1} {peculiar 1} {to 1} {him 1} {he 1} {raised 1} {the 1} {maid 1} {of 1} {honor 1} {s 1} {hand 1} {to 1} {his 1} {lips 1} {kissed 1} {it 1} {and 1} {swung 1} {it 1} {to 1} {and 1} {fro 1} {as 1} {he 1} {lay 1} {back 1} {in 1} {his 1} {arm 1} {e 1} {them 1} {And 1} {she 1} {smiled 1} {her 1} {ecstatic 1} {smile 1} {I 1} {can 1} {t 1} {help 1} {it 1} {said 1} {the 1} {prince 1} {Lavater 1} {would 1} {have 1} {said 1} {I 1} {lack 1} {the 1} {bump 1} {of 1} {paternity 1} {Don 1} {t 1} {joke 1} {I 1} {mean 1} {to 1} {have 1} {a 1} {serious 1} {talk 1} {with 1} {you 1} {Do 1} {you 1} {know 1} {I 1} {am 1} {dissatisfied 1} {with 1} {your 1} {younger 1} {son 1} {Between 1} {ourselves 1} {and 1} {her 1} {face 1} {assumed 1} {its 1} {melancholy 1} {expression 1} {he 1} {was 1} {mentioned 1} {at 1} {Her 1} {Majesty 1} {s 1} {and 1} {you 1} {were 1} {pitied 1} {The 1} {prince 1} {answered 1} {nothing 1} {but 1} {she 1} {looked 1} {at 1} {him 1} {significantly 1} {awaiting 1} {a 1} {reply 1} {He 1} {frowned 1} {What 1} {would 1} {you 1} {have 1} {me 1} {do 1} {he 1} {said 1} {at 1} {last 1} {You 1} {know 1} {I 1} {did 1} {all 1} {a 1} {father 1} {could 1} {for 1} {their 1} {education 1} {and 1} {they 1} {have 1} {both 1} {turned 1} {out 1} {fools 1} {Hippolyte 1} {is 1} {at 1} {least 1} {a 1} {quiet 1} {fool 1} {but 1} {Anatole 1} {is 1} {an 1} {active 1} {one 1} {That 1} {is 1} {the 1} {only 1} {difference 1} {between 1} {them 1} {He 1} {said 1} {this 1} {smiling 1} {in 1} {a 1} {way 1} {more 1} {natural 1} {and 1} {animated 1} {than 1} {usual 1} {so 1} {that 1} {the 1} {wrinkles 1} {round 1} {his 1} {mouth 1} {very 1} {clearly 1} {revealed 1} {something 1} {unexpectedly 1} {coarse 1} {and 1} {unpleasant 1} {And 1} {why 1} {are 1} {children 1} {born 1} {to 1} {such 1} {men 1} {as 1} {you 1} {If 1} {you 1} {were 1} {not 1} {a 1} {father 1} {there 1} {would 1} {be 1} {nothing 1} {I 1} {could 1} {reproach 1} {you 1} {with 1} {said 1} {An 1}]
```

根据输出结果可以看出，当遇到“ ”字符时依然放入结果中了，因此对代码进行优化：

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"unicode"
)

type KeyValue struct {
	key   string
	value string
}

const chunksize = 1 << (10) //二进制赋值

func mapF(contents string) []KeyValue {
	//debug("Map %v\n", value)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	keys := strings.FieldsFunc(contents, f)
	var res []KeyValue
	for _, key := range keys {
		res = append(res, KeyValue{key, "1"})
	}
	return res
}

func main() {
	var wg sync.WaitGroup
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
    ans := make([][]KeyValue, file_num)
	b1 := bufio.NewReader(file)
	for i := 0; i < int(file_num); i++ {
		p := make([]byte, chunksize)
		b1.Read(p)
		wg.Add(1)
		go func(b []byte,a int) {
			defer wg.Done()
			res1 := mapF(string(b))
			ans[a] = append(ans[a], res1...)
		}(p, i)
	}
	wg.Wait()
	fmt.Println(ans)
}
```



我的思路是将每一块生成好的值存储在本地磁盘上，后续计数功能也放在磁盘上进行，为了方便起见，对每个文件命名也设定一下，

定义一个生成文件名称的函数reducename代码如下：

```GO
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type KeyValue struct {
	key   string
	value string
}

const chunksize = 1 << (10) //二进制赋值

//生成文件名称
func reduceName(mapTask int) string {
	return "mrtmp." + "-" + strconv.Itoa(mapTask) + ".json"
}

func mapF(contents string) []KeyValue {
	//debug("Map %v\n", value)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	keys := strings.FieldsFunc(contents, f)
	var res []KeyValue
	for _, key := range keys {
		res = append(res, KeyValue{key, "1"})
	}
	return res

}

func main() {
	var wg sync.WaitGroup
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
	ans := make([][]KeyValue, int(file_num))
	b1 := bufio.NewReader(file)
	for i := 0; i < int(file_num); i++ {
		p := make([]byte, chunksize)
		b1.Read(p)
		wg.Add(1)
		go func(b []byte, a int) {
			defer wg.Done()
			res1 := mapF(string(b))
			for _, keyValuePair := range res1 {
				ans[a] = append(ans[a], keyValuePair)
			}
		}(p, i)
	}
	wg.Wait()

	for index, task := range ans {
		file_name := reduceName(index)
		files, err := os.Create(file_name)
		if err != nil {
			panic(err)
		}
		taskJson, err := json.Marshal(task)
		if err != nil {
			panic(err)
		}
		if _, err := files.Write(taskJson); err != nil {
			panic(err)
		}
		if err := files.Close(); err != nil {
			panic(err)
		}
	}

}

```

我感觉逻辑上没有问题，但是生成的json文件全部为空值，暂时还没有想到解决方法。。。 这块先放置在一遍

**json写入后没有结果问题解决**

首先上述代码在逻辑上没有问题，但是在后续调试时候发现问题：

```go 
for index, task := range ans {
		file_name := reduceName(index)
		files, err := os.Create(file_name)
		if err != nil {
			panic(err)
		}
		taskJson, err := json.Marshal(task) // 出问题的地方
		if err != nil {
			panic(err)
		}
		if _, err := files.Write(taskJson); err != nil {
			panic(err)
		}
		if err := files.Close(); err != nil {
			panic(err)
		}
	}
```

代码中标记出问题的地方所在，在进行`json.Marshal`序列化的时候即转化成二进制时，结果显示可以转化成byte流，但是当进行string转化打印时候发现问题，打印出来的都是空元素，如下所示：

```go
[91 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 44 123 125 93]
[{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{},{}]
```

我们发现，其实在序列化成比特流时候就已经出错，原因在于没有正确的转化成byte流，针对这一问题，我暂时给出的解决方法就是采用第一种生成key-value的方式，并在每次读取文件转化成map时生成json文件，代码如下：

```go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

const chunksize = 1 << (10) //二进制赋值
func reduceName(mapTask int) string {
	return "mrtmp." + "-" + strconv.Itoa(mapTask) + ".json"
}

func main() {
	filename := "file.txt"
	ans := make(map[string]int)
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

		ss := strings.Fields(string(p[:n1]))

		for _, v := range ss {

			word := strings.ToLower(v)
			for len(word) > 0 && (word[0] < 'a' || word[0] > 'z') {
				word = word[1:]
			}
			for len(word) > 0 && (word[len(word)-1] < 'a' || word[len(word)-1] > 'z') {
				word = word[:len(word)-1]
			}
			ans[word]++
		}
		file_name := reduceName(i)
		files, err := os.Create(file_name)
		if err != nil {
			panic(err)
		}
		taskJson, err := json.Marshal(ans)
		if err != nil {
			panic(err)
		}
		if _, err := files.Write(taskJson); err != nil {
			panic(err)
		}
		if err := files.Close(); err != nil {
			panic(err)
		}

	}
}

```

输出结果：

```go 
{"":3,"a":3,"all":2,"and":9,"anna":2,"antichrist":2,"are":2,"arrive":1,"as":2,"at":1,"being":1,"believe":1,"buonapartes":1,"but":2,"by":2,"call":1,"chapter":1,"cough":1,"days":1,"defend":1,"do":3,"don't":1,"down":1,"elite":1,"empress":1,"estates":1,"faithful":1,"family":1,"favorite":1,"fedorovna":1,"first":1,"for":1,"friend":1,"frightened":1,"from":1,"genoa":1,"golang":1,"greeted":1,"grippe":2,"had":2,"have":2,"he":1,"hello":2,"her":2,"high":1,"honor":1,"horrors":1,"how":1,"i":6,"if":2,"importance":1,"in":2,"infamies":1,"invitations":1,"is":1,"it":1,"july":1,"just":1,"kuragin":1,"la":1,"longer":2,"lucca":1,"maid":1,"man":1,"marya":1,"me":2,"means":1,"more":1,"my":2,"new":1,"news":1,"no":2,"nothing":1,"now":1,"of":4,"only":1,"pavlovna":2,"peace":2,"perpetrated":1,"petersburg":1,"prince":2,"rank":1,"really":1,"reception":1,"said":1,"scherer":1,"see":1,"she":3,"sit":1,"slave":1,"so":1,"some":1,"speaker":1,"st":1,"still":1,"suffering":1,"tell":2,"that":2,"the":8,"then":1,"these":1,"this":1,"to":3,"try":1,"used":1,"vasili":1,"volume":1,"war":3,"warn":1,"was":4,"well":1,"well-known":1,"who":1,"will":1,"with":2,"witho"...}
```

这样就可以正常保存文件了， 至于第二种方法怎么在不改动的情况下解决生成json文件问题，还没有具体的方法，近期会给出结果。

## **三、reduce过程**

reduce过程就是将读取到的json文件里面重新计数并排序好，最后生成一个结果文件即可。

下面先给出读取json文件的代码：

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	//第一步读取文件
	inputFileName := "D://go example//go 项目//MapReduce//map_function//mrtmp.-0.json"
	files, err := os.Open(inputFileName)
	if err != nil {

		log.Fatal(err)
	}

	dec := json.NewDecoder(files)
	result := make(map[string]int)
	dec.Decode(&result)
	fmt.Println(result)

}
```

输出：

```go
map[:3 a:3 all:2 and:9 anna:2 antichrist:2 are:2 arrive:1 as:2 at:1 being:1 believe:1 buonapartes:1 but:2 by:2 call:1 chapter:1 cough:1 days:1 defend:1 do:3 don't:1 down:1 elite:1 empress:1 estates:1 faithful:1 family:1 favorite:1 fedorovna:1 first:1 for:1 friend:1 frightened:1 from:1 genoa:1 golang:1 greeted:1 grippe:2 had:2 have:2 he:1 hello:2 her:2 high:1 honor:1 horrors:1 how:1 i:6 if:2 importance:1 in:2 infamies:1 invitations:1 is:1 it:1 july:1 just:1 kuragin:1 la:1 longer:2 lucca:1 maid:1 man:1 marya:1 me:2 means:1 more:1 my:2 new:1 news:1 no:2 nothing:1 now:1 of:4 only:1 pavlovna:2 peace:2 perpetrated:1 petersburg:1 prince:2 rank:1 really:1 reception:1 said:1 scherer:1 see:1 she:3 sit:1 slave:1 so:1 some:1 speaker:1 st:1 still:1 suffering:1 tell:2 that:2 the:8 then:1 these:1 this:1 to:3 try:1 used:1 vasili:1 volume:1 war:3 warn:1 was:4 well:1 well-known:1 who:1 will:1 with:2 witho:1 word:1 words:1 world:1 you:8 yourself:1]
```

根据上述结果我们可以顺利的将json文件数据读出，

下面要做的工作就是将json文件读取，进行计数排序，下面给出计数排序的代码：

```go
package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"strconv"
)

func main() {
	//第一步读取文件
	inputFileName := "D://go example//go 项目//MapReduce//map_function//mrtmp.-0.json"
	files, err := os.Open(inputFileName)
	if err != nil {
		log.Fatal(err)
	}

	dec := json.NewDecoder(files)
	result := make(map[string]int)
	dec.Decode(&result)
	// 第二步 进行计数
	ans := make(map[string]int)
	for k, v := range result {
		ans[k] += v
	}
	// 第三部进行排序
	sortmap := []string{}
	for k := range ans {
		sortmap = append(sortmap, k)
	}
	sort.Strings(sortmap)

	//fmt.Println(sortmap)
	//保存结果
	final_result, err := os.Create("result.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer final_result.Close()

	for _, v := range sortmap {
		final_result.WriteString(v + ":" + strconv.Itoa(ans[v]) + "\n")
	}

}
```

上述代码给出了完整的从读取文件到计数排序最后生成结果文件。 我们可以看下最后结果

```txt
:3
a:3
all:2
and:9
anna:2
antichrist:2
are:2
arrive:1
as:2
at:1
being:1
believe:1
buonapartes:1
but:2
by:2
...
```

接着实现读取所有文件并保存结果，整体思路是一致，剩余的就是解决一个读取文件的问题。

代码如下：

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
)

func reduceName(mapTask int) string {
	return "mrtmp." + "-" + strconv.Itoa(mapTask) + ".json"
}
func main() {
	//第一步读取文件
	inputFileName := make([]*os.File, 12) // 这里后续都要改成 具体的任务数
	for i := 0; i < 12; i++ {
		file_name := reduceName(i)
		fmt.Println(file_name)
		inputFileName[i], _ = os.Open("D://go example//go 项目//MapReduce//map_function//" + file_name)
	}
    //第二步保存文件
	ans := make(map[string]int)
	for _, files := range inputFileName {
		result := make(map[string]int)
		dec := json.NewDecoder(files)
		dec.Decode(&result)
		for k, v := range result {
			ans[k] += v
		}
	}
	// 第三部进行排序
	sortmap := []string{}
	for k := range ans {
		sortmap = append(sortmap, k)
	}
	sort.Strings(sortmap)
	//保存结果
	final_result, err := os.Create("result.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer final_result.Close()

	for _, v := range sortmap {
		final_result.WriteString(v + ":" + strconv.Itoa(ans[v]) + "\n")
	}
}

```

最后结果保存在`result.txt`文件

到这为止一个简单的单机MapReuce的几个主要功能基本都实现了，下面将借助`goroutinue`、`资源池`、`一致性hash`

等手段进一步提高单机MapReuce性能

## 完整的简单单机MapReduce实现

代码如下：

```go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func doMap(
	chunksize int, //缓存大小
	filename string, // 文件名称
	nReduceTask int, //reduce任务数，块数
	ans map[string]int, // 保存中间存放k-v
	wg *sync.WaitGroup,

) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	b1 := bufio.NewReader(file)
	for i := 0; i < nReduceTask; i++ {
		p := make([]byte, chunksize)
		n1, err := b1.Read(p)
		if err != nil {
			log.Fatal(err)
		}
		ss := strings.Fields(string(p[:n1]))
		for _, v := range ss {
			word := strings.ToLower(v)
			for len(word) > 0 && (word[0] < 'a' || word[0] > 'z') {
				word = word[1:]
			}
			for len(word) > 0 && (word[len(word)-1] < 'a' || word[len(word)-1] > 'z') {
				word = word[:len(word)-1]
			}
			ans[word]++
		}
		file_name := ReduceName(i)
		files, err := os.Create(file_name)
		if err != nil {
			panic(err)
		}
		taskJson, err := json.Marshal(ans)
		if err != nil {
			panic(err)
		}
		if _, err := files.Write(taskJson); err != nil {
			panic(err)
		}
		if err := files.Close(); err != nil {
			panic(err)
		}

	}
	defer wg.Done()

}
func ReduceName(mapTask int) string {
	return "mrtmp." + "-" + strconv.Itoa(mapTask) + ".json"
}

func doReduce(
	filename string,
	result chan map[string]int,
	wg *sync.WaitGroup,
) {
	files, _ := os.Open(filename)
	ans := make(map[string]int)
	ans1 := make(map[string]int)
	dec := json.NewDecoder(files)
	dec.Decode(&ans)
	for k, v := range ans {
		ans1[k] += v
	}
	result <- ans1
	defer wg.Done()
}

const chunksize = 1 << (10) //二进制赋值

func main() {
	var wg sync.WaitGroup
	var wg1 sync.WaitGroup
	filename := "D:\\go example\\go 项目\\MapReduce\\map_function\\file.txt"
	ans := make(map[string]int)
	fi, err := os.Stat(filename) //使用fi.size得到文件大小
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(float64(fi.Size())) 查看大小
	file_num := math.Ceil(float64(fi.Size()) / float64(chunksize))
	nReduceTask := file_num
	wg.Add(1)
	go doMap(chunksize, filename, int(nReduceTask), ans, &wg)

	result := make([]chan map[string]int, int(nReduceTask))
	for i := 0; i < int(nReduceTask); i++ {
		result[i] = make(chan map[string]int, 10000)
	}
	for i := 0; i < int(nReduceTask); i++ {
		wg1.Add(1)
		file_name := ReduceName(i)
		file_name = "D://go example//go 项目//MapReduce//map_function//" + file_name
		go doReduce(file_name, result[i], &wg1)
	}
	wg.Wait()
	wg1.Wait()
	result_all := make(map[string]int)
	for _, v := range result {
		for k, value := range <-v {
			result_all[k] += value
		}
		if len(v) == 0 {
			close(v)
		}
	}
	sortmap := []string{}
	for k := range result_all {
		sortmap = append(sortmap, k)
	}
	sort.Strings(sortmap)
	//保存结果
	final_result, err := os.Create("result.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer final_result.Close()

	for _, v := range sortmap {
		final_result.WriteString(v + ":" + strconv.Itoa(result_all[v]) + "\n")
	}
}

```

这里面还是相对简单只加入了goroutinue还没有引入shuffle过程，在后面的多机练习中会陆续加入。

结果全部放入github上

链接：[renzhuangzhuang (github.com)](https://github.com/renzhuangzhuang)





