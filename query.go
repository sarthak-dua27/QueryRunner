package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Struct definitions

type Root struct {
	Bklctrcb struct {
		Geometry struct {
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
		Relationship string `json:"relationship"`
	} `json:"bklctrcb"`
}

type LocationQuery struct {
	Query struct {
		Location struct {
			Lon      float64 `json:"lon"`
			Lat      float64 `json:"lat"`
		} `json:"location"`
		Distance string  `json:"distance"`
		Field    string  `json:"field"`
	} `json:"query"`
}

type RelationshipQuery struct {
	Query struct {
		Match string `json:"match"`
		Field string `json:"field"`
	} `json:"query"`
}

type ConjunctQuery struct {
	Query struct {
		Conjuncts []interface{} `json:"conjuncts"`
	} `json:"query"`
}

// Function to generate queries
func makeQueries(locations []Root, n int) []interface{} {
	queries := make([]interface{}, 0, n*3) // Pre-allocate space for n queries of each type

	for i := 0; i < n; i++ {
		// Select random location for each iteration
		randomLoc := locations[rand.Intn(len(locations))]
		coords := randomLoc.Bklctrcb.Geometry.Coordinates
		relationship := randomLoc.Bklctrcb.Relationship

		// Generate location query
		locQuery := LocationQuery{}
		locQuery.Query.Location.Lon = coords[0] // Correct field access for Lon
		locQuery.Query.Location.Lat = coords[1] // Correct field access for Lat
		locQuery.Query.Distance = "100mi"
		locQuery.Query.Field = "bklctrcb.geometry.coordinates" // Correct field access for Field
		queries = append(queries, locQuery)

		// Generate relationship query
		relationshipQuery := RelationshipQuery{}
		relationshipQuery.Query.Match = relationship
		relationshipQuery.Query.Field = "bklctrcb.relationship"
		queries = append(queries, relationshipQuery)

		// Generate conjunct query
		conjunctQuery := ConjunctQuery{}
		conjunctQuery.Query.Conjuncts = []interface{}{
			map[string]interface{}{
				"location": map[string]interface{}{
					"lon": coords[0],
					"lat": coords[1],
				},
				"distance": "100mi",
				"field":    "bklctrcb.geometry.coordinates",
			},
			map[string]interface{}{
				"match": relationship,
				"field": "bklctrcb.relationship",
			},
		}
		queries = append(queries, conjunctQuery)
	}

	return queries
}

func GenerateQueries(n int) {
	// Read JSON file containing locations
	data, err := os.ReadFile("long-lat.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON into locations slice
	var locations []Root
	if err := json.Unmarshal(data, &locations); err != nil {
		panic(err)
	}

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Generate random queries
	queries := makeQueries(locations, n/3)

	// Marshal the queries into JSON format with indentation
	queryJSON, err := json.MarshalIndent(queries, "", "    ")
	if err != nil {
		panic(err)
	}

	// Write the JSON queries to a file
	err = os.WriteFile("queries.json", queryJSON, 0644)
	if err != nil {
		panic(err)
	}

	// Print success message
	fmt.Println("Queries saved to queries.json")
}
