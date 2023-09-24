package main

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func stringsToInterfaces(sx []string) []interface{} {
	interfaces := make([]interface{}, len(sx))
	for i, s := range sx {
		interfaces[i] = s
	}
	return interfaces
}
