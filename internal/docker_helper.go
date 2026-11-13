package internal

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"os"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

const (
	PostgresImage = "postgres:latest"
)

type DockerHelper struct {
	client *client.Client
}

func NewDockerHandler() (*DockerHelper, error) {
	dockerClient, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	fmt.Println("host is ->", dockerClient.DaemonHost())

	return &DockerHelper{
		client: dockerClient,
	}, nil
}

type ContainerCreateRequest struct {
	Port          string
	User          string
	Password      string
	Name          string
	Host          string
	ContainerName string
	Image         string
	Envs          []string
}

func (d *DockerHelper) CreateDBContainer(config ContainerCreateRequest) (func() error, error) {
	_, err := d.client.Ping(context.Background(), client.PingOptions{})
	if err != nil {
		return nil, fmt.Errorf("ping docker failed with: %w", err)
	}

	filterBy := make(client.Filters).Add("name", config.Image)
	result, err := d.client.ImageList(context.Background(), client.ImageListOptions{Filters: filterBy})

	if err != nil || len(result.Items) == 0 {
		slog.Info(fmt.Sprintf("image %s not found... pulling...", config.Image))
		reader, err := d.client.ImagePull(context.Background(), config.Image, client.ImagePullOptions{})
		if err != nil || reader == nil {
			if reader != nil {
				err := reader.Close()
				if err != nil {
					return nil, fmt.Errorf("while pulling dbImage (reader): %w of %s", err, config.Image)
				}
			}
			return nil, fmt.Errorf("while pulling dbImage: %w of %s", err, config.Image)
		}
		_, err = io.Copy(os.Stdout, reader)
		_ = reader.Close()
		if err != nil {
			return nil, fmt.Errorf("while handling dbImage: %w of %s", err, config.Name)
		}
	}

	postgresPort, err := network.ParsePort("5432/tcp")
	if err != nil {
		return nil, fmt.Errorf("parsing port: %w", err)
	}

	response, err := d.client.ContainerCreate(context.Background(), client.ContainerCreateOptions{
		Config: &container.Config{
			ExposedPorts: network.PortSet{
				postgresPort: struct{}{},
			},
			Image: config.Image,
			Env: []string{
				fmt.Sprintf("POSTGRES_USER=%s", config.User),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Password),
				fmt.Sprintf("POSTGRES_DB=%s", config.Name),
			},
		},
		HostConfig: &container.HostConfig{
			PortBindings: network.PortMap{
				postgresPort: []network.PortBinding{{
					HostIP:   netip.MustParseAddr("0.0.0.0"),
					HostPort: config.Port,
				}},
			},
		},
		Name: config.ContainerName,
	})

	if err != nil {
		return nil, fmt.Errorf("during container creation: %w", err)
	}

	cleanup := func() error {
		_, err := d.client.ContainerStop(context.Background(), response.ID, client.ContainerStopOptions{})
		if err != nil {
			return fmt.Errorf("during container stop: %w", err)
		}

		_, err = d.client.ContainerRemove(context.Background(), response.ID, client.ContainerRemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: true})
		if err != nil {
			return fmt.Errorf("during container removal: %w", err)
		}
		return nil
	}

	if _, err := d.client.ContainerStart(context.Background(), response.ID, client.ContainerStartOptions{}); err != nil {
		return cleanup, fmt.Errorf("during container startup: %w", err)
	}

	return cleanup, nil
}

func (d *DockerHelper) CloseDockerClient() error {
	if d.client == nil {
		return fmt.Errorf("docker client is nil")
	}

	err := d.client.Close()
	if err != nil {
		return fmt.Errorf("while closing docker client: %s", err.Error())
	}
	return nil
}
