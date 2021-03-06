package convert

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	container "github.com/docker/docker/api/types/container"
	mounttypes "github.com/docker/docker/api/types/mount"
	types "github.com/docker/docker/api/types/swarm"
	swarmapi "github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/protobuf/ptypes"
)

func containerSpecFromGRPC(c *swarmapi.ContainerSpec) types.ContainerSpec {
	containerSpec := types.ContainerSpec{
		Image:     c.Image,
		Labels:    c.Labels,
		Command:   c.Command,
		Args:      c.Args,
		Hostname:  c.Hostname,
		Env:       c.Env,
		Dir:       c.Dir,
		User:      c.User,
		Groups:    c.Groups,
		TTY:       c.TTY,
		OpenStdin: c.OpenStdin,
		Hosts:     c.Hosts,
		Secrets:   secretReferencesFromGRPC(c.Secrets),
	}

	if c.DNSConfig != nil {
		containerSpec.DNSConfig = &types.DNSConfig{
			Nameservers: c.DNSConfig.Nameservers,
			Search:      c.DNSConfig.Search,
			Options:     c.DNSConfig.Options,
		}
	}

	// Mounts
	for _, m := range c.Mounts {
		mount := mounttypes.Mount{
			Target:   m.Target,
			Source:   m.Source,
			Type:     mounttypes.Type(strings.ToLower(swarmapi.Mount_MountType_name[int32(m.Type)])),
			ReadOnly: m.ReadOnly,
		}

		if m.BindOptions != nil {
			mount.BindOptions = &mounttypes.BindOptions{
				Propagation: mounttypes.Propagation(strings.ToLower(swarmapi.Mount_BindOptions_MountPropagation_name[int32(m.BindOptions.Propagation)])),
			}
		}

		if m.VolumeOptions != nil {
			mount.VolumeOptions = &mounttypes.VolumeOptions{
				NoCopy: m.VolumeOptions.NoCopy,
				Labels: m.VolumeOptions.Labels,
			}
			if m.VolumeOptions.DriverConfig != nil {
				mount.VolumeOptions.DriverConfig = &mounttypes.Driver{
					Name:    m.VolumeOptions.DriverConfig.Name,
					Options: m.VolumeOptions.DriverConfig.Options,
				}
			}
		}
		containerSpec.Mounts = append(containerSpec.Mounts, mount)
	}

	if c.StopGracePeriod != nil {
		grace, _ := ptypes.Duration(c.StopGracePeriod)
		containerSpec.StopGracePeriod = &grace
	}

	if c.Healthcheck != nil {
		containerSpec.Healthcheck = healthConfigFromGRPC(c.Healthcheck)
	}

	return containerSpec
}

func secretReferencesToGRPC(sr []*types.SecretReference) []*swarmapi.SecretReference {
	refs := make([]*swarmapi.SecretReference, 0, len(sr))
	for _, s := range sr {
		refs = append(refs, &swarmapi.SecretReference{
			SecretID:   s.SecretID,
			SecretName: s.SecretName,
			Target: &swarmapi.SecretReference_File{
				File: &swarmapi.SecretReference_FileTarget{
					Name: s.Target.Name,
					UID:  s.Target.UID,
					GID:  s.Target.GID,
					Mode: s.Target.Mode,
				},
			},
		})
	}

	return refs
}
func secretReferencesFromGRPC(sr []*swarmapi.SecretReference) []*types.SecretReference {
	refs := make([]*types.SecretReference, 0, len(sr))
	for _, s := range sr {
		target := s.GetFile()
		if target == nil {
			// not a file target
			logrus.Warnf("secret target not a file: secret=%s", s.SecretID)
			continue
		}
		refs = append(refs, &types.SecretReference{
			SecretID:   s.SecretID,
			SecretName: s.SecretName,
			Target: &types.SecretReferenceFileTarget{
				Name: target.Name,
				UID:  target.UID,
				GID:  target.GID,
				Mode: target.Mode,
			},
		})
	}

	return refs
}

func containerToGRPC(c types.ContainerSpec) (*swarmapi.ContainerSpec, error) {
	containerSpec := &swarmapi.ContainerSpec{
		Image:     c.Image,
		Labels:    c.Labels,
		Command:   c.Command,
		Args:      c.Args,
		Hostname:  c.Hostname,
		Env:       c.Env,
		Dir:       c.Dir,
		User:      c.User,
		Groups:    c.Groups,
		TTY:       c.TTY,
		OpenStdin: c.OpenStdin,
		Hosts:     c.Hosts,
		Secrets:   secretReferencesToGRPC(c.Secrets),
	}

	if c.DNSConfig != nil {
		containerSpec.DNSConfig = &swarmapi.ContainerSpec_DNSConfig{
			Nameservers: c.DNSConfig.Nameservers,
			Search:      c.DNSConfig.Search,
			Options:     c.DNSConfig.Options,
		}
	}

	if c.StopGracePeriod != nil {
		containerSpec.StopGracePeriod = ptypes.DurationProto(*c.StopGracePeriod)
	}

	// Mounts
	for _, m := range c.Mounts {
		mount := swarmapi.Mount{
			Target:   m.Target,
			Source:   m.Source,
			ReadOnly: m.ReadOnly,
		}

		if mountType, ok := swarmapi.Mount_MountType_value[strings.ToUpper(string(m.Type))]; ok {
			mount.Type = swarmapi.Mount_MountType(mountType)
		} else if string(m.Type) != "" {
			return nil, fmt.Errorf("invalid MountType: %q", m.Type)
		}

		if m.BindOptions != nil {
			if mountPropagation, ok := swarmapi.Mount_BindOptions_MountPropagation_value[strings.ToUpper(string(m.BindOptions.Propagation))]; ok {
				mount.BindOptions = &swarmapi.Mount_BindOptions{Propagation: swarmapi.Mount_BindOptions_MountPropagation(mountPropagation)}
			} else if string(m.BindOptions.Propagation) != "" {
				return nil, fmt.Errorf("invalid MountPropagation: %q", m.BindOptions.Propagation)

			}

		}

		if m.VolumeOptions != nil {
			mount.VolumeOptions = &swarmapi.Mount_VolumeOptions{
				NoCopy: m.VolumeOptions.NoCopy,
				Labels: m.VolumeOptions.Labels,
			}
			if m.VolumeOptions.DriverConfig != nil {
				mount.VolumeOptions.DriverConfig = &swarmapi.Driver{
					Name:    m.VolumeOptions.DriverConfig.Name,
					Options: m.VolumeOptions.DriverConfig.Options,
				}
			}
		}

		containerSpec.Mounts = append(containerSpec.Mounts, mount)
	}

	if c.Healthcheck != nil {
		containerSpec.Healthcheck = healthConfigToGRPC(c.Healthcheck)
	}

	return containerSpec, nil
}

func healthConfigFromGRPC(h *swarmapi.HealthConfig) *container.HealthConfig {
	interval, _ := ptypes.Duration(h.Interval)
	timeout, _ := ptypes.Duration(h.Timeout)
	return &container.HealthConfig{
		Test:     h.Test,
		Interval: interval,
		Timeout:  timeout,
		Retries:  int(h.Retries),
	}
}

func healthConfigToGRPC(h *container.HealthConfig) *swarmapi.HealthConfig {
	return &swarmapi.HealthConfig{
		Test:     h.Test,
		Interval: ptypes.DurationProto(h.Interval),
		Timeout:  ptypes.DurationProto(h.Timeout),
		Retries:  int32(h.Retries),
	}
}
