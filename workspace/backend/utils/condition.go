package utils

import "fmt"

func AppendStringCondition(field, value string, index int, conditions *[]string, params *[]interface{}) int {
	if value != "" {
		*conditions = append(*conditions, fmt.Sprintf("%s = $%d", field, index))
		*params = append(*params, value)
		index++
	}
	return index
}

func AppendIntCondition(field string, value, index int, conditions *[]string, params *[]interface{}) int {
	if value != 0 {
		*conditions = append(*conditions, fmt.Sprintf("%s = $%d", field, index))
		*params = append(*params, value)
		index++
	}
	return index
}
