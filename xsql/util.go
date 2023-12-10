package xsql

func insertAt(dest, src []any, index int) []any {
	srcLen := len(src)
	if srcLen > 0 {
		oldLen := len(dest)
		dest = append(dest, src...)
		if index < oldLen {
			copy(dest[index+srcLen:], dest[index:])
			copy(dest[index:], src)
		}
	}

	return dest
}
