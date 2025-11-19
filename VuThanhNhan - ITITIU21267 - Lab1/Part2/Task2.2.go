package main

import (
	"fmt"
)

// Format price for display
func formatPrice(price float64) string {
	if price >= 1000000000 {
		billions := price / 1000000000
		return fmt.Sprintf("%.1f tỷ VND", billions)
	}
	millions := price / 1000000
	return fmt.Sprintf("%.0f triệu VND", millions)
}

// Group properties by district
func analyzeByDistrict(properties []Property) map[string][]Property {
	districtMap := make(map[string][]Property)
	for _, prop := range properties {
		districtMap[prop.District] = append(districtMap[prop.District], prop)
	}
	return districtMap
}

// District statistics struct
type DistrictStats struct {
	District      string
	PropertyCount int
	AveragePrice  float64
	MostExpensive Property
}

// Calculate district statistics
func calculateDistrictStats(districtMap map[string][]Property) []DistrictStats {
	var stats []DistrictStats

	for district, props := range districtMap {
		totalPrice := 0.0
		mostExpensive := props[0]

		for _, prop := range props {
			totalPrice += prop.Price
			if prop.Price > mostExpensive.Price {
				mostExpensive = prop
			}
		}

		avgPrice := totalPrice / float64(len(props))

		stats = append(stats, DistrictStats{
			District:      district,
			PropertyCount: len(props),
			AveragePrice:  avgPrice,
			MostExpensive: mostExpensive,
		})
	}

	return stats
}
