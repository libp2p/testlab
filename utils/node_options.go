package utils

type NodeOptions map[string]interface{}

func (opts NodeOptions) String(key string) (string, bool) {
	opt, ok := opts[key]
	if !ok {
		return "", ok
	}

	optStr, ok := opt.(string)
	return optStr, ok
}

func (opts NodeOptions) Bool(key string) (bool, bool) {
	opt, ok := opts[key]
	if !ok {
		return false, false
	}

	optBool, ok := opt.(bool)
	return optBool, ok
}

func (opts NodeOptions) Int(key string) (int, bool) {
	opt, ok := opts[key]
	if !ok {
		return 0, ok
	}

	optInt, ok := opt.(float64)
	return (int)(optInt), ok
}

func (opts NodeOptions) Float(key string) (float64, bool) {
	opt, ok := opts[key]
	if !ok {
		return 0, ok
	}

	optInt, ok := opt.(float64)
	return optInt, ok
}

func (opts NodeOptions) Object(key string) (NodeOptions, bool) {
	opt, ok := opts[key]
	if !ok {
		return nil, ok
	}

	obj, ok := opt.(NodeOptions)
	return obj, ok
}

func (opts NodeOptions) Slice(key string) ([]interface{}, bool) {
	opt, ok := opts[key]
	if !ok {
		return nil, ok
	}

	obj, ok := opt.([]interface{})
	return obj, ok
}

func (opts NodeOptions) StringSlice(key string) ([]string, bool) {
	slice, ok := opts.Slice(key)
	if !ok {
		return nil, ok
	}

	stringSlice := make([]string, len(slice))
	for i, item := range slice {
		str, ok := item.(string)
		if !ok {
			return nil, ok
		}
		stringSlice[i] = str
	}

	return stringSlice, true
}
