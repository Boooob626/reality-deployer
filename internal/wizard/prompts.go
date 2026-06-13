// Package wizard 实现交互式部署向导：收集全部选项 → 校验 → 渲染 → 持久化 manifest。
package wizard

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

// opt 是一个可选项。
type opt struct {
	Key  string
	Desc string
}

// readLine 读取一行并去空白。
func readLine() string {
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimSpace(line)
}

// prompt 打印 label 并读取一行。
func prompt(label string) string {
	fmt.Printf("%s: ", label)
	return readLine()
}

// promptDefault 带默认值的字符串输入。
func promptDefault(label, def string) string {
	v := prompt(fmt.Sprintf("%s [%s]", label, def))
	if v == "" {
		return def
	}
	return v
}

// promptInt 带默认值的整数输入。
func promptInt(label string, def int) int {
	v := prompt(fmt.Sprintf("%s [%d]", label, def))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// confirm 是/否确认。
func confirm(label string, def bool) bool {
	hint := "Y/n"
	if !def {
		hint = "y/N"
	}
	v := prompt(fmt.Sprintf("%s [%s]", label, hint))
	if v == "" {
		return def
	}
	return strings.EqualFold(v, "y") || strings.EqualFold(v, "yes")
}

// selectOpt 单选，返回所选 opt.Key。
func selectOpt(label string, opts []opt, defIdx int) string {
	fmt.Println(label)
	for i, o := range opts {
		fmt.Printf("  %d) %s\n", i+1, o.Desc)
	}
	for {
		v := prompt(fmt.Sprintf("选择 [%d]", defIdx+1))
		idx := defIdx
		if v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				fmt.Println("  无效输入，请输入序号")
				continue
			}
			idx = n - 1
		}
		if idx >= 0 && idx < len(opts) {
			return opts[idx].Key
		}
		fmt.Println("  超出范围，重试")
	}
}

// multiSelect 多选，返回所选下标切片。
func multiSelect(label string, opts []opt, defIdxs []int) []int {
	fmt.Println(label)
	for i, o := range opts {
		fmt.Printf("  %d) %s\n", i+1, o.Desc)
	}
	defStr := make([]string, len(defIdxs))
	for i, d := range defIdxs {
		defStr[i] = strconv.Itoa(d + 1)
	}
	for {
		v := prompt(fmt.Sprintf("选择（空格/逗号分隔，默认 %s）", strings.Join(defStr, ",")))
		if v == "" {
			return defIdxs
		}
		parts := strings.FieldsFunc(v, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' })
		var out []int
		ok := true
		for _, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil || n < 1 || n > len(opts) {
				ok = false
				break
			}
			out = append(out, n-1)
		}
		if ok && len(out) > 0 {
			return out
		}
		fmt.Println("  无效输入，重试")
	}
}
