package internal

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	PostgresImage = "postgres:latest"
)

type DockerHelper struct {
	client *client.Client
}

func NewDockerHandler() (*DockerHelper, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("host is -> %s", dockerClient.DaemonHost()))

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
	_, err := d.client.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ping docker failed with: %w", err)
	}

	filterBy := filters.NewArgs()
	filterBy.Add("name", config.Image)
	image, err := d.client.ImageList(context.Background(), imagetypes.ListOptions{Filters: filterBy})

	if err != nil || image == nil {
		slog.Info(fmt.Sprintf("image %s not found... pulling...", config.Image))
		reader, err := d.client.ImagePull(context.Background(), config.Image, imagetypes.PullOptions{})
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

	response, err := d.client.ContainerCreate(context.Background(),
		&container.Config{
			ExposedPorts: map[nat.Port]struct{}{
				"5432" + "/tcp": {},
			},
			Image: config.Image,
			Env: []string{
				fmt.Sprintf("POSTGRES_USER=%s", config.User),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Password),
				fmt.Sprintf("POSTGRES_DB=%s", config.Name),
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"5432" + "/tcp": []nat.PortBinding{{
					HostIP:   "0.0.0.0",
					HostPort: config.Port,
				}},
			},
		},
		nil,
		nil,
		config.ContainerName)

	if err != nil {
		return nil, fmt.Errorf("during container creation: %w", err)
	}

	cleanup := func() error {
		err := d.client.ContainerStop(context.Background(), response.ID, container.StopOptions{})
		if err != nil {
			return fmt.Errorf("during container stop: %w", err)
		}

		err = d.client.ContainerRemove(context.Background(), response.ID, container.RemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: true})
		if err != nil {
			return fmt.Errorf("during container removal: %w", err)
		}
		return nil
	}

	if err := d.client.ContainerStart(context.Background(), response.ID, container.StartOptions{}); err != nil {
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
