package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

type executionResult struct {
	order int
	value string
}

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	inCh := make(chan interface{})
	outCh := make(chan interface{})
	for _, jobFunc := range jobs {
		wg.Add(1)
		go func(task job, in, out chan interface{}) {
			defer close(out)
			defer wg.Done()
			task(in, out)
		}(jobFunc, inCh, outCh)
		inCh = outCh
		outCh = make(chan interface{})
	}
	wg.Wait()
}

// SingleHash считает значение crc32(data)+"~"+crc32(md5(data)) ( конкатенация двух строк через ~),
// где data - то что пришло на вход (по сути - числа из первой функции)
func SingleHash(in, out chan interface{}) {
	var mu = &sync.Mutex{}
	var waitGroup sync.WaitGroup

	for v := range in {
		waitGroup.Add(1)
		go func(value interface{}) {
			defer waitGroup.Done()
			data := getStringFromChannel(value)
			// fmt.Println("SingleHash data " + data)

			mu.Lock()
			md5 := DataSignerMd5(data)
			// fmt.Println("SingleHash md5(data) " + md5)
			mu.Unlock()

			results := calcHashData(data, md5)
			evaluated := make(map[int]string)
			for i := range results {
				evaluated[i.order] = i.value
				// fmt.Println("SingleHash crc32 " + evaluated[i.order])
			}

			out <- evaluated[0] + "~" + evaluated[1]
		}(v)
	}

	waitGroup.Wait()
}

func calcHashData(data string, md5 string) <-chan executionResult {
	var wg sync.WaitGroup
	wg.Add(2)
	results := make(chan executionResult, 2)
	var getCrc32 = func(o int, d string) {
		defer wg.Done()
		results <- executionResult{order: o, value: DataSignerCrc32(d)}
	}

	go getCrc32(0, data)
	go getCrc32(1, md5)
	wg.Wait()
	close(results)
	return results
}

// MultiHash считает значение crc32(th+data)) (конкатенация цифры, приведённой к строке и строки), где th=0..5
// ( т.е. 6 хешей на каждое входящее значение ), потом берёт конкатенацию результатов в порядке расчета (0..5),
// где data - то что пришло на вход (и ушло на выход из SingleHash)
func MultiHash(in, out chan interface{}) {
	var waitGroup sync.WaitGroup

	for v := range in {
		waitGroup.Add(1)
		go func(value interface{}) {
			defer waitGroup.Done()
			data := getStringFromChannel(value)

			results := calcSixHashes(data)
			evaluated := make(map[int]string)
			for i := range results {
				evaluated[i.order] = i.value
			}

			finalResult := ""
			for i := 0; i < 6; i++ {
				finalResult += evaluated[i]
			}
			out <- finalResult
		}(v)
	}

	waitGroup.Wait()
}

func calcSixHashes(data string) <-chan executionResult {
	var wg sync.WaitGroup
	wg.Add(6)

	results := make(chan executionResult, 6)
	for i := 0; i < 6; i++ {
		go func(index int, input string) {
			defer wg.Done()
			r := DataSignerCrc32(strconv.Itoa(index) + input)
			results <- executionResult{order: index, value: r}
		}(i, data)
	}

	wg.Wait()
	close(results)
	return results
}

// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/),
// объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
func CombineResults(in, out chan interface{}) {
	res := make([]string, 0, 16)
	for v := range in {
		res = append(res, getStringFromChannel(v))
	}

	sort.Strings(res)
	result := strings.Join(res, "_")
	out <- result
}

func getStringFromChannel(in interface{}) string {
	switch t := (in).(type) {
	case int:
		return strconv.Itoa(t)
	case string:
		return t

	default:
		panic("unnonw type in channel")
	}
}
