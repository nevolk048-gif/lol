package utils

// MajorPay works with minor units (kopecks/cents)
// These helpers convert between rubles and kopecks

// RublesToKopecks converts rubles to kopecks (minor units)
// Example: 1500.00 RUB -> 150000 kopecks
func RublesToKopecks(rubles float64) int64 {
	return int64(rubles * 100)
}

// KopecksToRubles converts kopecks (minor units) to rubles
// Example: 150000 kopecks -> 1500.00 RUB
func KopecksToRubles(kopecks int64) float64 {
	return float64(kopecks) / 100.0
}

// ValidateMinorAmount checks if amount in kopecks is within MajorPay limits
// Min: 50 RUB (5000 kopecks), Max: 500,000 RUB (50,000,000 kopecks)
// Exception: mobile commerce allows from 50 RUB, standard min is 1000 RUB
func ValidateMinorAmount(kopecks int64, isMobileCommerce bool) bool {
	minKopecks := int64(100000) // 1000 RUB standard minimum
	if isMobileCommerce {
		minKopecks = 5000 // 50 RUB for mobile
	}
	maxKopecks := int64(50000000) // 500,000 RUB maximum

	return kopecks >= minKopecks && kopecks <= maxKopecks
}
