package api

import (
	"fmt"
	"strconv"
	"strings"
)

// PrintGoroutinesFlags 控制如何打印goroutine信息
type PrintGoroutinesFlags uint8

const (
	PrintGoroutinesStack  PrintGoroutinesFlags = 1 << iota // 打印goroutine堆栈
	PrintGoroutinesLabels                                  // 打印goroutine labels
)

// FormatGoroutineLoc 控制如何格式化goroutine位置信息
type FormatGoroutineLoc int

const (
	FormatGLocRuntimeCurrent = FormatGoroutineLoc(iota) // 打印goroutine当前地址
	FormatGLocUserCurrent                               // 打印goroutine当前地址，排除runtime调用部分
	FormatGLocGo                                        // 打印启动当前goroutine的指令地址
	FormatGLocStart                                     // 打印启动当前goroutine的函数地址
)

const (
	maxGroupMembers    = 5
	maxGoroutineGroups = 50
)

// The number of goroutines we're going to request on each RPC call
const goroutineBatchSize = 10000

// ParseGoroutineArgs parse goroutine's arguments
func ParseGoroutineArgs(argstr string) ([]ListGoroutinesFilter, GoroutineGroupingOptions, FormatGoroutineLoc, PrintGoroutinesFlags, int, int, error) {
	var (
		filters   []ListGoroutinesFilter
		group     GoroutineGroupingOptions
		fgl       = FormatGLocUserCurrent
		flags     PrintGoroutinesFlags
		depth     = 10
		batchSize = goroutineBatchSize
	)
	group.MaxGroupMembers = maxGroupMembers
	group.MaxGroups = maxGoroutineGroups

	args := strings.Split(argstr, " ")
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-u":
			fgl = FormatGLocUserCurrent
		case "-r":
			fgl = FormatGLocRuntimeCurrent
		case "-g":
			fgl = FormatGLocGo
		case "-s":
			fgl = FormatGLocStart
		case "-l":
			flags |= PrintGoroutinesLabels
		case "-t":
			flags |= PrintGoroutinesStack
			// optional depth argument
			if i+1 < len(args) && len(args[i+1]) > 0 {
				n, err := strconv.Atoi(args[i+1])
				if err == nil {
					depth = n
					i++
				}
			}
		case "-w", "-with":
			filter, err := readGoroutinesFilter(args, &i)
			if err != nil {
				return nil, GoroutineGroupingOptions{}, 0, 0, 0, 0, fmt.Errorf("wrong argument: '%s'", arg)
			}
			filters = append(filters, *filter)
		case "-wo", "-without":
			filter, err := readGoroutinesFilter(args, &i)
			if err != nil {
				return nil, GoroutineGroupingOptions{}, 0, 0, 0, 0, fmt.Errorf("wrong argument: '%s'", arg)
			}
			filter.Negated = true
			filters = append(filters, *filter)

		case "-group":
			var err error
			group.GroupBy, err = readGoroutinesFilterKind(args, i+1)
			if err != nil {
				return nil, GoroutineGroupingOptions{}, 0, 0, 0, 0, fmt.Errorf("wrong argument: '%s'", arg)
			}
			i++
			if group.GroupBy == GoroutineLabel {
				if i+1 >= len(args) {
					return nil, GoroutineGroupingOptions{}, 0, 0, 0, 0, fmt.Errorf("wrong argument: '%s'", arg)
				}
				group.GroupByKey = args[i+1]
				i++
			}
			batchSize = 0 // grouping only works well if run on all goroutines
		case "":
			// nothing to do
		default:
			return nil, GoroutineGroupingOptions{}, 0, 0, 0, 0, fmt.Errorf("wrong argument: '%s'", arg)
		}
	}
	return filters, group, fgl, flags, depth, batchSize, nil
}

func readGoroutinesFilterKind(args []string, i int) (GoroutineField, error) {
	if i >= len(args) {
		return GoroutineFieldNone, fmt.Errorf("%s must be followed by an argument", args[i-1])
	}

	switch args[i] {
	case "curloc":
		return GoroutineCurrentLoc, nil
	case "userloc":
		return GoroutineUserLoc, nil
	case "goloc":
		return GoroutineGoLoc, nil
	case "startloc":
		return GoroutineStartLoc, nil
	case "label":
		return GoroutineLabel, nil
	case "running":
		return GoroutineRunning, nil
	case "user":
		return GoroutineUser, nil
	default:
		return GoroutineFieldNone, fmt.Errorf("unrecognized argument to %s %s", args[i-1], args[i])
	}
}

func readGoroutinesFilter(args []string, pi *int) (*ListGoroutinesFilter, error) {
	r := new(ListGoroutinesFilter)
	var err error
	r.Kind, err = readGoroutinesFilterKind(args, *pi+1)
	if err != nil {
		return nil, err
	}
	*pi++
	switch r.Kind {
	case GoroutineRunning, GoroutineUser:
		return r, nil
	}
	if *pi+1 >= len(args) {
		return nil, fmt.Errorf("%s %s needs to be followed by an expression", args[*pi-1], args[*pi])
	}
	r.Arg = args[*pi+1]
	*pi++

	return r, nil
}
