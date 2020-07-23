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
            // 채널에 대하 ㄴ작업
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
