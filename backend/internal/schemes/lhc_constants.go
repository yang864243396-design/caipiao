package schemes

import (
	"strconv"
	"strings"
)

// 2026 马年生肖号码表（平台内置；第三方赔率/开奖为准）
var lhcZodiacNumbers = map[string][]int{
	"马": {1, 13, 25, 37, 49},
	"蛇": {2, 14, 26, 38},
	"龙": {3, 15, 27, 39},
	"兔": {4, 16, 28, 40},
	"虎": {5, 17, 29, 41},
	"牛": {6, 18, 30, 42},
	"鼠": {7, 19, 31, 43},
	"猪": {8, 20, 32, 44},
	"狗": {9, 21, 33, 45},
	"鸡": {10, 22, 34, 46},
	"猴": {11, 23, 35, 47},
	"羊": {12, 24, 36, 48},
}

var lhcDomesticZodiacs = map[string]bool{
	"牛": true, "马": true, "羊": true, "鸡": true, "狗": true, "猪": true,
}

var lhcRedNumbers = map[int]bool{
	1: true, 2: true, 7: true, 8: true, 12: true, 13: true, 18: true, 19: true,
	23: true, 24: true, 29: true, 30: true, 34: true, 35: true, 40: true, 45: true, 46: true,
}

var lhcBlueNumbers = map[int]bool{
	3: true, 4: true, 9: true, 10: true, 14: true, 15: true, 20: true, 25: true,
	26: true, 31: true, 36: true, 37: true, 41: true, 42: true, 47: true, 48: true,
}

var lhcGreenNumbers = map[int]bool{
	5: true, 6: true, 11: true, 16: true, 17: true, 21: true, 22: true, 27: true,
	28: true, 32: true, 33: true, 38: true, 39: true, 43: true, 44: true, 49: true,
}

var lhcElementNumbers = map[string]map[int]bool{
	"金": intSet(3, 4, 11, 12, 25, 26, 33, 34, 41, 42),
	"木": intSet(7, 8, 15, 16, 23, 24, 37, 38, 45, 46),
	"水": intSet(1, 2, 9, 10, 17, 18, 31, 32, 39, 40, 47, 48),
	"火": intSet(5, 6, 13, 14, 21, 22, 29, 30, 43, 44),
	"土": intSet(19, 20, 27, 28, 35, 36, 49),
}

func intSet(nums ...int) map[int]bool {
	m := make(map[int]bool, len(nums))
	for _, n := range nums {
		m[n] = true
	}
	return m
}

func parseLHCNumbers(raw string) []int {
	parts := parseTextTokens(raw)
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSuffix(strings.TrimSpace(p), "||")
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 49 {
			continue
		}
		out = append(out, n)
	}
	return out
}

func parseLHCZodiacs(raw string) []string {
	parts := parseTextTokens(raw)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if _, ok := lhcZodiacNumbers[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

func lhcZhengma(balls []string) []int {
	if len(balls) < 6 {
		return nil
	}
	out := make([]int, 0, 6)
	for i := 0; i < 6; i++ {
		out = append(out, atoiBall(balls[i]))
	}
	return out
}

func lhcTema(balls []string) int {
	if len(balls) < 7 {
		return 0
	}
	return atoiBall(balls[6])
}

func lhcAllNumbers(balls []string) []int {
	out := make([]int, 0, len(balls))
	for _, b := range balls {
		n := atoiBall(b)
		if n >= 1 && n <= 49 {
			out = append(out, n)
		}
	}
	return out
}

func lhcNumberSet(nums []int) map[int]bool {
	m := make(map[int]bool, len(nums))
	for _, n := range nums {
		m[n] = true
	}
	return m
}

func lhcZodiacOf(n int) string {
	for z, nums := range lhcZodiacNumbers {
		for _, v := range nums {
			if v == n {
				return z
			}
		}
	}
	return ""
}

func lhcColorOf(n int) string {
	switch {
	case lhcRedNumbers[n]:
		return "红"
	case lhcBlueNumbers[n]:
		return "蓝"
	case lhcGreenNumbers[n]:
		return "绿"
	default:
		return ""
	}
}

func lhcElementOf(n int) string {
	for e, set := range lhcElementNumbers {
		if set[n] {
			return e
		}
	}
	return ""
}

func lhcTailOf(n int) int {
	if n <= 0 {
		return -1
	}
	return n % 10
}

func lhcHeadOf(n int) int {
	if n < 10 {
		return 0
	}
	return n / 10
}

func lhcCountInSet(drawn []int, picks []int) int {
	set := lhcNumberSet(drawn)
	c := 0
	for _, p := range picks {
		if set[p] {
			c++
		}
	}
	return c
}

func lhcCombinations(nums []int, k int) [][]int {
	if k <= 0 || k > len(nums) {
		return nil
	}
	var out [][]int
	var buf []int
	var dfs func(start int)
	dfs = func(start int) {
		if len(buf) == k {
			c := make([]int, k)
			copy(c, buf)
			out = append(out, c)
			return
		}
		for i := start; i < len(nums); i++ {
			buf = append(buf, nums[i])
			dfs(i + 1)
			buf = buf[:len(buf)-1]
		}
	}
	dfs(0)
	return out
}

func lhcBuzhongCount(subID string) int {
	switch subID {
	case "5bz":
		return 5
	case "6bz":
		return 6
	case "7bz":
		return 7
	case "8bz":
		return 8
	case "9bz":
		return 9
	case "10bz":
		return 10
	case "11bz":
		return 11
	case "12bz":
		return 12
	case "15bz":
		return 15
	default:
		return 5
	}
}

func lhcXuanyiCount(subID string) int {
	switch subID {
	case "5x1":
		return 5
	case "6x1":
		return 6
	case "7x1":
		return 7
	case "8x1":
		return 8
	case "9x1":
		return 9
	case "10x1":
		return 10
	default:
		return 5
	}
}

func lhcRenzhongCount(subID string) int {
	switch subID {
	case "1l_rz":
		return 1
	case "2l_rz":
		return 2
	case "3l_rz":
		return 3
	case "4l_rz":
		return 4
	case "5l_rz":
		return 5
	default:
		return 1
	}
}

func lhcXiaoCount(subID string) int {
	switch {
	case subID == "1xiao" || subID == "1xiao_bz":
		return 1
	case len(subID) >= 2 && subID[1] == 'x':
		n, _ := strconv.Atoi(subID[:1])
		if n >= 2 && n <= 6 {
			return n
		}
	}
	return 1
}

func lhcWeiCount(subID string) int {
	switch {
	case subID == "2wei_z", subID == "2wei_bz":
		return 2
	case subID == "3wei_z", subID == "3wei_bz":
		return 3
	case subID == "4wei_z", subID == "4wei_bz":
		return 4
	default:
		return 1
	}
}

func lhcZodiacNums(zodiacs []string) []int {
	seen := map[int]bool{}
	var out []int
	for _, z := range zodiacs {
		for _, n := range lhcZodiacNumbers[z] {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out
}

func lhcDistinctZodiacs(nums []int) map[string]bool {
	m := map[string]bool{}
	for _, n := range nums {
		if z := lhcZodiacOf(n); z != "" {
			m[z] = true
		}
	}
	return m
}

// 七码（qima）：在 7 个开奖号中统计 单/双/大/小 个数，与选项「双1」等比对。
var lhcQimaKinds = []string{"单", "双", "大", "小"}

type lhcQimaPick struct {
	kind  string
	count int
}

func parseLHCQimaOption(token string) (kind string, count int, ok bool) {
	token = strings.TrimSpace(token)
	for _, k := range lhcQimaKinds {
		if !strings.HasPrefix(token, k) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(token, k))
		if err == nil && n >= 0 && n <= 7 {
			return k, n, true
		}
	}
	return "", 0, false
}

func parseLHCQimaPicks(raw string) []lhcQimaPick {
	parts := parseTextTokens(raw)
	out := make([]lhcQimaPick, 0, len(parts))
	for _, p := range parts {
		if k, n, ok := parseLHCQimaOption(p); ok {
			out = append(out, lhcQimaPick{kind: k, count: n})
		}
	}
	return out
}

func lhcQimaCategoryCount(nums []int, kind string) int {
	c := 0
	for _, n := range nums {
		if n < 1 || n > 49 {
			continue
		}
		switch kind {
		case "单":
			if n%2 == 1 {
				c++
			}
		case "双":
			if n%2 == 0 {
				c++
			}
		case "大":
			if n >= 25 {
				c++
			}
		case "小":
			if n <= 24 {
				c++
			}
		}
	}
	return c
}

func lhcQimaKindIndex(kind string) int {
	switch kind {
	case "单":
		return 0
	case "双":
		return 1
	case "大":
		return 2
	case "小":
		return 3
	default:
		return -1
	}
}

// lhcQimaOddsTable 与 hash.iyes.dev 七码面板一致（单/双/大/小 × 0–7）。
var lhcQimaOddsTable = [4][8]float64{
	{235.287, 24.541, 6.537, 3.433, 3.239, 5.645, 19.497, 171.69},
	{170.72, 19.497, 5.645, 3.239, 3.433, 6.537, 24.541, 234.74},
	{234.74, 24.541, 6.537, 3.433, 3.239, 5.645, 19.497, 171.69},
	{171.69, 19.497, 5.645, 3.239, 3.433, 6.537, 24.541, 234.74},
}

func lhcQimaOdds(kind string, count int) float64 {
	if count < 0 || count > 7 {
		return oddsLHCAttr
	}
	if i := lhcQimaKindIndex(kind); i >= 0 {
		return lhcQimaOddsTable[i][count]
	}
	return oddsLHCAttr
}

const oddsLHCTema = 48.0
const oddsLHCCombo = 24.0
const oddsLHCAttr = 2.0
