package main

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
