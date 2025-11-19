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

// Find properties within maxBudget
func findPropertiesInBudget(properties []Property, maxBudget float64) []Property {
    var result []Property
    for _, prop := range properties {
        if prop.Price <= maxBudget {
            result = append(result, prop)
        }
    }
    return result
}

// Find properties by number of bedrooms
func findPropertiesByBedrooms(properties []Property, bedrooms int) []Property {
    var result []Property
    for _, prop := range properties {
        if prop.Bedrooms == bedrooms {
            result = append(result, prop)
        }
    }
    return result
}

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
    District       string
    PropertyCount  int
    AveragePrice   float64
    MostExpensive  Property
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
