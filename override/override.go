package override

import "github.com/grafana/grafana-foundation-sdk/go/dashboard"

func ByQuery(ref string) dashboard.MatcherConfig {
	return matcherHelper("byFrameRefID", ref)
}

func matcherHelper(id, option string) dashboard.MatcherConfig {
	return dashboard.MatcherConfig{
		Id:      id,
		Options: option,
	}
}

func ByName(name string) dashboard.MatcherConfig {
	return matcherHelper("byName", name)
}

func ByRegex(regex string) dashboard.MatcherConfig {
	return matcherHelper("byRegexp", regex)
}

type MatcherConfigFieldType string

const (
	FieldTypeTime MatcherConfigFieldType = "time"
)

func ByType(fieldType MatcherConfigFieldType) dashboard.MatcherConfig {
	return matcherHelper("byType", string(fieldType))
}

func FixedColorScheme(color string) dashboard.DynamicConfigValue {
	return dashboard.DynamicConfigValue{
		Id: "color",
		Value: map[string]string{
			"fixedColor": color,
			"mode":       "fixed",
		},
	}
}

func Unit(unit string) dashboard.DynamicConfigValue {
	return dashboard.DynamicConfigValue{
		Id:    "unit",
		Value: unit,
	}
}

func FillOpacity(opacity int) dashboard.DynamicConfigValue {
	return dashboard.DynamicConfigValue{
		Id:    "custom.fillOpacity",
		Value: opacity,
	}
}

func NegativeY() dashboard.DynamicConfigValue {
	return dashboard.DynamicConfigValue{
		Id:    "custom.transform",
		Value: "negative-Y",
	}
}

type PlacementMode string

const (
	Hidden PlacementMode = "hidden"
	Auto   PlacementMode = "auto"
	Left   PlacementMode = "left"
	Right  PlacementMode = "right"
)

type StackMode string

const (
	// Unstacked will not stack series
	Unstacked StackMode = "none"
	// NormalStack will stack series as absolute numbers
	NormalStack StackMode = "normal"
	// PercentStack will stack series as percents
	PercentStack StackMode = "percent"
)

func Stack(mode StackMode) dashboard.DynamicConfigValue {
	return dashboard.DynamicConfigValue{
		Id: "custom.stacking",
		Value: map[string]interface{}{
			"group": false,
			"mode":  string(mode),
		},
	}
}

func AxisPlacement(placement PlacementMode) dashboard.DynamicConfigValue {
	return dashboard.DynamicConfigValue{
		Id:    "custom.axisPlacement",
		Value: string(placement),
	}
}
