package jobs

// derefStr safely dereferences a *string, returning "" if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// int64Ptr returns a pointer to v, or nil if v is zero.
func int64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}
