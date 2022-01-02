package library

import "fmt"

/*************** Below are small formatting helpers ***************/
func FormatTitleIDToString(titleID uint64) string {
	//Format titleID out as a fixed with hex string
	return fmt.Sprintf("%016X", titleID)
}

func FormatVersionToString(version uint32) string {
	//Format as decimal with a v prefix
	return fmt.Sprintf("v%d", version)
}

func FormatVersionToHumanString(version uint32) string {
	if version == 0 {
		return "" //This is base game, no point
	}
	/*
		https://switchbrew.org/wiki/Title_list
			Decimal versions use the format:

			Bit31-Bit26: Major
			Bit25-Bit20: Minor
			Bit19-Bit16: Micro
			Bit15-Bit0: Bugfix

		Dont know if games use this exact format, but ok for now :shrug:
		Using leading zero suppression to make things a little easier to read
	*/
	major := version >> 26
	minor := version >> 20 & 0b111111
	micro := version >> 16 & 0b1111
	bugfix := version & 0xFFFF
	if major != 0 {
		return fmt.Sprintf("v%d.%d.%d.%d", major, minor, micro, bugfix)
	} else if minor != 0 {
		return fmt.Sprintf("v%d.%d.%d", minor, micro, bugfix)
	} else if micro != 0 {
		return fmt.Sprintf("v%d.%d", micro, bugfix)
	}
	return fmt.Sprintf("v%d", bugfix)
}
