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
	for _, a := range add(multiply(ints, 2), 1) {
		fmt.Println(a)
	}

	for _, b := range multiply(add(multiply(ints, 2), 1), 2) {
		fmt.Println(b)
	}
}
