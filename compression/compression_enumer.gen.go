// Code generated by "enumer -type=Flag -output compression_enumer.gen.go"; DO NOT EDIT.

package compression

import "fmt"

const _FlagName = "DefaultGZIPNone"

var _FlagIndex = [...]uint8{0, 7, 11, 15}

func (i Flag) String() string {
	i -= 1
	if i >= Flag(len(_FlagIndex)-1) {
		return fmt.Sprintf("Flag(%d)", i+1)
	}
	return _FlagName[_FlagIndex[i]:_FlagIndex[i+1]]
}

var _FlagValues = []Flag{1, 2, 3}

var _FlagNameToValueMap = map[string]Flag{
	_FlagName[0:7]:   1,
	_FlagName[7:11]:  2,
	_FlagName[11:15]: 3,
}

// FlagString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func FlagString(s string) (Flag, error) {
	if val, ok := _FlagNameToValueMap[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to Flag values", s)
}

// FlagValues returns all values of the enum
func FlagValues() []Flag {
	return _FlagValues
}

// IsAFlag returns "true" if the value is listed in the enum definition. "false" otherwise
func (i Flag) IsAFlag() bool {
	for _, v := range _FlagValues {
		if i == v {
			return true
		}
	}
	return false
}
