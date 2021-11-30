package utils

func CString(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	end := len(data)
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			end = i
			break
		}
	}
	return string(data[0:end])
}
