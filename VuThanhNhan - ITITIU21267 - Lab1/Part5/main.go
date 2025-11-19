package main

import (
	"fmt"
)

func main() {
	// Property data
	properties := []Property{
		{"Saigon Apartment", 2500000000, 75.5, 2, "District 1"},
		{"HCMC House", 4200000000, 120.0, 3, "District 7"},
		{"Budget Studio", 800000000, 35.0, 1, "Binh Thanh"},
		{"Cozy Condo", 1800000000, 65.0, 2, "District 2"},
		{"Luxury Penthouse", 5500000000, 150.0, 3, "District 1"},
	}

	// Monthly rents for each property
	monthlyRents := []float64{25000000, 35000000, 12000000, 18000000, 45000000}

	// Main menu loop
	for {
		fmt.Println("\n=== Property Analyzer Menu ===")
		fmt.Println("1. View all properties")
		fmt.Println("2. Search by budget")
		fmt.Println("3. Investment analysis")
		fmt.Println("4. Loan calculator")
		fmt.Println("5. Get recommendations")
		fmt.Println("6. Optimize portfolio")
		fmt.Println("0. Exit")
		fmt.Print("\nChoose option: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			viewAllProperties(properties)

		case 2:
			searchByBudget(properties)

		case 3:
			investmentAnalysis(properties, monthlyRents)

		case 4:
			loanCalculator(properties)

		case 5:
			getRecommendations(properties, monthlyRents)

		case 6:
			optimizePortfolioMenu(properties, monthlyRents)

		case 0:
			fmt.Println("\n👋 Thank you for using Property Analyzer!")
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Println("\n❌ Invalid option! Please choose 0-6.")
		}

		fmt.Print("\nPress Enter to continue...")
		fmt.Scanln()
	}
}
