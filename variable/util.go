package variable

import (
	"fmt"

	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
)

type DashboardConstant string

func (d DashboardConstant) AsVar() string {
	return fmt.Sprintf("$%s", d.String())
}

func (d DashboardConstant) String() string {
	return string(d)
}

func (d DashboardConstant) AsRef() dashboard.DataSourceRef {
	dvar := d.AsVar()
	return dashboard.DataSourceRef{Uid: &dvar}
}

type Pixel uint32

func (p Pixel) String() string {
	return fmt.Sprintf("%dpx", p)
}

type DashboardVariable string

func (d DashboardVariable) AsVar() string {
	return fmt.Sprintf("$%s", d)
}
