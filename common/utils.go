package common

import (
	"flag"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spaolacci/murmur3"
)

func Success(args ...interface{}) {
	if args[len(args)-1] != nil {
		panic(args[len(args)-1])
	}
}

func Hash64(data []byte) int64 {
	return int64(murmur3.Sum64(data))
}

func Hash128(data []byte) (int64, int64) {
	h1, h2 := murmur3.Sum128(data)
	return int64(h1), int64(h2)
}

func ParseArgs() string {
	path := flag.String("conf", "hornet.yaml", "conf file path")

	flag.Parse()
	return *path
}

// ParseSize parses the size string into bytes. The size string must match the regex pattern "^[0-9]+[KMGT]?B?$".
func ParseSize(size string) int64 {
	re := regexp.MustCompile(`^([0-9]+)([KMGTkmgt]?)(B?)$`)
	match := re.FindStringSubmatch(strings.ToUpper(size))
	if match == nil {
		panic(fmt.Errorf("invalid size format: %s", size))
	}

	base, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		panic(fmt.Errorf("invalid size format: %s", size))
	}

	unit := match[2]
	switch unit {
	case "K", "k":
		base *= 1024
	case "M", "m":
		base *= 1024 * 1024
	case "G", "g":
		base *= 1024 * 1024 * 1024
	case "T", "t":
		base *= 1024 * 1024 * 1024 * 1024
	}

	return base
}

func IsIPPort(str string) bool {
	ipPortPattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+$`)
	return ipPortPattern.MatchString(str)
}
