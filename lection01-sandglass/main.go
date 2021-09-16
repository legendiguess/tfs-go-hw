package main

import "fmt"

type Size int

type Color int

func (color Color) String() string {
	switch color {
	case colorReset:
		return "\033[0m"
	case colorRed:
		return "\033[31m"
	case colorGreen:
		return "\033[32m"
	case colorYellow:
		return "\033[33m"
	case colorBlue:
		return "\033[34m"
	case colorPurple:
		return "\033[35m"
	case colorCyan:
		return "\033[36m"
	case colorWhite:
		return "\033[37m"
	default:
		return ""
	}
}

const (
	colorReset  = Color(iota)
	colorRed    = Color(iota)
	colorGreen  = Color(iota)
	colorYellow = Color(iota)
	colorBlue   = Color(iota)
	colorPurple = Color(iota)
	colorCyan   = Color(iota)
	colorWhite  = Color(iota)
)

type Sandglass [5]int

func newSandglass() Sandglass {
	return Sandglass([5]int{15, int('⨯'), int(colorYellow), '·', int(colorCyan)})
}

func (sandglass Sandglass) size(newSize Size) Sandglass {
	sandglass[0] = int(newSize)
	return sandglass
}

func (sandglass Sandglass) char(newChar rune) Sandglass {
	sandglass[1] = int(newChar)
	return sandglass
}

func (sandglass Sandglass) charColor(newCharColor Color) Sandglass {
	sandglass[2] = int(newCharColor)
	return sandglass
}

func (sandglass Sandglass) bgChar(newBgChar rune) Sandglass {
	sandglass[3] = int(newBgChar)
	return sandglass
}

func (sandglass Sandglass) bgCharColor(newBgCharColor Color) Sandglass {
	sandglass[4] = int(newBgCharColor)
	return sandglass
}

func (sandglass Sandglass) String() string {
	var outputString string

	var size = sandglass[0]

	var charColored = Color(sandglass[2]).String() + string(rune(sandglass[1]))
	var bgCharColored = Color(sandglass[4]).String() + string(rune(sandglass[3]))

	for row := 0; row < size; row++ {
		for column := 0; column < size; column++ {
			if row == 0 || row == size-1 || row == column || row == size-1-column {
				outputString += charColored
			} else {
				outputString += bgCharColored
			}
		}
		outputString += "\n"
	}

	outputString += colorReset.String()

	return outputString
}

func main() {
	fmt.Print(newSandglass())
	fmt.Print(newSandglass().size(Size(20)))
	fmt.Print(newSandglass().size(Size(13)).char('X').charColor(colorRed))
	fmt.Print(newSandglass().size(Size(14)).char('O').charColor(colorGreen).bgChar('=').bgCharColor(colorWhite))
	fmt.Print(newSandglass().char('/').bgChar('|'))
	fmt.Print(newSandglass().char('J').bgChar('o').charColor(colorPurple).bgCharColor(colorRed).size(Size(25)))
}
