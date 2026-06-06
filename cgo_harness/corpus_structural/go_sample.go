package main

import (
	"fmt"
	"os"
	"strings"
)

// FunctionDefinition with multiple parameters and return types
func processFile(path string, maxSize int64) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("file too large: %d > %d", len(data), maxSize)
	}
	return data, nil
}

// Method with receiver, interface, and struct
type Parser interface {
	Parse(source []byte) (*Tree, error)
	SetLanguage(lang *Language) error
}

type Tree struct {
	root     *Node
	source   []byte
	hasError bool
}

type Node struct {
	kind      string
	startByte uint32
	endByte   uint32
	children  []*Node
	parent    *Node
}

// ChildByFieldName — critical for xgrep pattern matching
func (n *Node) ChildByFieldName(name string) *Node {
	for _, child := range n.children {
		if child.kind == name {
			return child
		}
	}
	return nil
}

// Switch statement with type assertion
func analyzeNode(n *Node) string {
	switch n.kind {
	case "function_definition":
		body := n.ChildByFieldName("body")
		if body == nil {
			return "empty function"
		}
		return fmt.Sprintf("function with %d statements", len(body.children))
	case "class_definition":
		name := n.ChildByFieldName("name")
		if name != nil {
			return "class: " + name.kind
		}
		return "anonymous class"
	default:
		return "unknown: " + n.kind
	}
}

// Goroutine, channel, select — concurrent patterns
func worker(jobs <-chan string, results chan<- string) {
	for job := range jobs {
		result := strings.ToUpper(job)
		results <- result
	}
}

// Defer, panic, recover
func safeDivide(a, b float64) (result float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if b == 0 {
		panic("division by zero")
	}
	return a / b, nil
}

// Generics (Go 1.18+)
func Map[T any, U any](slice []T, f func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}
	return result
}

// Embedded struct and promoted methods
type BaseParser struct {
	language string
	timeout  int
}

type AdvancedParser struct {
	BaseParser
	maxRetries int
}
