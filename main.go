package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

func calculate(a float64, op string, b float64) (float64, error) {
	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, errors.New("division by zero")
		}
		return a / b, nil
	default:
		return 0, fmt.Errorf("unknown operator %q", op)
	}
}

func main() {
	fmt.Println("Calculator. Enter e.g. `3 + 4` (or `q` to quit)")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "q" {
			return
		}

		var a, b float64
		var op string
		if _, err := fmt.Sscanf(line, "%f %s %f", &a, &op, &b); err != nil {
			fmt.Println("invalid input, use: number operator number")
			continue
		}

		result, err := calculate(a, op, b)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Println(result)
	}
}