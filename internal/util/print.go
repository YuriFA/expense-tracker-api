package util

import (
	"encoding/json"
	"fmt"
)

func PrintJSON(obj any) {
	bytes, _ := json.MarshalIndent(obj, "\t", "\t")
	fmt.Println(string(bytes))
}
