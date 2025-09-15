package grafanasdkclistarter

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path"
	"slices"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	testContainerNetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/grafana/grafana-foundation-sdk/go/cog/plugins"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"

	"github.com/urfave/cli/v3"
)

type CliValues = string

const (
	CliServer            CliValues = "server"
	CliApiKey            CliValues = "apikey"
	CliApiBasePath       CliValues = "apibasepath"
	CliFolderName        CliValues = "foldername"
	CliYamlTargetFile    CliValues = "file"
	CliDevDatasourceName string    = "datasource_name"
	CliDevSubnet         string    = "subnet"
	CliDevGateway                  = "gateway"
)

//go:embed prometheus.yml.tmpl
var prometheusTmpl []byte

type Option func(runner *Runner, app *cli.Command) error

type Runner struct {
	cfg       *goapi.TransportConfig
	client    *goapi.GrafanaHTTPAPI
	Dashboard DashboardCreator
}

func NewCli(appName string, options ...Option) (*cli.Command, error) {
	plugins.RegisterDefaultPlugins()
	runner := Runner{}

	applyDestroyFlags := []cli.Flag{

		&cli.StringFlag{
			Name:    CliServer,
			Sources: cli.EnvVars(GetFlagEnvByFlagName(CliServer, appName)),
			Usage:   "grafana url",
		},
		&cli.StringFlag{
			Name:     CliApiKey,
			Sources:  cli.EnvVars(GetFlagEnvByFlagName(CliApiKey, appName)),
			Required: true,
			Usage:    "grafana api key",
		},
		&cli.StringFlag{
			Name:    CliApiBasePath,
			Sources: cli.EnvVars(GetFlagEnvByFlagName(CliApiBasePath, appName)),
			Value:   "/api",
			Usage:   "Base Path",
		},
	}

	app := &cli.Command{
		Usage: fmt.Sprintf("%s-grafana sdk cli", appName),
		Commands: []*cli.Command{
			{
				Name:  "dashboard",
				Usage: "To apply destroy and plan current dashboard",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    CliFolderName,
						Sources: cli.EnvVars(GetFlagEnvByFlagName(CliFolderName, appName)),
						Usage:   "GrafanaFolder to create dashboards",
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "apply",
						Before: runner.Before,
						Action: runner.Apply,
						Usage:  "Upload Dashboard to target configuration",
						Flags:  applyDestroyFlags,
					},
					{
						Name:   "destroy",
						Action: runner.Destroy,
						Before: runner.Before,
						Usage:  "Remove Dashboard from target configuration",
						Flags:  applyDestroyFlags,
					},
					{
						Name:   "plan",
						Action: runner.Plan,
						Usage:  "Upload Dashboard to target configuration",
					},
				},
			},
			{
				Name:   "dev",
				Before: runner.BeforeDev,
				Commands: []*cli.Command{
					{
						Name:   "init",
						Usage:  "Generate template prometheus folder/file to configure scrape stuff for local dev server (DO NOT move this files and start dev server from same path)",
						Action: runner.InitDev,
					},
					{
						Name:  "run",
						Usage: "Start DEV prometheus and grafana",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     CliDevDatasourceName,
								Sources:  cli.EnvVars(GetFlagEnvByFlagName(CliDevDatasourceName, appName)),
								Aliases:  []string{"datasource"},
								Required: true,
							},
							&cli.StringFlag{
								Name:    CliDevSubnet,
								Sources: cli.EnvVars(GetFlagEnvByFlagName(CliDevSubnet, appName)),
								Value:   "192.168.192.0/20",
							},
							&cli.StringFlag{
								Name:    CliDevGateway,
								Sources: cli.EnvVars(GetFlagEnvByFlagName(CliDevGateway, appName)),
								Value:   "192.168.192.1",
							},
						},
						Action: runner.startDev,
					},
				},
			},
		},
	}
	for _, o := range options {
		err := o(&runner, app)
		if err != nil {
			return nil, err
		}
	}
	return app, nil
}

func (r *Runner) BeforeDev(ctx context.Context, c *cli.Command) (context.Context, error) {
	return ctx, nil
}
func (r *Runner) Before(ctx context.Context, c *cli.Command) (context.Context, error) {
	p, err := url.Parse(c.String(CliServer))
	if err != nil {
		return ctx, fmt.Errorf("%s is not a valid url: %w", c.String(CliServer), err)
	}

	cfg := &goapi.TransportConfig{
		// Host is the doman name or IP address of the host that serves the API.
		Host: p.Host,
		// BasePath is the URL prefix for all API paths, relative to the host root.
		BasePath: c.String(CliApiBasePath),
		// Schemes are the transfer protocols used by the API (http or https).
		Schemes: []string{p.Scheme},
		APIKey:  c.String(CliApiKey),
	}
	client := goapi.NewHTTPClientWithConfig(strfmt.Default, cfg)
	r.cfg = cfg
	r.client = client

	return ctx, nil
}

func (r *Runner) Apply(ctx context.Context, c *cli.Command) error {
	foldername := c.String(CliFolderName)
	_, err := r.client.Folders.GetFolderByUID(foldername)
	if err != nil {
		_, err := r.client.Folders.CreateFolder(&models.CreateFolderCommand{
			UID:   foldername,
			Title: foldername,
		})
		if err != nil {
			return fmt.Errorf("apply: can not create folder %s: %w", foldername, err)
		}
	}

	dashboards, err := r.getDashboards(ctx, c)
	if err != nil {
		return fmt.Errorf("failed apply Dashboard %w", err)
	}
	for _, d := range dashboards {
		p, err := r.client.Dashboards.PostDashboard(&models.SaveDashboardCommand{
			Dashboard: d,
			FolderUID: foldername,
			Overwrite: true,
		})
		if err != nil {
			return fmt.Errorf("unable to post Dashboard: %w", err)
		}
		fmt.Printf("%s: %s://%s%s\n", *d.Title, r.cfg.Schemes[0], r.cfg.Host, *p.Payload.URL)
	}

	var sb strings.Builder
	sb.WriteString("Dashboard")
	if len(dashboards) > 1 {
		sb.WriteRune('s')
	}
	sb.WriteString(" created")
	fmt.Println(sb.String())
	return nil
}

func (r *Runner) Plan(ctx context.Context, c *cli.Command) error {
	dashboards, err := r.getDashboards(ctx, c)
	if err != nil {
		return fmt.Errorf("failed plan %w ", err)
	}
	for _, d := range dashboards {
		b, err := json.MarshalIndent(d, " ", "    ")
		if err != nil {
			return fmt.Errorf("unable to marshal: %w´", err)
		}
		fmt.Println(string(b))
	}
	return nil
}
func (r *Runner) Destroy(ctx context.Context, c *cli.Command) error {
	dashboards, err := r.getDashboards(ctx, c)
	if err != nil {
		return fmt.Errorf("failed destroy: %w", err)
	}

	errList := errors.Join(nil)
	for _, d := range dashboards {

		_, err := r.client.Dashboards.DeleteDashboardByUID(*d.Uid)
		if err != nil {
			errList = errors.Join(errList, err)
		}

	}

	if errList != nil {
		return errList
	}

	fmt.Println("Destroyed")
	return nil
}

func (r *Runner) getDashboards(ctx context.Context, c *cli.Command) ([]dashboard.Dashboard, error) {
	foldername := c.String(CliFolderName)
	dashboards, err := r.Dashboard(foldername, c)
	if err != nil {
		return nil, fmt.Errorf("failed get Dashboard %w", err)
	}
	return dashboards, nil
}

func (r *Runner) InitDev(ctx context.Context, c *cli.Command) error {
	err := EnsureDir("./prometheus")
	if err != nil {
		return err
	}
	if !DirExist("./prometheus/prometheus.yml") {
		err = os.WriteFile("./prometheus/prometheus.yml", prometheusTmpl, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}
func DefaultDevRunDataSource(value string) Option {
	return func(runner *Runner, app *cli.Command) error {
		for _, c := range app.Commands {
			if c.Name == "dev" {
				for _, sc := range c.Commands {
					if sc.Name == "run" {
						for _, f := range sc.Flags {
							if slices.Contains(f.Names(), CliDevDatasourceName) {
								strFlag, ok := f.(*cli.StringFlag)
								if !ok {
									return fmt.Errorf("Oh shit something big goes wrong")
								}
								strFlag.Value = value
								strFlag.Required = false
							}
						}
					}
				}

			}
		}

		return nil
	}
}

func (r *Runner) startDev(ctx context.Context, c *cli.Command) error {
	err := r.InitDev(ctx, c)
	if err != nil {
		return err
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	newNetwork, err := testContainerNetwork.New(ctx,
		testContainerNetwork.WithIPAM(&network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet:  c.String(CliDevSubnet),
					Gateway: c.String(CliDevGateway),
				},
			},
		}),
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := newNetwork.Remove(ctx); err != nil {
			panic(err)
		}
	}()
	prometheusContainerName := "prometheus_" + uuid.New().String()
	prometheusPort := "9090/tcp"
	req := testcontainers.ContainerRequest{
		Name:         prometheusContainerName,
		Image:        "prom/prometheus:latest",
		ExposedPorts: []string{prometheusPort},
		Cmd: []string{
			"--config.file=/etc/prometheus/prometheus.yml",
			"--storage.tsdb.path=/prometheus",
			"--web.console.libraries=/usr/share/prometheus/console_libraries",
			"--web.console.templates=/usr/share/prometheus/consoles",
			"--web.enable-lifecycle",
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Mounts = append(hc.Mounts, mount.Mount{Source: path.Join(pwd, "prometheus"), Target: "/etc/prometheus", Type: mount.TypeBind})
		},
		Privileged: true,
		Networks:   []string{newNetwork.Name},
		WaitingFor: wait.ForListeningPort(nat.Port(prometheusPort)),
	}
	prometheusC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("error start prometheus: %w", err)
	}
	defer func() {
		if err := prometheusC.Terminate(ctx); err != nil {
			panic(err)
		}
	}()
	grafanaPort := "3000/tcp"

	req2 := testcontainers.ContainerRequest{
		Image:        "grafana/grafana:latest",
		ExposedPorts: []string{grafanaPort},
		Networks:     []string{newNetwork.Name},
		WaitingFor:   wait.ForListeningPort(nat.Port(grafanaPort)),
	}
	grafanaC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req2,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("error start Grafana %w", err)
	}
	defer func() {
		if err := grafanaC.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	grafanaRealPort, err := grafanaC.MappedPort(ctx, nat.Port(grafanaPort))
	if err != nil {
		return fmt.Errorf("unable to get mapped grafana port: %w", err)
	}

	grafanaUrl, err := grafanaC.PortEndpoint(ctx, nat.Port(grafanaPort), "http")
	if err != nil {
		return fmt.Errorf("error get grafana endpoint: %w", err)
	}
	prometheusUrl, err := prometheusC.PortEndpoint(ctx, nat.Port(prometheusPort), "http")
	if err != nil {
		return fmt.Errorf("error get prometheus endpoint: %w", err)
	}
	cfg := &goapi.TransportConfig{
		// Host is the doman name or IP address of the host that serves the API.
		Host: fmt.Sprintf("localhost:%s", grafanaRealPort.Port()),
		// BasePath is the URL prefix for all API paths, relative to the host root.
		BasePath: "/api",
		// Schemes are the transfer protocols used by the API (http or https).
		Schemes:   []string{"http"},
		BasicAuth: url.UserPassword("admin", "admin"),
	}
	client := goapi.NewHTTPClientWithConfig(strfmt.Default, cfg)
	// prometheusDatasource, err := prometheus.New(c.String(CliDevDatasourceName), fmt.Sprintf("http://%s:9090", prometheusContainerName))
	// if err != nil {
	// 	return err
	// }
	_, err = client.Datasources.AddDataSource(&models.AddDataSourceCommand{
		Name:   c.String(CliDevDatasourceName),
		URL:    fmt.Sprintf("http://%s:9090", prometheusContainerName),
		UID:    c.String(CliDevDatasourceName),
		Type:   "prometheus",
		Access: "proxy",
	})
	if err != nil {
		return fmt.Errorf("error create prometheus datasource at grafana: %w", err)
	}

	grabanaClient := NewGrafanaAddOn(grafanaUrl, "admin", "admin")

	apiKey, err := grabanaClient.CreateAPIKey("debug", "test")

	if err != nil {
		return fmt.Errorf("error create grafana apikey: %w", err)
	}

	fmt.Printf("Prometheus endpoint: %s \n", prometheusUrl)
	fmt.Printf("\tReload Config: curl -s -XPOST %s/-/reload\n ", prometheusUrl)
	fmt.Printf("Grafana endpoint: %s \n", grafanaUrl)
	fmt.Printf("\tGrafana user: admin \n")
	fmt.Printf("\tGrafana password: admin \n")
	fmt.Printf("\tPrometheus Datasourcename: %s\n", c.String(CliDevDatasourceName))
	fmt.Printf("\tApi key: %s \n", apiKey)
	fmt.Printf("Simple run\n go run . dashboard apply --server %s --apikey %s \n", grafanaUrl, apiKey)
	<-done
	return nil
}

type APIKeyReader struct{}

func (APIKeyReader) ReadResponse(cr runtime.ClientResponse, c runtime.Consumer) (interface{}, error) {

	buf := new(strings.Builder)
	_, err := io.Copy(buf, cr.Body())
	if err != nil {
		return nil, err
	}
	fmt.Println(buf.String())

	return nil, nil
}
