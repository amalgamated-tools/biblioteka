package goodreads

// ValidISBN10CheckDigit reports whether the given string is a valid ISBN-10
// with a correct check digit. The input may contain hyphens or spaces, which
// are stripped before validation. The check digit may be 'X' (representing 10).
func ValidISBN10CheckDigit(isbn string) bool {
	// Strip hyphens and spaces
	var digits []byte
	for i := 0; i < len(isbn); i++ {
		c := isbn[i]
		if c >= '0' && c <= '9' {
			digits = append(digits, c)
		} else if (c == 'X' || c == 'x') && len(digits) == 9 {
			digits = append(digits, 'X')
		} else if c == '-' || c == ' ' {
			continue
		} else {
			return false
		}
	}
	if len(digits) != 10 {
		return false
	}

	sum := 0
	for i := range 9 {
		sum += int(digits[i]-'0') * (10 - i)
	}
	if digits[9] == 'X' {
		sum += 10
	} else {
		sum += int(digits[9] - '0')
	}
	return sum%11 == 0
}

// ValidISBN13CheckDigit reports whether the given string is a valid ISBN-13
// with a correct check digit. The input may contain hyphens or spaces, which
// are stripped before validation. All 13 characters must be digits.
func ValidISBN13CheckDigit(isbn string) bool {
	var digits []byte
	for i := 0; i < len(isbn); i++ {
		c := isbn[i]
		if c >= '0' && c <= '9' {
			digits = append(digits, c)
		} else if c == '-' || c == ' ' {
			continue
		} else {
			return false
		}
	}
	if len(digits) != 13 {
		return false
	}

	sum := 0
	for i := range 13 {
		d := int(digits[i] - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	return sum%10 == 0
}
