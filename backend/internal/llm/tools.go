package llm

import "encoding/json"

// RecommendFertilizerName is the tool Claude calls to fetch catalog products
// (ARCH §7.2). Kept as a const so the worker can match on it and the backend
// callback can validate it without a magic string.
const RecommendFertilizerName = "recommend_fertilizer"

// recommendFertilizerSchema is the JSON Schema for recommend_fertilizer (ARCH
// §7.2). problem is a closed enum so Claude maps a diagnosis onto a stable key
// that the catalog query (problems @> ARRAY[...]) can match.
const recommendFertilizerSchema = `{
  "type": "object",
  "properties": {
    "problem":  { "type": "string", "enum": ["leaf_yellowing","leaf_spots","wilting","stunted_growth","poor_fruiting","root_rot","pest_aphid","pest_mite","nitrogen_deficiency","phosphorus_deficiency","potassium_deficiency","calcium_deficiency","magnesium_deficiency","iron_deficiency","general_stress"] },
    "plant":    { "type": "string", "description": "tomato, cucumber, pepper, ... or omit if unknown" },
    "severity": { "type": "string", "enum": ["mild","moderate","severe"] },
    "notes":    { "type": "string" }
  },
  "required": ["problem"]
}`

// FertilizerTools returns the tool set sent to Claude on every chat turn. The
// worker marks the last tool's definition with cache_control (ARCH §7.3), so
// adding tools here stays cache-friendly.
func FertilizerTools() []Tool {
	return []Tool{
		{
			Name:        RecommendFertilizerName,
			Description: "Подбирает 1-3 удобрения из каталога компании, подходящих к проблеме растения.",
			InputSchema: json.RawMessage(recommendFertilizerSchema),
		},
	}
}
