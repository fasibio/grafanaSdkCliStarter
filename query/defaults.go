package query

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
)

func PrometheusQuery(query string, legend string) *prometheus.DataqueryBuilder {
	return prometheus.NewDataqueryBuilder().
		Expr(query).
		LegendFormat(legend)
}

func TablePrometheusQuery(query string, refID string) *prometheus.DataqueryBuilder {
	return prometheus.NewDataqueryBuilder().
		Expr(query).
		Format(prometheus.PromQueryFormatTable).
		RefId(refID)
}

func QueryVariable(name string, label string, query string, datasource dashboard.DataSourceRef, all, allSelected bool) *dashboard.QueryVariableBuilder {
	res := dashboard.NewQueryVariableBuilder(name).
		Label(label).
		Query(dashboard.StringOrMap{String: cog.ToPtr[string](query)}).
		Datasource(datasource)
	if all {
		res = res.Current(dashboard.VariableOption{
			Selected: cog.ToPtr[bool](allSelected),
			Text:     dashboard.StringOrArrayOfString{ArrayOfString: []string{"All"}},
			Value:    dashboard.StringOrArrayOfString{ArrayOfString: []string{"$__all"}},
		})
	}
	res = res.
		Refresh(dashboard.VariableRefreshOnTimeRangeChanged).
		Sort(dashboard.VariableSortAlphabeticalAsc).
		Multi(true).
		IncludeAll(true)
	return res
}
