package main

import (
	"fmt"
	"math"
)

// Property struct holds real estate info
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

// --- Task 3.1: Investment Calculator ---

// Part 1: Calculate ROI
func (p Property) CalculateROI(monthlyRent float64) float64 {
	annualRent := monthlyRent * 12
	roi := (annualRent / p.Price) * 100
	return roi
}

// Part 2: Investment Grade based on ROI
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

// Part 3: Find property with highest ROI
func findBestInvestment(properties []Property, rents []float64) (Property, float64) {
	bestProperty := properties[0]
	bestROI := properties[0].CalculateROI(rents[0])

	for i := 1; i < len(properties); i++ {
		roi := properties[i].CalculateROI(rents[i])
		if roi > bestROI {
			bestROI = roi
			bestProperty = properties[i]
		}
	}

	return bestProperty, bestROI
}

// --- Task 3.2: Loan Calculator ---

// Part 1: LoanInfo struct for loan details
type LoanInfo struct {
	LoanAmount     float64
	MonthlyPayment float64
	TotalInterest  float64
}

// Part 2: Calculate monthly mortgage payment
func calculateMonthlyPayment(loanAmount, annualRate float64, years int) float64 {
	monthlyRate := annualRate / 100 / 12
	numPayments := float64(years * 12)

	if annualRate == 0 {
		return loanAmount / numPayments
	}

	// Mortgage formula: M = P * [r(1+r)^n] / [(1+r)^n - 1]
	numerator := loanAmount * monthlyRate * math.Pow(1+monthlyRate, numPayments)
	denominator := math.Pow(1+monthlyRate, numPayments) - 1
	return numerator / denominator
}

// Part 3: Calculate full loan info for a property
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
