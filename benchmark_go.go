package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("----------------------------------------")
	fmt.Println("    Go Execution Performance Benchmark  ")
	fmt.Println("----------------------------------------")

	start := time.Now()

	limit := 50000
	count := 0

	for i := 2; i <= limit; i++ {
		isPrime := true
		for j := 2; j*j <= i; j++ {
			if i%j == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			count++
		}
	}

	elapsed := time.Since(start)

	fmt.Printf("Target Limit    : %d\n", limit)
	fmt.Printf("Primes Found    : %d\n", count)
	fmt.Printf("Execution Time  : %f seconds\n", elapsed.Seconds())
	fmt.Println("----------------------------------------")
}
