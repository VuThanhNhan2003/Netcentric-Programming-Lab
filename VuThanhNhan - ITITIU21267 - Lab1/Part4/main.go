package main

import (
	"fmt"
)

// --- Main Function ---
func main() {
	properties := []Property{
		{"Saigon Apartment", 2500000000, 75.5, 2, "District 1"},
		{"HCMC House", 4200000000, 120.0, 3, "District 7"},
		{"Budget Studio", 800000000, 35.0, 1, "Binh Thanh"},
		{"Cozy Condo", 1800000000, 65.0, 2, "District 2"},
	}

	monthlyRents := []float64{25000000, 35000000, 12000000, 18000000}

	// TASK 4.1: Smart Recommendation System
	fmt.Println("=== Smart Property Recommendations ===")
	budget := 5000000000.0   // 5 billion VND
	maxMonthly := 30000000.0 // 30 million VND/month

	for _, prop := range properties {
		recommendation, details := smartRecommendProperty(prop, budget, maxMonthly)
		fmt.Printf("\n%s (%.1f m², %s)\n", prop.Name, prop.Area, prop.District)
		fmt.Printf("Price: %s | Price/m²: %s\n",
			formatPrice(prop.Price), formatPrice(prop.PricePerM2()))
		fmt.Printf("Recommendation: %s\n", recommendation)
		fmt.Printf("Details: %s\n", details)
	}

	// TASK 4.2: Portfolio Optimization
	fmt.Println("\n=== Portfolio Optimization ===")
	portfolioBudget := 8000000000.0 // 8 billion VND
	fmt.Printf("Budget: %s\n\n", formatPrice(portfolioBudget))

	portfolio := optimizePortfolio(properties, monthlyRents, portfolioBudget)

	fmt.Println("Selected Properties:")
	totalInvested, avgROI := calculatePortfolioStats(portfolio, monthlyRents, properties)

	for i, prop := range portfolio {
		var roi float64
		for j, p := range properties {
			if p.Name == prop.Name {
				roi = prop.CalculateROI(monthlyRents[j])
				break
			}
		}
		fmt.Printf("%d. %s: %s (ROI: %.1f%%)\n",
			i+1, prop.Name, formatPrice(prop.Price), roi)
	}

	remaining := portfolioBudget - totalInvested
	fmt.Printf("\nTotal Invested: %s\n", formatPrice(totalInvested))
	fmt.Printf("Remaining Budget: %s\n", formatPrice(remaining))
	fmt.Printf("Portfolio Average ROI: %.1f%%\n", avgROI)
}
