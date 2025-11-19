package main

import (
	"fmt"
	"math"
	"sort"
)

// Property struct
type Property struct {
	Name     string
	Price    float64
	Area     float64
	Bedrooms int
	District string
}

// LoanInfo struct
type LoanInfo struct {
	LoanAmount     float64
	MonthlyPayment float64
	TotalInterest  float64
}

// PropertyWithROI helper struct
type PropertyWithROI struct {
	Property Property
	ROI      float64
}

// ============= UTILITY FUNCTIONS =============

func formatPrice(price float64) string {
	if price >= 1000000000 {
		billions := price / 1000000000
		return fmt.Sprintf("%.1f tỷ VND", billions)
	}
	millions := price / 1000000
	return fmt.Sprintf("%.0f triệu VND", millions)
}

// ============= PROPERTY METHODS =============

func (p Property) PricePerM2() float64 {
	if p.Area == 0 {
		return 0
	}
	return p.Price / p.Area
}

func (p Property) IsAffordable(budget float64) bool {
	return p.Price <= budget
}

func (p Property) CalculateROI(monthlyRent float64) float64 {
	annualRent := monthlyRent * 12
	roi := (annualRent / p.Price) * 100
	return roi
}

func (p Property) InvestmentGradeWithRent(monthlyRent float64) string {
	roi := p.CalculateROI(monthlyRent)

	if roi > 8 {
		return "EXCELLENT"
	} else if roi >= 5 {
		return "GOOD"
	} else if roi >= 3 {
		return "FAIR"
	}
	return "POOR"
}

func calculateMonthlyPayment(loanAmount, annualRate float64, years int) float64 {
	monthlyRate := annualRate / 100 / 12
	numPayments := float64(years * 12)

	if annualRate == 0 {
		return loanAmount / numPayments
	}

	return loanAmount * monthlyRate * math.Pow(1+monthlyRate, numPayments) /
		(math.Pow(1+monthlyRate, numPayments) - 1)
}

func (p Property) CalculateLoan(downPaymentPercent, interestRate float64, years int) LoanInfo {
	downPayment := p.Price * (downPaymentPercent / 100)
	loanAmount := p.Price - downPayment

	monthlyPayment := calculateMonthlyPayment(loanAmount, interestRate, years)

	totalPaid := monthlyPayment * float64(years*12)
	totalInterest := totalPaid - loanAmount

	return LoanInfo{
		LoanAmount:     loanAmount,
		MonthlyPayment: monthlyPayment,
		TotalInterest:  totalInterest,
	}
}

// ============= MENU FUNCTIONS =============

func viewAllProperties(properties []Property) {
	fmt.Println("\n=== All Properties ===")
	for i, prop := range properties {
		fmt.Printf("%d. %s\n", i+1, prop.Name)
		fmt.Printf("   Price: %s | Area: %.1f m² | Bedrooms: %d\n",
			formatPrice(prop.Price), prop.Area, prop.Bedrooms)
		fmt.Printf("   District: %s | Price/m²: %s\n",
			prop.District, formatPrice(prop.PricePerM2()))
	}
}

func searchByBudget(properties []Property) {
	var budget float64
	fmt.Print("\nEnter your budget (VND): ")
	fmt.Scanln(&budget)

	fmt.Printf("\n=== Properties under %s ===\n", formatPrice(budget))
	found := false
	for _, prop := range properties {
		if prop.IsAffordable(budget) {
			fmt.Printf("✓ %s: %s\n", prop.Name, formatPrice(prop.Price))
			found = true
		}
	}

	if !found {
		fmt.Println("No properties found within your budget.")
	}
}

func investmentAnalysis(properties []Property, monthlyRents []float64) {
	fmt.Println("\n=== Investment Analysis ===")

	bestROI := 0.0
	var bestProp Property

	for i, prop := range properties {
		roi := prop.CalculateROI(monthlyRents[i])
		grade := prop.InvestmentGradeWithRent(monthlyRents[i])

		fmt.Printf("\n%s:\n", prop.Name)
		fmt.Printf("  Monthly Rent: %s\n", formatPrice(monthlyRents[i]))
		fmt.Printf("  ROI: %.1f%% per year - %s\n", roi, grade)

		if roi > bestROI {
			bestROI = roi
			bestProp = prop
		}
	}

	fmt.Printf("\n🏆 Best Investment: %s (%.1f%% ROI)\n", bestProp.Name, bestROI)
}

func loanCalculator(properties []Property) {
	fmt.Println("\n=== Loan Calculator ===")

	// Get loan parameters
	var downPayment, interestRate float64
	var years int

	fmt.Print("Down payment percentage (e.g., 20 for 20%): ")
	fmt.Scanln(&downPayment)
	fmt.Print("Annual interest rate (e.g., 8.5 for 8.5%): ")
	fmt.Scanln(&interestRate)
	fmt.Print("Loan term in years: ")
	fmt.Scanln(&years)

	fmt.Println("\n--- Loan Details ---")
	for _, prop := range properties {
		loanInfo := prop.CalculateLoan(downPayment, interestRate, years)

		fmt.Printf("\n%s:\n", prop.Name)
		fmt.Printf("  Property Price: %s\n", formatPrice(prop.Price))
		fmt.Printf("  Loan Amount: %s (%.0f%% of price)\n",
			formatPrice(loanInfo.LoanAmount), 100-downPayment)
		fmt.Printf("  Monthly Payment: %s\n", formatPrice(loanInfo.MonthlyPayment))
		fmt.Printf("  Total Interest: %s over %d years\n",
			formatPrice(loanInfo.TotalInterest), years)
	}
}

func getRecommendations(properties []Property, monthlyRents []float64) {
	var budget, maxMonthly float64

	fmt.Print("\nEnter your budget (VND): ")
	fmt.Scanln(&budget)
	fmt.Print("Enter max monthly payment you can afford (VND): ")
	fmt.Scanln(&maxMonthly)

	fmt.Println("\n=== Property Recommendations ===")

	for i, prop := range properties {
		recommendation, details := smartRecommendProperty(prop, budget, maxMonthly)

		fmt.Printf("\n%s (%.1f m², %s)\n", prop.Name, prop.Area, prop.District)
		fmt.Printf("Price: %s | Price/m²: %s\n",
			formatPrice(prop.Price), formatPrice(prop.PricePerM2()))
		fmt.Printf("Estimated Rent: %s/month\n", formatPrice(monthlyRents[i]))
		fmt.Printf("Recommendation: %s\n", recommendation)
		fmt.Printf("Details: %s\n", details)
	}
}

func smartRecommendProperty(p Property, budget, maxMonthlyPayment float64) (string, string) {
	var bonus []string
	var warnings []string

	if !p.IsAffordable(budget) {
		return "❌ SKIP - Over budget", "Cannot afford this property"
	}

	loanInfo := p.CalculateLoan(20, 8.5, 20)
	if loanInfo.MonthlyPayment > maxMonthlyPayment {
		warnings = append(warnings, "High monthly payment")
	}

	premiumDistricts := []string{"District 1", "District 2", "District 7"}
	for _, district := range premiumDistricts {
		if p.District == district {
			bonus = append(bonus, "Premium location")
			break
		}
	}

	if p.Area >= 50 && p.Area <= 100 {
		bonus = append(bonus, "Optimal size")
	} else if p.Area < 50 {
		warnings = append(warnings, "Small size")
	} else {
		warnings = append(warnings, "Large size")
	}

	pricePerM2 := p.PricePerM2()
	if pricePerM2 > 60000000 {
		warnings = append(warnings, "High price per m²")
	} else if pricePerM2 < 25000000 {
		bonus = append(bonus, "Good value per m²")
	}

	roi := p.CalculateROI(p.Price * 0.012)
	if roi > 10 {
		bonus = append(bonus, "Excellent ROI")
	} else if roi > 6 {
		bonus = append(bonus, "Good ROI")
	}

	recommendation := "🤔 NEUTRAL"
	bonusCount := len(bonus)
	warningCount := len(warnings)

	if bonusCount >= 3 && warningCount == 0 {
		recommendation = "🔥 BUY NOW - Highly recommended"
	} else if bonusCount >= 2 && warningCount <= 1 {
		recommendation = "✅ GOOD BUY - Recommended"
	} else if warningCount > bonusCount {
		recommendation = "⚠️ CONSIDER - Has concerns"
	} else if bonusCount > 0 {
		recommendation = "👍 OKAY - Worth considering"
	}

	details := fmt.Sprintf("Bonuses: %d %v | Warnings: %d %v",
		bonusCount, bonus, warningCount, warnings)

	return recommendation, details
}

func optimizePortfolioMenu(properties []Property, monthlyRents []float64) {
	var budget float64
	fmt.Print("\nEnter your total investment budget (VND): ")
	fmt.Scanln(&budget)

	fmt.Printf("\n=== Portfolio Optimization ===\n")
	fmt.Printf("Budget: %s\n\n", formatPrice(budget))

	portfolio := optimizePortfolio(properties, monthlyRents, budget)

	if len(portfolio) == 0 {
		fmt.Println("No properties can be purchased within your budget.")
		return
	}

	fmt.Println("Selected Properties:")
	totalInvested := 0.0
	totalROI := 0.0

	for i, prop := range portfolio {
		var roi float64
		for j, p := range properties {
			if p.Name == prop.Name {
				roi = prop.CalculateROI(monthlyRents[j])
				totalROI += roi
				break
			}
		}
		totalInvested += prop.Price
		fmt.Printf("%d. %s: %s (ROI: %.1f%%)\n",
			i+1, prop.Name, formatPrice(prop.Price), roi)
	}

	remaining := budget - totalInvested
	avgROI := totalROI / float64(len(portfolio))

	fmt.Printf("\nTotal Invested: %s\n", formatPrice(totalInvested))
	fmt.Printf("Remaining Budget: %s\n", formatPrice(remaining))
	fmt.Printf("Portfolio Average ROI: %.1f%%\n", avgROI)
}

func optimizePortfolio(properties []Property, monthlyRents []float64, totalBudget float64) []Property {
	var portfolio []Property
	remainingBudget := totalBudget

	var propsWithROI []PropertyWithROI
	for i, prop := range properties {
		roi := prop.CalculateROI(monthlyRents[i])
		propsWithROI = append(propsWithROI, PropertyWithROI{
			Property: prop,
			ROI:      roi,
		})
	}

	sort.Slice(propsWithROI, func(i, j int) bool {
		return propsWithROI[i].ROI > propsWithROI[j].ROI
	})

	for _, item := range propsWithROI {
		if item.Property.Price <= remainingBudget {
			portfolio = append(portfolio, item.Property)
			remainingBudget -= item.Property.Price
		}
	}

	return portfolio
}
