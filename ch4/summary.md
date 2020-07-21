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
