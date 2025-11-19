package main

import "fmt"

// Categorize property based on price per m²
func categorizeProperty(pricePerM2 float64) string {
    if pricePerM2 > 50000000 {
        return "LUXURY"
    } else if pricePerM2 > 30000000 {
        return "PREMIUM"
    } else if pricePerM2 > 20000000 {
        return "STANDARD"
    }
    return "BUDGET"
}

// Format price to billions (tỷ) or millions (triệu)
func formatPrice(price float64) string {
    if price >= 1000000000 {
        billions := price / 1000000000
        return fmt.Sprintf("%.1f tỷ VND", billions)
    }
    millions := price / 1000000
    return fmt.Sprintf("%.0f triệu VND", millions)
}

func main() {
    // Property 1
    var property1Name string = "Saigon Apartment"
    var property1Price float64 = 2500000000
    var property1Area float64 = 75.5
    pricePerM2_1 := property1Price / property1Area

    // Property 2
    var property2Name string = "Hanoi Condo"
    var property2Price float64 = 2800000000
    var property2Area float64 = 90.0
    pricePerM2_2 := property2Price / property2Area

    // Property 3
    var property3Name string = "Da Nang Villa"
    var property3Price float64 = 3200000000
    var property3Area float64 = 120.0
    pricePerM2_3 := property3Price / property3Area

    // Print property comparison
    fmt.Println("=== Property Comparison ===")
    fmt.Printf("Property 1: %s - %.0f VND/m²\n", property1Name, pricePerM2_1)
    fmt.Printf("Property 2: %s - %.0f VND/m²\n", property2Name, pricePerM2_2)
    fmt.Printf("Property 3: %s - %.0f VND/m²\n", property3Name, pricePerM2_3)

    // Created slices to hold names and prices
    names := []string{property1Name, property2Name, property3Name}
    prices := []float64{pricePerM2_1, pricePerM2_2, pricePerM2_3}

    // Find the cheapest per m²
    cheapestName := names[0]
    cheapestPrice := prices[0]

    for i := 1; i < len(prices); i++ {
        if prices[i] < cheapestPrice {
            cheapestPrice = prices[i]
            cheapestName = names[i]
        }
    }

    fmt.Printf("\nCheapest per m²: %s at %.0f VND/m²\n", cheapestName, cheapestPrice)

    category1 := categorizeProperty(pricePerM2_1)
    category2 := categorizeProperty(pricePerM2_2)
    category3 := categorizeProperty(pricePerM2_3)

    luxuryCount := 0
    premiumCount := 0
    standardCount := 0
    budgetCount := 0

    categories := []string{category1, category2, category3}
    for _, cat := range categories {
        if cat == "LUXURY" {
            luxuryCount++
        } else if cat == "PREMIUM" {
            premiumCount++
        } else if cat == "STANDARD" {
            standardCount++
        } else if cat == "BUDGET" {
            budgetCount++
        }
    }

    // Display property categories
    fmt.Println("\n=== Property Categories ===")
    fmt.Printf("%s: %s (%s)\n", property1Name, category1, formatPrice(property1Price))
    fmt.Printf("%s: %s (%s)\n", property2Name, category2, formatPrice(property2Price))
    fmt.Printf("%s: %s (%s)\n", property3Name, category3, formatPrice(property3Price))

    // Display category summary
    fmt.Println("\nCategory Summary:")
    fmt.Printf("LUXURY: %d properties\n", luxuryCount)
    fmt.Printf("PREMIUM: %d properties\n", premiumCount)
    fmt.Printf("STANDARD: %d properties\n", standardCount)
    fmt.Printf("BUDGET: %d properties\n", budgetCount)
}