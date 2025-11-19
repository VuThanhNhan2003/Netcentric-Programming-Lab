package main

import (
	"fmt"
	"sort"
)

// Property struct to organize data better
type Property struct {
	Name     string
	Price    float64
	Area     float64
	Bedrooms int
	District string
}

func main() {
	// Sample properties
	properties := []Property{
		{"Saigon Apartment", 2500000000, 75.5, 2, "District 1"},
		{"HCMC House", 4200000000, 120.0, 3, "District 7"},
		{"Budget Studio", 800000000, 35.0, 1, "Binh Thanh"},
		{"Luxury Penthouse", 5500000000, 150.0, 3, "District 1"},
		{"Cozy Condo", 1800000000, 60.0, 2, "District 7"},
	}

	fmt.Println("=== All Properties ===")
	for i, prop := range properties {
		pricePerM2 := prop.Price / prop.Area
		fmt.Printf("%d. %s: %s (%.0f VND/m²)\n",
			i+1, prop.Name, formatPrice(prop.Price), pricePerM2)
	}

	// TASK 2.1: Search functions
	// Find properties under budget
	// In main(), test your functions:
	budget := 3000000000.0
	affordable := findPropertiesInBudget(properties, budget)
	fmt.Printf("\n=== Properties under %s ===\n", formatPrice(budget))
	for _, prop := range affordable {
		fmt.Printf("- %s: %s\n", prop.Name, formatPrice(prop.Price))
	}

	// Find properties by bedrooms
	bedroomSearch := 2
	byBedrooms := findPropertiesByBedrooms(properties, bedroomSearch)
	fmt.Printf("\n=== Properties with %d bedrooms ===\n", bedroomSearch)
	for _, prop := range byBedrooms {
		fmt.Printf("- %s: %s\n", prop.Name, formatPrice(prop.Price))
	}

	// TASK 2.2: District analysis
	// District analysis
	districtMap := analyzeByDistrict(properties)
	stats := calculateDistrictStats(districtMap)

	fmt.Println("\n=== District Analysis ===")
	for _, stat := range stats {
		fmt.Printf("%s: %d properties, Avg: %s, Most expensive: %s\n",
			stat.District,
			stat.PropertyCount,
			formatPrice(stat.AveragePrice),
			stat.MostExpensive.Name)
	}

	// Sort districts by average price descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].AveragePrice > stats[j].AveragePrice
	})

	fmt.Println("\n=== Ranking by Average Price ===")
	for i, stat := range stats {
		fmt.Printf("%d. %s: %s\n", i+1, stat.District, formatPrice(stat.AveragePrice))
	}
}
