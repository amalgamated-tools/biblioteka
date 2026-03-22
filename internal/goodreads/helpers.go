package goodreads

import "fmt"

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
	for i := 0; i < 9; i++ {
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
	for i := 0; i < 13; i++ {
		d := int(digits[i] - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	return sum%10 == 0
}

// ConvertISBN10To13 converts a valid ISBN-10 string to its corresponding ISBN-13 string.
// A 10-digit ISBN is converted to a 13-digit ISBN by prepending "978" to the ISBN-10 and
// recalculating the final checksum digit using the ISBN-13 algorithm. The reverse process
// can also be performed, but not for numbers commencing with a prefix other than 978,
// which have no 10-digit equivalent.
func ConvertISBN10To13(isbn10 string) (string, error) {
	if !ValidISBN10CheckDigit(isbn10) {
		return "", fmt.Errorf("invalid ISBN-10: %s", isbn10)
	}
	isbnDigits := ""
	for _, r := range isbn10 {
		if r >= '0' && r <= '9' {
			isbnDigits += string(r)
		} else if (r == 'X' || r == 'x') && len(isbnDigits) == 9 {
			isbnDigits += "X"
		}
	}
	if len(isbnDigits) != 10 {
		return "", fmt.Errorf("invalid ISBN-10: %s", isbn10)
	}
	isbn13Digits := "978" + isbnDigits[:9]
	sum := 0
	for i := 0; i < 12; i++ {
		d := int(isbn13Digits[i] - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	checkDigit := (10 - (sum % 10)) % 10
	return isbn13Digits + fmt.Sprintf("%d", checkDigit), nil
}
