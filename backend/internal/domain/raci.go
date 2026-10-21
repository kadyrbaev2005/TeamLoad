package domain

// RACI check results shown in the Check column.
const (
	RaciOK       = ""
	RaciNoA      = "No A"
	RaciSeveralA = "Several A"
	RaciNoR      = "No R"
)

// ValidRaciValue reports whether v is R, A, C, I or empty.
func ValidRaciValue(v string) bool {
	switch v {
	case "", "R", "A", "C", "I":
		return true
	}
	return false
}

// RaciCheck returns the problem with one activity's values, or RaciOK.
func RaciCheck(values []string) string {
	a, r := 0, 0
	for _, v := range values {
		switch v {
		case "A":
			a++
		case "R":
			r++
		}
	}
	switch {
	case a == 0:
		return RaciNoA
	case a > 1:
		return RaciSeveralA
	case r == 0:
		return RaciNoR
	}
	return RaciOK
}
