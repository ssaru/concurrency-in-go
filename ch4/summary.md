# 2020-07-22, Concurrency in Go, 127~132p

## Go의 동시성 패턴

-   이전까지 Go의 동시성 기본 요소를 탐구함
    1. `Sync.Cond`
    2. `Channel`
-   4장에서는 **시스템을 확장 가능하고 유지보수가 용이하도록 유지하는 방법**에 대해서 다룸
-   빈 인터페이스(`interface{}`)에 대하여
    -   논란의 여지가 있으나, 사용한 이유가 있음
    -   간결한 예제들을 더 쉽게 작성할 수 있다
    -   패턴을 통해 얻고자 하는 것들을 더 잘 표현한다 (자세한 내용은 **파이프라인**에서 설명)
    -   필요시, Go 생성자(**generator**)를 통해 필요로 하는 타입을 생성하고 활용하는 패턴을 사용할 수 있음

### 제한

-   동시성 코드를 작성할 때, **안전한 작동을 위한 몇가지 옵션**이 있음
    1. 메모리 공유를 위한 동기화 기본 요소(`sync.Mutext`)
    2. 통신을 위한 동기화(`channel`)
    3. 변경 불가능한 데이터(`immutable`)
    4. 제한에 의해 보호되는 데이터(`confinement`)

#### 변경 불가능한 데이터 (`immutable`)

-   암시적으로 동시에 실행해도 안전함
    1. 여러 프로세스는 동일한 데이터에 대해서 동작할 수 있으나, 수정할 수는 없음
    2. 새로운 데이터를 생성하기 위해서는 수정할 수 있는 새로운 복사본을 만들어야함
    3. `immutable`성격의 데이터는 임계 영역의 크기를 줄여 프로그램을 더 빠르게 만들어주기도 함
        - Q. 임계 영역의 크기를 줄이는 것과 프로그램을 더 빠르게 만들어주게한다는 의미는?
        - A. 임계 영역이 작으면 작을 수록 동기화에 리소스가 줄어들기 때문에

#### 제한 (`confinement`)

-   제한은 개발자의 인지 부하를 줄여줌
-   임계 영역의 크기를 줄여줌
-   동시성 값을 제한하는 기법은 단순히 값의 복사본을 전달하는 것보다는 조금 복잡함
-   **제한**의 철학은 **하나의 프로세스에서만 정보를 사용할 수 있도록 하는 것**

    -   **제한**은 암묵적으로 안전하며, 동기화도 필요 없음
    -   **제한**은 1). 애드 훅(`ad hoc`) 방식, 2). 어휘적(`lexical`) 방식으로 나누어짐
        1. 애드 훅 (`ad hoc`) 방식
            - 코드 작성자들끼리 정하는 약속
            - 많은 작성자가 코드를 건드리거나, 마감 시간이 다가오면 실수가 일어나거나 쉽게 깨질 수 있음
        2. 어휘적 (`lexical`) 방식
            - 컴파일러가 제한을 시행하도록 함
            - 잘못된 작업이 일어날 수 없음
    -   이러한 이유로 어휘적(`lexical`) 방식을 선호함

-   애드 훅 (`ad hoc`) 방식

    ```golang
    package main

    import "fmt"

    func main(){
        data := make([]int, 4)

        loopData := func(handleData chan <- int){
            defer close(handleData)

            for i:= range data{
                handleData <- data[i]
            }
        }

        handleData := make(chan int)
        go loopData(handleData)

        for num := range handleData{
            fmt.Println(num)
        }
    }
    ```

    -   `handleData`는 관례적으로 `loopData`함수에서만 접근
    -   실수할 여지가 있어, 제한이 깨질 수 있음
        -   Q. 실수할 여지가 있다는 것은 어떤 의미일까요?...
        -   A. `main` 고루틴에서 채널을 직접 인스턴스화하기 때문에 직접 쓰기 연산을 수행할 수 있음
    -   정적분석 도구를 이용해서 문제를 파악할 수 있으나, 정적분석은 팀의 성숙도 수준을 요구함

-   어휘적(`lexical`) 방식

    ```golang
    package main

    import "fmt"

    func main(){
        chanOwner := func() <- chan int {
            results := make(chan int, 5)

            go func(){
                defer close(results)

                for i:=0; i<=5; i++{
                    results <- i
                }
            }()
            return results
        }

        consumer := func(results <- chan int){
            for result := range results{
                fmt.Printf("Received: %d\n", results)
            }
            fmt.Println("Done receiving!")
        }

        results := chanOwner()
        consumer(results)
    }
    ```

    -   채널 `results`의 쓰기 권한이 클로져로 감싸져있어, 외부로 보일 일이 없음
    -   즉, 다른 고루틴이 채널 `results`에 쓰기 연산 수행을 하지못하도록 방지하는 역활을 함
    -   읽기 연산 또한 `consumer`로 감싸며, `consumer`가 읽기 연산 외에 다른 작업을 수행하지 못하도록 제한함
        -   Q. 이 부분은 코드상에서 그렇게 표현해서 그렇지, 다른 연산을 수행하도록 코딩할 수 있지 않나 싶습니다.

    ```golang
    package main

    import "fmt"

    func main(){
        printData := func(wg * sync.WaitGroup, data [] byte){
            defer wg.Done()

            var buff bytes.Buffer
            for _, b := range data{
                fmt.Fprintf(&buff, "%c", b)
            }
            fmt.Println(buff.String())
        }

        var wg sync.WaitGroup
        wg.Add(2)
        data := []byte("golang")
        go printData(&wg, data[:3])
        go printData(&wg, data[3:])
        wg.Wait()
    }
    ```

    -   `printData`는 `data` 슬라이스와 같은 클로져 내에 있지 않아 `data` 슬라이스에 접근할 수 없음.
    -   `printData`가 작업을 수행하기 위해서는 byte의 슬라이스를 인자로 받아야만 함(golang은 `call by value`)
    -   메모리를 **제한**하므로 메모리 접근을 동기화하거나 통신을 통해 데이터를 공유할 필요가 없음

-   위의 예제와 같이 번거롭게 작업하는 이유는 1). 성능을 향상시키고, 2). 개발자의 인지 부하를 줄이기 위함
-   동기화에는 비용이 들며, 동기화를 피할 수 있으면 임계 영역이 없으므로 동기화 비용을 지불할 필요가 없음
-   어휘적 제한을 사용하는 동시성 코드는 어휘적으로 제한된 변수를 사용하지 않는 동시성 코드보다 더 이해하기 쉬움

# 2020-07-23, Concurrency in Go, 132~147p

## Go의 동시성 패턴

-   **제한**을 설정하는게 어려운 상황들이 있으며, 이런 경우에는 Go의 동시성 기본 요소를 사용행야한다.
-   Go의 동시성 기본 요소는 아래와 같다
    1. for-select 루프
    2. 고루틴 누수 방지
    3. or채널
    4. 에러처리
    5. 파이프라인
    6. 팬 아웃, 팬인
    7. or-done 채널
    8. tee 채널
    9. bridge 채널
    10. 대기열 사용
    11. context 패키지

### 1. for-select 루프

-   for-select 루프는 Go 프로그램에서 반복적으로 나타남
-   for-select 루프는 아래와 같음
    ```golang
    for {   // 무한 반복 혹은 특정 범위에 대한 루프
        select{
            // 채널에 대하 작업
        }
    }
    ```
-   for-select 루프 패턴이 나타날 수 있는 시나리오는 아래와 같음
    1. 채널에서 반복 변수 보내기
    2. 멈추기를 기다리면서 무한 대기

#### 1. 채널에서 반복 변수 보내기

-   순회할 수 있는 것을 채널을 값으로 변환하려고하는 경우 사용함
    ```golang
    for _, s := range [] string { "a", "b", "c"}{
        select {
            case <- done:
                return
            case stringStream <- s:
        }
    }
    ```

#### 2. 멈추기를 기다리면서 무한히 대기

-   여러 방식으로 코딩할 수 있으며, 선호도에 따라 다름

-   `select` 구문을 가능한 짧게 유지하는 스타일

    ```golang
    for {
        select {
            case <-done:
                return
            default:
        }

        // 선점 불가능한 작업 수행
    }
    ```

    -   Q. 왜 비선점인가/
    -   A. 외부에서 작업을 가로채서 리소스를 가져갈 수 없기 때문에?

-   `select`구문을 길게하고, `default` 키워드를 이용하는 경우

    ```golang
    for {
        select {
            case <-done:
                return
            default;
                // 선점 불가능한 작업 수행
        }
    }
    ```

-   Q. for-select 루프는 어떤 의미에서 동시성 패턴일까?...

### 2. 고루틴 누수 방지

-   고루틴은 자원을 필요로하며, 런타임에 의해 가비지 컬렉션되지 않음
-   프로그래밍을 잘못하면, 고루틴 종료를 보장하지 못할 수 있으며, 이는 리소스 누수로 이어진다
-   고루틴이 종료되는 조건은 아래와 같음
    1. 작업이 완료되었을 때
    2. 복구할 수 없는 에러로 인해, 더 이상 작업이 불가능할 경우
    3. 작업을 중단하라는 요청을 받았을 때
-   (1), (2)는 사용자의 알고리즘 측면이라, 종료가 보장됨
-   (3)의 **작업 취소**는 사용자가 직접 제어해야하며, 네트워크 효과(종료 신호를 broadcast)가 있으므로 가장 중요함
-   여러 고루틴을 사용한다면, 어떻게든 일련의 방식으로 고루틴끼리 협력할 가능성이 높음
-   협력으로 인해, 상대 고루틴의 상태에 따라 종속적인 고루틴이 강제 종료되어야할 수도 있음
-   아래 예제의 경우, 고루틴 종료를 보장하지 못함

    ```golang
    doWork := func(strings <- chan string) <- chan interface{} {
        completed := make(chan interface{})
        go func(){
            defer fmt.Println("doWork exited.")
            defer close(completed)

            for s:= range strings{
                // 원하는 작업을 수행
                fmt.Println(s)
            }
        }()
        return completed
    }

    doWork(nil)
    // 추가적인 작업
    fmt.Println("Done")
    ```

    1. `nil`채널이 고루틴 `doWork`로 전달됨
    2. `strings`채널은 실제로 어떠한 문자열도 쓰지 않음
    3. 이로인해 `doWork`를 포함하는 고루틴은 프로세스를 지속하는 동안 메모리에 남아있음
    4. 심지어 join시, 데드락 발생

-   위의 예제는 간단하지만, 실제 프로그램에서 최악의 경우 main 고루틴이 평생 동안 고루틴을 계속 돌려, 메모리 사용량에 영향을 줄 수 있음

-   그렇다면, 고루틴 종료를 어떻게 보장할 수 있는가?

    -   부모 고루틴이 자식 고루틴에게 취소(`cancellation`)신호를 보냄
    -   부모 고루틴은 `done`이라는 읽기 전용 채널을 통해서 취소 신호를 전달함
    -   부모 고루틴은 자식 고루틴에게 `done`채널을 전달하고, `done`채널을 닫는 것으로 취소신호를 전달함

    -   (읽기)`done`채널을 활용하여 고루틴을 종료하는 예제는 아래와 같음

        ```golang
        doWork := func(done <- chan interface{},
                       strings <- chan string) <- chan interface{}{
                           terminated := make(chan interface{})

                           go func(){
                               defer fmt.Println("doWork exited.")
                               defer close(terminated)

                               for {
                                   select{
                                        case s:= <- strings:
                                            // 원하는 작업 수행
                                        case <-done:
                                            return
                                   }
                               }
                           }
                       }()
        done := make(chan interface{})
        terminated := doWork(done, nil)

        go func(){
            time.Sleep(1 * time.Second)
            fmt.Println("Canceling doWork goroutine...")
            close(done)
        }()

        <- terminated
        fmt.Println("Done.")
        ```

        -   strings챈널에 `nil`채널을 전달했음에도, 고루틴은 성공적으로 종료됨
        -   두 개의 고루틴을 조인하기 전에, 세 번째 고루틴을 생성해 1초 후에 `doWork`내에서 고루틴을 종료시키기에 이전처럼 데드락이 발생하지 않음

    -   (쓰기) 채널에 값을 스려는 시도를 차단하는 고루틴 예제는 아래와 같음

        ```golang
        newRandstream := func() <- chan int{
            randStream := make(chan int)
            go func(){
                defer fmt.Println("newRandStream closure exited.")
                defer close(randStream)

                for {
                    randStream <- rand.Int()
                }
            }()
            return randStream
        }

        randStream := newRandStream()
        fmt.Println("3 random ints:")
        for i:=1; i <=3; i++{
            fmt.Printf("%d: %d\n", i, <-randStream)
        }
        ```

        ```bash
        >>> 3 random ints:
        1: xxxx
        2: xxxx
        3: xxxx
        ```

        -   `fmt.Println("newRandStream closure exited.")`는 출력되지 않음
        -   `for i:=1; i <=3; i++`에서 3번을 순회하고나면 main고루틴은 종료되며, `newRandstream`는 계속해서 `randStream`채널에 랜덤정수를 씀
        -   예제에서는 생산자에게 멈춰도 된다고 전달할 수 없음

    -   이를 보완하기 위해서 생산자에게 종료를 알리는 채널을 제공해야함

        ```golang
        newRandStream := func(done <-chan interface{}) <-chan int{
            randStream := make(chan int)
            go func(){
                defer fmt.Println("newRandStream closure exited.")
                defer close(randStream)
                for {
                    select{
                        case randStream <- rand.Int():
                        case <- done:
                            return
                    }
                }
            }()
            return randStream
        }

        done := make(chan interface{})
        randStream := newRandStream(done)
        fmt.Println("3 random ints:")
        for i:=1; i <=3; i++{
            fmt.Printf("%d: %d\n", i, <-randStream)
        }
        close(done)

        time.Sleep(1 * time.Second)
        ```

        ```bash
        3 random ints:
        1: xxxx
        2: xxxx
        3: xxxx
        newRandStream closure exited.
        ```

    -   위의 예제들을 통해 다른 고루틴을 생성한 책임이 있는 고루틴은 해당 고루틴을 중지시킬 책임도 짐
    -   고루틴의 중지를 보장하는 방법은 고루틴의 타입과 목적에 따라 다를 수 있지만, 모두 done 채널을 전달하는 것을 바탕으로 구축됨

### 3. or 채널

-   여래 개의 `done` 채널을 하나의 `done`채널로 결합해, 그 중 하나의 채널이 닫힐 때, 결합된 채널이 모두 닫아야하는 경우가 있음
-   하나의 채널이 닫힐 때, 결합된 채널을 닫는 방식으로 `select`를 쓸 수 있지만, 때로는 런타임이 작업 중인 `done`채널의 개수를 알 수 없을 수 있음

-   작업 중인 `done`채널가 가변적일 때, 결합된 채널을 모두 닫기위해 `or 채널 패턴`을 사용함
-   `or 채널`은 재귀 및 고루틴을 통해 복합 `done`채널을 만듬
-   `or 채널`의 예제는 아래와 같음

    ```golang
    var or func(channels ... <-chan interface{}) <- chan interface{}
    or = func(channels... <-chan interface{}) <-chan interface{}{
        switch len(channels){
            case 0:
                return nil
            case 1:
                return channels[0]
        }

        orDone := make(chan interface{})
        go func(){
            defer close(orDone)

            switch len(channels){
                case 2:
                    select {
                        case <- channels[0]:
                        case <- channels[1]:
                    }
                    default:
                        select {
                            case <- channels[0]:
                            case <- channels[1]:
                            case <- channels[2]:
                            case <- or(append(channels[3:], orDone)...):
                        }
            }
        }()
        return orDone
    }
    ```

    -   가변개수의 채널이 3번째 인덱스 뒤로 슬라이싱되어 재귀적으로 도는 방식
        -   연쇄적으로 `orDone`채널을 공유하기 때문에 `orDone`채널이 닫히면, 결합된 채널 또한 연쇄적으로 닫힘
    -   고루틴의 수를 제한하는 최적화를 위해, 두 개 채널에 대한 호출 똔는 두개의 채널을 가지고 있는 특별한 `case`문을 추가함

    -   `or채널`을 사용하는 간단한 예시는 아래와 같음
    -   `after`에 명시된 시간이 지나면 닫히는 채널을 통해, 결합된 채널을 모두 닫는 예제

        ```golang
        sig := func(after time.Duration) <- chan interface{}{

            c := make(chan interface{})
            go func(){
                defer close(c)
                time.Sleep(after)
            }()
            return c
        }

        start := time.Now()
        <- or(
            sig(2 * time.Houre),
            sig(5 * time.Minute),
            sig(1 * time.Second),
            sig(1 * ttime.Hour),
            sig(1 * time.Minute),
        )

        fmt.Printf("done after %v", ttime.SSince(start))
        ```

    -   x가 고루틴의 수라고 가정했을 때, 추가적인 고루틴들의 비용인 `f(x) = floor(x/2)`으로 간결함을 얻을 수 있다(???)
    -   추후 `context` 패키지에서 이와 같은 작업을 하는 다른 예제도 살펴볼 예정
    -   추후 "복제된 요청" 예제에서 이 패턴을 응용하여 복잡한 패턴을 형성한는 방법을 살펴볼 예정

### 4. 에러 처리

-   Go는 널리 알려진 에러의 예외 모델을 피하면서 에러 처리가 중요하다고 이야기했으며, 알고리즘에 주의를 기울이는 것과 동일한 수준으로 에러 경로(`error path`)에 신경써야한다고 이야기함
-   에러 처리에 대해서 제일 근본적은 질문은 **"에러 처리의 책임자는 누구인가?"**
    1. 어떤 시점에서 스택을 따라 에러를 전달하는 것을 멈춰야하는가
-   동시에 실행되는 프로세스인 경우, 이 질문은 매우 복잡해짐

```golang
checkStatus := func(done <- chan interface{},
                    urls ...string) <- chan *http.Response{
                        responses := make(chan *http.Response)

                        go func(){
                            defer close(responses)

                            for _, url := range urls{
                                resp, err := http.Get(url)
                                if err != nil{
                                    fmt.Println(err)
                                    continue
                                }
                                select {
                                    case <- done:
                                        return
                                    case responses <- resp:
                                }
                            }
                        }()
                        return responses
                    }

                    done := make(chan interface{})
                    defer close(done)

                    urls := []string("https://www.google.com", "httpss;//badhost")
                    for response := range checckStatus(done, urls...){
                        fmt.Printf("Response: %v\n", response.Status)
                    }
```

```bash
>>> Response: 200 OK
Get https://badhost: dial tcp: lookup badhost on 127.0.1.1.53: no such host
```

-   에러를 모아서 어떤 선택을 할 수 없으며, 누군가가 봐주기를 기다릴 수 밖에 없음
-   관심사항을 분리해서, 프로그램의 상태에 대해 완전한 정보를 가지고있는 다른 부분으로 에러를 보내야함
-   에러를 모두 취합한 부분은 취합한 에러를 통해서 무엇을 할지 결정할 수 있음
-   위의 예제를 다시 구조화하면, 아래와 같음

```golang
type Result struct{
    Error error
    Response *http.Responsses
}

checkStatus := func(done <-chan interface{},
                    urls ...string) <-chan Result{
                        results := make(chan Result)

                        go func(){
                            defer close(results)

                            for _, url := range urls{
                                var result Result
                                resp, err := http.Get(url)
                                result = Result{Error: err, Response: resp}

                                select{|
                                    case <- done:
                                        return
                                    case results <- result:
                                }
                            }
                        }()
                        return results
                    }

                    done := make(chan interface{})
                    defer close(done)

                    urls := [] string {"https://www.google.com", "https://badhost"}
                    for result := range checkStatus(done, urls...){
                        if result.Error != nil{
                            fmt.Printf("error: %v, ressult.Error")
                            continue
                        }

                        fmt.Printf("Response: %v\n", result.Response.Statuss)
                    }
```

-   `*http.Response`와 고루틴 내 반복 시 발생할 수 있는 error를 포함하는 타입을 생성
-   이렇게 프로그래밍을 할 경우, 전체 컨텍스트 내에서 에러를 처리할 수 있음
-   이를 활용하면, 아래와 같이 3번 이상의 에러가 발생하면, 상태체크를 멈추게끔 할 수 있다.

```golang
done := make(chan interface{})
defer close(done)

errCount := 0
urls := []string("a", "https://www.google.com", "b", "c", "d")
for results := range checckStatus(done, urls...){
    if result.Error != nil{
        fmt.Println("error: %v\n", result.Error)
        errCount++

        if errCount >= 3{
            fmt.Println("Too many errors, breaking!!")
            break
        }
        continue
    }
    fmt.Printf("ressponse: %v\n", result.Responssse.Status)
}
```

# 2020-07-29, Concurrency in Go, 147~p159

## 파이프라인

-   파이프라인은 시스템에서 추상화를 구성하는데 사용할 수 있는 또 다른 도구
-   특히, 스트림이나 데이터에 대한 일괄 처리(batch) 작업들을 처리해야 할 때 사용하는 매우 강력한 도구
-   파이프라인은 데이터를 가져와서, 그 데이터를 대상으로 작업을 수행하고, 결과 데이터를 다시 전달하는 일련의 작업에 불과. 여기서 각각의 작업을 파이프라인상의 단계(stage)라고 부름
-   파이프라인을 사용하면, 각 단계의 관심사를 분리할 수 있어 많은 이점을 얻을 수 있음
    -   상호 독립적으로 각 단계를 수정할 수 있음
    -   단계들의 결합 방식을 짜맞출 수 있음
    -   일부분을 팬 아웃하거나 속도를 제한할 수 있음

```golang
package main

import "fmt"

func main() {
	multiply := func(values []int, multiplier int) []int {
		multipliedValues := make([]int, len(values))

		for i, v := range values {
			multipliedValues[i] = v * multiplier
		}

		return multipliedValues
	}

	add := func(values []int, additive int) []int {
		addedValues := make([]int, len(values))

		for i, v := range values {
			addedValues[i] = v + additive
		}

		return addedValues
	}

	ints := []int{1, 2, 3, 4}
	for _, v := range add(multiply(ints, 2), 1) {
		fmt.Println(v)
	}
}
```

-   위 예시 코드에서 `multiply`, `add`함수는 일상에서 접할만한 함수이지만, 파이프라인 단계의 속성을 갖도록 구성했기 때문에, 이를 결합해 파이프라인을 구성할 수 있음
-   파이프라인 단계의 특성이란 무엇인가?

    1. 각 단계는 동일한 타입을 인자로 받고, 리턴한다.
    2. 각 단계는 전달될 수 있도록 언어에 의해 구체화돼야한다.
        > 구체화란 함수 개념을 노출시켜주는 것을 의미하는데, Go는 함수 시그니쳐 타입을 갖는 변수를 정의할 수 있다.
        > `var or func(channels ... <-chan interface{}) <- chan interface{}`
        > 시그니쳐란 "함수와 인자들의 이름을 제외한 나머지"이다
        > 즉, 리턴값의 데이터타입, 인자의 개수, 각 인자의 데이터 타입과 순서를 의미한다.

-   파이프라인의 단계는 함수형 프로그래밍의 "모나드"의 부분 집합으로 간주할 수 있다

-   코드를 일부 변경해야할 경우, 파이프라인을 사용하게되면 "재사용"을 통해서 이를 해결할 수 있다.
-   위의 파이프라인 예제에서 2를 곱하는 새로운 스테이지를 추가한다고 했을 때, 아래와 같이 변경할 수 있다.

```golang
ints := []int{1,2,3,4}
for _, v := range multiply(add(multiply(ints, 2), 1), 2){
    fmt.Println(v)
}
```

-   위 예제를 절차적으로 표현하면 아래와 같다.
-   아래 예제와 같이 절차적으로 표현된 코드는 스트림을 처리할 때, 파이프라인과 같은 이점을 제공하지 못한다.

```golang
ints := []int{1,2,3,4}
for _, v := range ints{
    fmt.Println(2*(v*2+1))
}
```

-   지금까지 언급한 내용과 같은 내용이지만, 파이프라인의 단계들은 일괄처리를 수행한다.
-   일괄처리란 한 번에 하나씩 이산값을 처리하는 대신에 모든 데이터 덩어리를 한 번에 처리한다는 의미이다.

-   스트림 처리를 수행하는 타입의 파이프라인인 경우 각 단계가 한 번에 하나의 요소를 수신하고 방출하는 단계가 하나 더 있다
-   파이프라인 코드 예제를 스트림 지향으로 변환하면 아래와 같다(거의 유사하다. 단지 한 번에 하나의 요소만 수신하고 방출하게끔 `multiply`, `add`, `for-loop`이 변경되었다)

```golang
package main

import "fmt"

func main() {
	multiply := func(value, multiplier int) int {
		return value * multiplier
	}

	add := func(value, additive int) int {
		return value + additive
	}

	ints := []int{1, 2, 3, 4}

	for _, v := range ints {
		fmt.Println(multiply(add(multiply(v, 2), 1), 2))
	}
}
```

-   위의 예제는 파이프라인을 for-loop에 배치하고 range가 파이프라인에 데이터를 공급하는 무거운(?) 작업을 수행하도록 작성되었다.
-   이러한 무거운(?) 작업은 파이프라인에 몇가지 단점을 갖는다.

    1. 데이터를 공급하는 방식의 재사용을 제한한다(?)
    2. 확장성을 제한한다(?)
    3. loop가 반복될 때 마다 파이프라인을 인스턴스화한다.

-   이제 Go에서 파이프라인을 구성하기 위한 **최상의 방법**이 무엇이고, Go의 채널 기본 요소를 시작한다.

**NOTE**

-   Q1. `위의 예제는 파이프라인을 for-loop에 배치하고 range가 파이프라인에 데이터를 공급하는 무거운(?) 작업을 수행하도록 작성되었다`
    -   무거운 작업이라는게 어떤 의미인지 모르습니다.
-   Q2. `데이터를 공급하는 방식의 재사용을 제한한다(?)`
    -   A. 재사용이라고하면, 뒤에 나오는 generator를 활용하여 재사용성을 증가시킨다고 이해했습니다.
-   Q3. `확장성을 제한한다(?)`
    -   A. 책의 후미에서 보강해주리라 믿습니다...

### 파이프라인 구축의 모범 사례

-   채널은 Go의 모든 기본 요구 사항(?)을 충족한다

    1. 채널의 값을 받고 방출할 수 있다.
    2. 동시에 실행해도 안전하다.
    3. 언어에 의해 구체화된다.
    4. 여러가지를 아우른다(?)

-   앞의 예제를 채널을 활용하는 방향으로 변경해보자

```golang
package main

import "fmt"

func main() {
	generator := func(done <-chan interface{}, integers ...int) <-chan int {
		intStream := make(chan int, len(integers))

		go func() {
			defer close(intStream)

			for _, i := range integers {
				select {
				case <-done:
					return
				case intStream <- i:
				}
			}
		}()
		return intStream
	}

	multiply := func(
		done <-chan interface{},
		intStream <-chan int,
		multiplier int,
	) <-chan int {

		multipliedStream := make(chan int)
		go func() {
			defer close(multipliedStream)
			for i := range intStream {
				select {
				case <-done:
					return
				case multipliedStream <- i * multiplier:
				}
			}
		}()
		return multipliedStream
	}
	add := func(done <-chan interface{},
		intStream <-chan int,
		additive int,
	) <-chan int {
		addedStream := make(chan int)
		go func() {
			defer close(addedStream)
			for i := range intStream {
				select {
				case <-done:
					return
				case addedStream <- i + additive:
				}
			}
		}()
		return addedStream
	}

	done := make(chan interface{})
	defer close(done)

	intStream := generator(done, 1, 2, 3, 4)
	pipeline := multiply(done, add(done, multiply(done, intStream, 2), 1), 2)

	for v := range pipeline {
		fmt.Println(v)
	}

}
```

-   변경된 코드는 모두 채널을 사용한다는 점에서 이전 코드와 다르며 결정적인 차이는 아래와 같다

    1. 파이프라인의 끝에서 `range`구문을 사용해 값을 추출한다
    2. 파이프라인의 각 단계가 동시에 실행된다.
    3. 입력 및 출력이 동시에 실행되는 컨텍스트에서 안전하다
    4. (3)으로 인해 각 단계에서 안전하게 동시에 실행할 수 있다

-   코드의 실행순서를 테이블로 나타내면 아래와 같다

| 반복변수 | Generator | Multiply |   Add   | Multiply | 값  |
| :------: | :-------: | :------: | :-----: | :------: | :-: |
|    0     |     1     |          |         |          |     |
|    0     |           |    1     |         |          |     |
|    0     |     2     |          |    2    |          |     |
|    0     |           |    2     |         |    3     |     |
|    0     |     3     |          |    4    |          |  6  |
|    1     |           |    3     |         |    5     |     |
|    1     |     4     |          |    6    |          | 10  |
|    2     |  (close)  |    4     |         |    7     |     |
|    2     |           | (close)  |    8    |          | 14  |
|    3     |           |          | (close) |    9     |     |
|    3     |           |          |         | (close)  | 18  |

-   파이프라인이 실행되는 도중 `done`채널을 닫으면 닫힌 `done`채널이 파이프라인에 전파된다
-   닫힌 `done`채널의 전파로인해 파이프라인은 강제로 종료된다.

-   앞서, 파이프라인의 시작 부분에서 이산 값을 채널로 변경해야한다고 했다
-   이로인해, 위 예제의 프로세스에서는 반드시 선점 가능해야하는 두가지 지점이 있다(?)

    1. 즉각적으로 이루어지지 않는 이산 값의 생성(?)
    2. 해당 채널로의 이산 값 전송

-   첫 번째는 코드 작성자에게 달렸다(?)
-   두 번째는 `select`와 `done`채널을 통해 처리된다

    -   이로인해 `intStream`에 쓰기를 시도하는 것이 차단된 경우에도 generator가 선점 가능하도록 한다.

-   입력 채널에서 값을 가져오기 위해 어떤 단계가 대기 상태가 되면, 해당 채널이 닫힐 때, 그 단계가 대기 해제된다.
-   값을 전송하다가 단계가 대기 상태가 되는 경우, `select`문을 사용한다면 선점 가능하다
-   전체 파이프라인은 `done`채널을 닫음으로써 항상 선점 가능하다.

**NOTE**

-   Q1. `채널은 Go의 모든 기본 요구 사항(?)을 충족한다`
    -   A. 여기서 의미하는 **모든 기본 요구 사항**은 **파이프라인 단계의 특성**에 대한 요구사항을 충족한다고 이해했습니다.
-   Q2. `여러가지를 아우른다(?)`
    -   여러가지가 어떤 의미인지 모르겠네요...
-   Q3. `파이프라인의 시작 부분에서 이산 값을 채널로 변경해야하므로, 해당 프로세스에서는 반드시 선점 가능해야하는 두가지 지점이 있다(?)`
    -   이산값을 채널로 변경해야하는 것과 선점해야하는 지점은 어떤 관계가 있는지 모르겠습니다... 애초에 왜 선점해야하는지 모르겠습니다.
    -   (!!) 파이프라인의 시작부분이기 때문에, 이를 선점하고 들어가야 파이프라인을 실행시킬 수 있지 않을까 생각해봤습니다.
        선점한다는게 `제어가능/개입가능(Controllable)`을 의미하는게 아닌가 싶습니다. 파이프라인을 제어하기 위해서 선점 가능해야한다고 해석됩니다.
-   Q4. `즉각적으로 이루어지지 않는 이산 값의 생성(?)`
    -   (Q3)이 해소되면 자연스럽게 해결되리라 기대해봅니다...
-   Q5. `첫 번째는 코드 작성자에게 달렸다(?)`
    -   (Q3)이 해소되면 자연스럽게 해결되리라 기대해봅니다...

# 2020-07-29, Concurrency in Go, 159~p172

### 유용한 생성기들

-   파이프라인의 `generator`는 이산 값의 집합을 채널의 값 스트림으로 변환하는 함수
-   본 섹션에서는 `generator`의 다양한 형태를 알아본다.

    1. repeat
    2. take
    3. repeatFn

-   특정 타입을 처리하기 위해 `generator`에 타입 단정문을 수행하는 단계를 배치하고, 벤치마크한다.

#### repeat

```golang
package main

func main() {
	repeat := func(
		done <-chan interface{},
		values ...interface{},
	) <-chan interface{} {
		valueStream := make(chan interface{})

		go func() {
			defer close(valueStream)
			for {
				for _, v := range values {
					select {
					case <-done:
						return
					case valueStream <- v:
					}
				}
			}
		}()
		return valueStream
	}
}
```

-   위 예제는 사용자가 멈추라고 말할 때까지 사용자가 전달한 값을 무한 반복한다

#### take

```golang
package main

func main() {
	take := func(
		done <-chan interface{},
		valueStream <-chan interface{},
		num int,
	) <-chan interface{} {
		takeStream := make(chan interface{})
		go func() {
			defer close(takeStream)

			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}
}
```

-   `take` 단계는 입력 `valueStream`에서 첫 번째 항목을 취한 다음 종료

#### repeat과 take의 조합

```golang
package main

import "fmt"

func main() {
	repeat := func(
		done <-chan interface{},
		values ...interface{},
	) <-chan interface{} {
		valueStream := make(chan interface{})

		go func() {
			defer close(valueStream)
			for {
				for _, v := range values {
					select {
					case <-done:
						return
					case valueStream <- v:
					}
				}
			}
		}()
		return valueStream
	}

	take := func(
		done <-chan interface{},
		valueStream <-chan interface{},
		num int,
	) <-chan interface{} {
		takeStream := make(chan interface{})
		go func() {
			defer close(takeStream)

			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}

	done := make(chan interface{})
	defer close(done)

	for num := range take(done, repeat(done, 1), 10) {
		fmt.Printf("%v ", num)
	}
}
```

-   `repeat`은 1을 무한히 반복해서 생성하는 `generator`
-   `take`는 `repeat`에서 10개만을 가져오는 `generator`

#### repeatFn

-   위의 예제를 확장해서 반복적으로 함수를 호출하는 `generator`를 만들 수 있음

```golang
package main

import (
	"fmt"
	"math/rand"
)

func main() {
	repeatFn := func(
		done <-chan interface{},
		fn func() interface{},
	) <-chan interface{} {
		valueStream := make(chan interface{})

		go func() {
			defer close(valueStream)
			for {
				select {
				case <-done:
					return
				case valueStream <- fn():
				}
			}
		}()
		return valueStream
	}

	take := func(
		done <-chan interface{},
		valueStream <-chan interface{},
		num int,
	) <-chan interface{} {
		takeStream := make(chan interface{})
		go func() {
			defer close(takeStream)

			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}

	done := make(chan interface{})
	defer close(done)

	rand := func() interface{} { return rand.Int() }

	for num := range take(done, repeatFn(done, rand), 10) {
		fmt.Println(num)
	}
}

```

-   `repeat`과 `repeatFn` 생성기는 목록 또는 연산자를 반복해 데이터 스트림을 생성하는 것이 주된 관심사
-   `take`단계는 파이프라인을 제한하는 것이 주된 관심사
-   위 연산들(`repeat`, `repeatFn`, `take`)은 작업 중인 타입에 대한 정보가 필요 없음

#### 특정 타입을 처리하는 단계

-   특정 타입을 처리해야하는 경우, 타입 단정문(assertion)을 수행하는 단계를 배치할 수 있음

```golang
package main

import "fmt"

func main() {

	toString := func(
		done <-chan interface{},
		valueStream <-chan interface{},
	) <-chan string {
		stringStream := make(chan string)

		go func() {
			defer close(stringStream)
			for v := range valueStream {
				select {
				case <-done:
					return
				case stringStream <- v.(string):
				}
			}
		}()
		return stringStream
	}

	repeat := func(
		done <-chan interface{},
		values ...interface{},
	) <-chan interface{} {
		valueStream := make(chan interface{})

		go func() {
			defer close(valueStream)
			for {
				for _, v := range values {
					select {
					case <-done:
						return
					case valueStream <- v:
					}
				}
			}
		}()
		return valueStream
	}

	take := func(
		done <-chan interface{},
		valueStream <-chan interface{},
		num int,
	) <-chan interface{} {
		takeStream := make(chan interface{})
		go func() {
			defer close(takeStream)

			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}

	done := make(chan interface{})
	defer close(done)

	var message string
	for token := range toString(done, take(done, repeat(done, "I", "am."), 5)) {
		message += token
	}

	fmt.Printf("message: %s...", message)
}
```

**Benchmark**

-   파이프라인을 일반화하는 부분의 성능상 비용은 무시할 수 있음을 증명
-   성능상 비용을 무시할 수 있음을 증명하기 위해서 일반적인 단계를 테스트하는 함수와 타입에 특화된 단계를 테스트하는 함수를 작성

```golang
package pipelines

import "testing"

func BenchmarkGeneric(b *testing.B) {

	toString := func(
		done <-chan interface{},
		valueStream <-chan interface{},
	) <-chan string {
		stringStream := make(chan string)

		go func() {
			defer close(stringStream)
			for v := range valueStream {
				select {
				case <-done:
					return
				case stringStream <- v.(string):
				}
			}
		}()
		return stringStream
	}

	repeat := func(
		done <-chan interface{},
		values ...interface{},
	) <-chan interface{} {
		valueStream := make(chan interface{})

		go func() {
			defer close(valueStream)
			for {
				for _, v := range values {
					select {
					case <-done:
						return
					case valueStream <- v:
					}
				}
			}
		}()
		return valueStream
	}

	take := func(
		done <-chan interface{},
		valueStream <-chan interface{},
		num int,
	) <-chan interface{} {
		takeStream := make(chan interface{})
		go func() {
			defer close(takeStream)

			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}

	done := make(chan interface{})
	defer close(done)

	b.ResetTimer()
	for range toString(done, take(done, repeat(done, "a"), b.N)) {

	}
}

func BenchmarkTyped(b *testing.B) {
	repeat := func(done <-chan interface{}, values ...string) <-chan string {
		valueStream := make(chan string)

		go func() {
			defer close(valueStream)
			for {
				for _, v := range values {
					select {
					case <-done:
						return
					case valueStream <- v:
					}
				}
			}
		}()
		return valueStream
	}

	take := func(
		done <-chan interface{},
		valueStream <-chan string,
		num int,
	) <-chan string {
		takeStream := make(chan string)
		go func() {
			defer close(takeStream)
			for i := num; i > 0 || i == -1; {
				if i != -1 {
					i--
				}

				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}

	done := make(chan interface{})
	defer close(done)
	b.ResetTimer()

	for range take(done, repeat(done, "a"), b.N) {

	}
}
```

```bash
$ go test -bench=.
>>>
goos: darwin
goarch: amd64
BenchmarkGeneric-12      1000000              1478 ns/op
BenchmarkTyped-12        2000000               859 ns/op
PASS
ok      _/Users/martin/Documents/dev/Study/concurrency-go/ch4/09-pipeline-test  4.098s
```

-   특정 타입에 특화된 단계가 2배 정도 빠르지만, 크게 의미있는 정도는 아님
    -   일반적으로 파이프라인의 성능상 제한 요소는 `generator` 또는 계산 집약적인 단계 중 하나
    -   디스크나 네트워크에 읽어오는 경우 여기서 보이는 성능상의 부하는 큰 의미가 없음

## 팬 아웃, 팬 인

-   작업 단계 중 하나에 계산량이 많이 필요하다면, 이것 역시 성능상 부하를 퇴색시킬 것
-   계산적으로 비용이 많이 드는 단계를 어떻게 줄일 수 있을까?
-   이를 완화하는 방법이 `팬 아웃`, `팬 인`
-   파이프라인의 단계에서 특별하게 많은 계산을 필요할 수 있음

    -   이 경우 많은 계산을 필요로 하는 단계가 완료되기를 기다리는 동안 파이프라인상의 앞쪽 단계들이 대기 상태가 될 수 있음
    -   파이프라인은 기본적으로 개별 단계를 조합해 데이터 스트림에서 연산할 수 있음
    -   파이프라인의 단계들을 여러 번 재사용할 수 있음

-   `팬 아웃`, `팬 인`은 여러개의 고루틴을 통해 파이프라인의 상류 단계로부터 데이터를 가져오는 것을 병렬화하면서, 파이프라인상의 한 단계를 재사용하는 것

-   `팬 아웃`은 파이프라인의 입력을 처리하기 위해 여러개의 고루틴들을 시작하는 프로세스를 나타내는 용어
-   `팬 인`은 여러 결과를 하나의 채널로 결합하는 프로세스를 설명하는 용어

-   `팬 아웃`, `팬 인` 패턴을 활용하기 위해서는 아래 두 가지 사항이 모두 적용되는지를 확인해야함

    1. 단계가 이전에 계산한 값에 의존하지 않음
    2. 단계를 실행하는데 오랜 시간이 걸림

    -   순서 독립성(order-independence)는 중요한데, 이는 동시에 실행되는 해당 단계의 복사본이 어떤 순서로 실행되는지, 어떤 순서로 리턴되는지에 대한 보장이 없기 때문

# 중략... (보강필요)

# 2020-08-10, Concurrency in Go, 178~p182

### 대기열 사용

-   파이프라인이 준비되지 않았더라도, 파이프라인에 대한 작업을 받아들이는 것이 유용할 때가 있음

    -   이를 **대기열 사용**이라고 부름

-   **대기열 사용**이란

    1. 어떤 단계가 일부 작업을 완료하면 이를 메모리의 임시 위치에 저장해 다른 단계에서 나중에 조회할 수 있음
    2. 작업을 완료한 단계는 작업 결과에 대한 참조를 저장할 필요가 없음

-   대기열을 성급하게 도입하면 데드락이나 라이브락과 같은 동기화 문제가 드러나지 않을 수 있음
-   프로그램이 정확해짐에 따라 대기열이 얼마나 필요한지 알게 될 수도 있음

-   시스템 성능을 조정하려고 할 때, 사람들이 범하는 일반적인 실수는 **성능 문제를 해결하기 위해서 대기열의 도입을 고민 하는 것**
    -   대기열은 프로그램의 총 실행 속도를 거의 높여주지 않음

```golang
done := make(chan interface{})
defer close(done)
zeros := take(done, 3, repeat(done, 0))
short := sleep(done, 1*time.Second, zeros)
long := sleep(done, 4*time.Second, short)

pipeline := long
```

-   위 코드는 4개의 단계를 하나로 연결함

    1. 끊임없이 0의 스트림을 생성하는 반복 단계
    2. 3개의 아이템을 받으면 그 이전 단계를 취소하는 단계
    3. 1초간 슬립하는 짧은 단계
    4. 4초간 슬립하는 긴 단계

-   (1),(2)는 즉각적임을 가정하고, (3),(4)가 어떻게 작동하는지 분석해보면 아래와 같음
    -   파이프라인을 실행하는데 대략 13초가 걸림

| 시간(t) |  i  | Long 단계 | Short 단계 |
| :-----: | :-: | :-------: | :--------: |
|    0    |  0  |           |    1초     |
|    1    |  0  |    4초    |    1초     |
|    2    |  0  |    3초    |   (대기)   |
|    3    |  0  |    2초    |   (대기)   |
|    4    |  0  |    1초    |   (대기)   |
|    5    |  1  |    4초    |    1초     |
|    6    |  1  |    3초    |   (대기)   |
|    7    |  1  |    2초    |   (대기)   |
|    8    |  1  |    1초    |   (대기)   |
|    9    |  2  |    4초    |   (닫힘)   |
|   10    |  2  |    3초    |            |
|   11    |  2  |    2초    |            |
|   12    |  2  |    1초    |            |
|   13    |  3  |  (닫힘)   |            |

-   이를 버퍼를 포함하도록 파이프라인을 수정하면, 아래 코드와 같음

```golang
done := make(chan interface{})
defer close(done)

zeros := take(done, 3, repeat(done, 0))
short := sleep(done, 1*time.Second, zeros)
buffer := buffer(done, 2, short) // short로부터 최대 2개까지 받는다.
long := sleep(done, 4*time.Second, buffer)
pipeline := long
```

| 시간(t) |  i  | Long 단계 | 버퍼 | Short 단계 |
| :-----: | :-: | :-------: | :--: | :--------: |
|    0    |  0  |           | 0/2  |    1초     |
|    1    |  0  |    4초    | 0/2  |    1초     |
|    2    |  0  |    3초    | 1/2  |    1초     |
|    3    |  0  |    2초    | 2/2  |   (닫힘)   |
|    4    |  0  |    1초    | 2/2  |            |
|    5    |  1  |    4초    | 1/2  |            |
|    6    |  1  |    3초    | 1/2  |            |
|    7    |  1  |    2초    | 1/2  |            |
|    8    |  1  |    1초    | 1/2  |            |
|    9    |  2  |    4초    | 0/2  |            |
|   10    |  2  |    3초    | 0/2  |            |
|   11    |  2  |    2초    | 0/2  |            |
|   12    |  2  |    1초    | 0/2  |            |
|   13    |  3  |  (닫힘)   |      |            |

-   전체 파이프라인은 여전히 13초가 걸림
-   short단계의 실행시간은 대기열 사용이전에는 9초, 사용이후에는 3초만에 완료됨

-   **대기열 사용**은 단계 중 하나의 실행 시간이 줄어드는데 초점을 맞추는 것이 아니라, **그 단계가 차단 상태에 있는 시간을 줄여주는데** 의의가 있음

```golang
p := processRequest(done, acceptConnection(done, httpHandler))
```

-   위의 코드의 파이프라인은

    1. 취소될 때까지 종료되지 않음
    2. 연결을 수학하는 단계는 파이프라인이 취소될 때까지 계속해서 연결을 수락

    -   이 시나리오에서 대기열을 사용하지 않는다면, processRequest 단계가 acceptConnection 단계를 차단(대기)시키는 것으로 인해 프로그램에 대한 연결이 시간 초과된다. 이를 막기 위해 대기열을 사용한다.
    -   대기열을 사용할 경우 사용자의 요청이 지연될 가능성이 있지만, 서비스를 완전히 거부하지는 않는다.
        -   Q. 대기열도 일종의 버퍼링이라, 가변 버퍼링이나 고정 버퍼링을 하더라도 용량에 대한 제한이 있을 것 같다.

-   이렇듯, 대기열은 단계를 분리해 한 단계의 실행 시간이 다른 단계의 실행 시간에 영향을 미치지 않도록 하는 점에서 유용함
-   대기열 사용으로 단계를 분리하면, 전체 시스템의 런타임 동작이 단계적으로 변경되는데 이는 시스템에 따라 좋거나 나쁠 수 있다.

# 중략.. (보강필요)