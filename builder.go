package grafanasdkclistarter

import (
	"fmt"
	"slices"

	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/urfave/cli/v3"
)

type DashboardCreator func(folderName string, c *cli.Command) ([]dashboard.Dashboard, error)

func DashboardBuilder(d DashboardCreator) Option {
	return func(runner *Runner, app *cli.Command) error {
		if runner.Dashboard != nil {
			return fmt.Errorf("Dashboard already set")
		}
		runner.Dashboard = d
		return nil
	}
}

func DefaultDashboardCliFlagValue(key CliValues, value string) Option {
	return func(runner *Runner, app *cli.Command) error {
		for _, c := range app.Commands {
			if c.Name == "dashboard" {
				for _, f := range c.Flags {
					if slices.Contains(f.Names(), key) {
						strFlag, ok := f.(*cli.StringFlag)
						if !ok {
							return fmt.Errorf("Oh shit something big goes wrong")
						}
						if strFlag.IsRequired() {
							strFlag.Required = false
						}
						strFlag.Value = value
					}
				}
				for _, s := range c.Commands {
					for _, f := range s.Flags {
						if slices.Contains(f.Names(), key) {
							strFlag, ok := f.(*cli.StringFlag)
							if !ok {
								return fmt.Errorf("Oh shit something big goes wrong")
							}
							if strFlag.IsRequired() {
								strFlag.Required = false
							}
							strFlag.Value = value
						}
					}
				}
			}
		}

		return nil
	}
}
