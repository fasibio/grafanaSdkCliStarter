# GrafanaSdkCliStarter

**GrafanaSdkCliStarter** is a CLI helper for working with Go-written dashboards using the [grafana-foundation-sdk](https://github.com/grafana/grafana-foundation-sdk).

## Features

- Uses [urfave/cli](https://github.com/urfave/cli/) to enable automatic deployment and destruction of dashboards.
- Easily extendable to add more commands.
- Provides development commands to start a local Grafana and Prometheus instance for testing new metrics.

## Development Commands

- `go run . dev init`

  - Generates a `prometheus.yml` file where you can attach your application (via your computer's IP address).

- `go run . dev run`

  - Starts Grafana and Prometheus using [testcontainers](https://github.com/testcontainers/testcontainers-go).

## Example Usage

Here’s an example `main.go` implementation:

```go
package main

import (
	"os"

	g "github.com/fasibio/grafanaSdkCliStarter"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/urfave/cli/v2"
)

func main() {
	defaultDataSourceName := "datasource"

	app, err := g.NewCli("your-app-name",
		g.DashboardBuilder(
			func(folderName string, c *cli.Context) ([]dashboard.Dashboard, error) {
        
				o := NewGrafanaFoundationSDKDashboard("some-datasource")
				d, err := o.Build(string(folderName), "overview")
				return []dashboard.Dashboard{d}, err // attach here your builded Dashboards 
			},
		),
		g.DefaultDashboardCliFlagValue(g.CliServer, "your-grafana-server"),
		g.DefaultDashboardCliFlagValue(g.CliFolderName, "test_cli_starter"),
		g.DefaultDevRunDataSource(defaultDataSourceName),
	)
	if err != nil {
		panic(err)
	}
	if err := app.Run(os.Args); err != nil {
		panic("Error: " + err.Error())
	}
}
```

## Building Dashboards
How to build Dashboards you can find at [Grafana Foundation SDK Examples](https://github.com/grafana/grafana-foundation-sdk/tree/main/examples/go)

## Contributing

Feel free to fork this repository and submit pull requests for improvements.

## License

This project is licensed under the MIT License.

