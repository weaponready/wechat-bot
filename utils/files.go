package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ReadMapFromFile(filePath string) (map[string]string, error) {
	aliases := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return aliases, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			fmt.Printf("Invalid line: %s\n", line)
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		aliases[key] = value
	}
	// 检查是否读取过程中有错误
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return aliases, err
	}
	// 打印读取到的 map
	fmt.Println("Loaded aliases:")
	for k, v := range aliases {
		fmt.Printf("%s -> %s\n", k, v)
	}
	return aliases, nil
}
