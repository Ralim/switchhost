package utilities

// Sanitise is small filters to strip certan symbols from names for system friendlyness
// Mostly removing slashes and trademark/copyright icons and such
// Things that are "ugly" and dont contribute useful information

//And : breaks some certain software when browsing
var BadRunes map[rune]bool = map[rune]bool{'/': false, '™': false, '®': false, '©': false, '\\': false, ':': false}

func CleanName(s string) string {
	out := ""
	for _, x := range s {
		if _, ok := BadRunes[x]; !ok {
			out += string(x)
		}
	}
	return out
}
