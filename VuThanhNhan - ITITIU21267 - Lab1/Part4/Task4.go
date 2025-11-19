package main

import (
	"fmt"
	"math"
	"sort"
)

type Property struct {
	Name     string
	Price    float64
	Area     float64
	Bedrooms int
	District string
}

// Basic methods
func (p Property) PricePerM2() float64 {
	if p.Area == 0 {
		return 0
	}
	return p.Price / p.Area
}

func (p Property) IsAffordable(budget float64) bool {
	return p.Price <= budget
}

func formatPrice(price float64) string {
	if price >= 1000000000 {
		billions := price / 1000000000
		return fmt.Sprintf("%.1f tỷ VND", billions)
	}
	millions := price / 1000000
	return fmt.Sprintf("%.0f triệu VND", millions)
}

// ROI calculation and investment grade methods
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

// Loan calculation structs and methods
type LoanInfo struct {
	LoanAmount     float64
	MonthlyPayment float64
	TotalInterest  float64
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

// --- Part 4: Control Flow and Logic ---

// Basic Recommendation Function
func recommendProperty(p Property, budget, maxMonthlyPayment float64) string {
	if !p.IsAffordable(budget) {
		return "❌ SKIP - Over budget"
	}

	loanInfo := p.CalculateLoan(20, 8.5, 20)
	if loanInfo.MonthlyPayment > maxMonthlyPayment {
		return "⚠️ CONSIDER - High monthly payment"
	}

	roi := p.CalculateROI(p.Price * 0.012) // Assume 1.2% monthly rent
	if roi > 10 {
		return "🔥 BUY NOW - Excellent ROI"
	} else if roi > 6 {
		return "✅ GOOD BUY - Solid investment"
	}
	return "🤔 MAYBE - Average investment"
}

// Task 4.1: Enhanced Smart Recommendation Function
func smartRecommendProperty(p Property, budget, maxMonthlyPayment float64) (string, string) {
	var bonus []string
	var warnings []string

	// Basic affordability check
	if !p.IsAffordable(budget) {
		return "❌ SKIP - Over budget", "Cannot afford this property"
	}

	// Check monthly payment
	loanInfo := p.CalculateLoan(20, 8.5, 20)
	if loanInfo.MonthlyPayment > maxMonthlyPayment {
		warnings = append(warnings, "High monthly payment")
	}

	// Location premium check
	premiumDistricts := []string{"District 1", "District 2", "District 7"}
	for _, district := range premiumDistricts {
		if p.District == district {
			bonus = append(bonus, "Premium location")
			break
		}
	}

	// Size efficiency check (50-100 m²)
	if p.Area >= 50 && p.Area <= 100 {
		bonus = append(bonus, "Optimal size")
	} else if p.Area < 50 {
		warnings = append(warnings, "Small size")
	} else {
		warnings = append(warnings, "Large size")
	}

	// Price per m² reasonableness
	pricePerM2 := p.PricePerM2()
	if pricePerM2 > 60000000 {
		warnings = append(warnings, "High price per m²")
	} else if pricePerM2 < 25000000 {
		bonus = append(bonus, "Good value per m²")
	}

	// Calculate ROI
	roi := p.CalculateROI(p.Price * 0.012)
	if roi > 10 {
		bonus = append(bonus, "Excellent ROI")
	} else if roi > 6 {
		bonus = append(bonus, "Good ROI")
	}

	// Determine final recommendation
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

	// Format details
	details := fmt.Sprintf("Bonuses: %d %v | Warnings: %d %v",
		bonusCount, bonus, warningCount, warnings)

	return recommendation, details
}

// Task 4.2: PropertyWithROI struct for sorting
type PropertyWithROI struct {
	Property Property
	ROI      float64
}

// Optimize Portfolio function
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

	// Sort by ROI descending
	sort.Slice(propsWithROI, func(i, j int) bool {
		return propsWithROI[i].ROI > propsWithROI[j].ROI
	})

	// Greedy add properties while budget allows
	for _, item := range propsWithROI {
		if item.Property.Price <= remainingBudget {
			portfolio = append(portfolio, item.Property)
			remainingBudget -= item.Property.Price
		}
	}

	return portfolio
}

// Calculate portfolio stats
func calculatePortfolioStats(portfolio []Property, monthlyRents []float64, properties []Property) (float64, float64) {
	totalInvested := 0.0
	totalROI := 0.0

	for _, prop := range portfolio {
		totalInvested += prop.Price

		// Find matching rent
		for i, p := range properties {
			if p.Name == prop.Name {
				totalROI += prop.CalculateROI(monthlyRents[i])
				break
			}
		}
	}

	avgROI := 0.0
	if len(portfolio) > 0 {
		avgROI = totalROI / float64(len(portfolio))
	}

	return totalInvested, avgROI
}
