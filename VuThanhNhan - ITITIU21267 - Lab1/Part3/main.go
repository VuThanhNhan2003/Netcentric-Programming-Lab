package main

import (
	"fmt"
)

func main() {
	// Sample properties
	properties := []Property{
		{"Saigon Apartment", 2500000000, 75.5, 2, "District 1"},
		{"HCMC House", 4200000000, 120.0, 3, "District 7"},
		{"Budget Studio", 800000000, 35.0, 1, "Binh Thanh"},
	}

	// Monthly rents for each property
	monthlyRents := []float64{25000000, 35000000, 12000000}

	// TASK 3.1: Investment Analysis
	fmt.Println("=== Investment Analysis ===")
	for i, prop := range properties {
		roi := prop.CalculateROI(monthlyRents[i])
		grade := prop.InvestmentGradeWithRent(monthlyRents[i])
		fmt.Printf("%s: ROI %.1f%% per year - %s\n", prop.Name, roi, grade)
	}

	// Find best investment
	bestProp, bestROI := findBestInvestment(properties, monthlyRents)
	fmt.Printf("\nBest Investment: %s (%.1f%% ROI)\n", bestProp.Name, bestROI)

	// TASK 3.2: Loan Analysis
	fmt.Println("\n=== Loan Analysis ===")
	downPayment := 20.0 // 20%
	interestRate := 8.5 // 8.5%
	loanYears := 20     // 20 years

	for _, prop := range properties {
		loanInfo := prop.CalculateLoan(downPayment, interestRate, loanYears)

		fmt.Printf("%s:\n", prop.Name)
		fmt.Printf("  Loan Amount: %s (%.0f%% of price)\n",
			formatPrice(loanInfo.LoanAmount), 100-downPayment)
		fmt.Printf("  Monthly Payment: %s\n", formatPrice(loanInfo.MonthlyPayment))
		fmt.Printf("  Total Interest: %s over %d years\n\n",
			formatPrice(loanInfo.TotalInterest), loanYears)
	}
}
