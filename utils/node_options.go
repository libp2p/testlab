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
