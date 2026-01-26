package regex

import (
	"pegParser/scanner"
	"regexp"
)

var memo = make(map[string]*regexp.Regexp)

func RunRegex(s *scanner.Scanner, rule string) (bool, string) {
	var r *regexp.Regexp
	if rr, ok := memo[rule]; ok {
		r = rr
	} else {
		r = regexp.MustCompile(rule)
		memo[rule] = r
	}
	pos := s.Mark()
	text := s.Text()
	locs := r.FindStringSubmatchIndex(text)

	if len(locs) == 0 {
		return false, ""
	} else if len(locs) == 2 {
		if locs[0] == 0 {
			s.Reset(pos + locs[1])
			return true, text[locs[0]:locs[1]]
		}
	} else {
		if locs[2] == 0 {
			s.Reset(pos + locs[3])
			return true, text[locs[2]:locs[3]]
		}
	}
	return false, ""
}
