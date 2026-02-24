package helper

import "strconv"

func ParseFloat32(s string) (float32, error) {
	f64, err := strconv.ParseFloat(s, 32)
	return float32(f64), err
}

func ParseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
