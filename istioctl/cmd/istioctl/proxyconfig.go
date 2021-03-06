// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"

	"istio.io/istio/istioctl/pkg/kubernetes"
	"istio.io/istio/istioctl/pkg/writer/envoy/configdump"
	"istio.io/istio/pilot/pkg/model"
)

const (
	jsonOutput    = "json"
	summaryOutput = "short"
)

var (
	configCmd = &cobra.Command{
		Use:   "proxy-config",
		Short: "Retrieve information about proxy configuration from Envoy [kube only]",
		Long:  `A group of commands used to retrieve information about proxy configuration from the Envoy config dump`,
		Example: `  # Retrieve information about proxy configuration from an Envoy instance.
  istioctl proxy-config <clusters|listeners|routes|bootstap> <pod-name>`,
		Aliases: []string{"pc"},
	}

	fqdn, direction, subset string
	port                    int

	clusterConfigCmd = &cobra.Command{
		Use:   "cluster <pod-name>",
		Short: "Retrieves cluster configuration for the Envoy in the specified pod",
		Long:  `Retrieve information about cluster configuration for the Envoy instance in the specified pod.`,
		Example: `  # Retrieve summary about cluster configuration for a given pod from Envoy.
  istioctl proxy-config clusters <pod-name>

  # Retrieve cluster summary for clusters with port 9080.
  istioctl proxy-config clusters <pod-name> --port 9080

  # Retrieve full cluster dump for clusters that are inbound with a FQDN of details.default.svc.cluster.local.
  istioctl proxy-config clusters <pod-name> --fqdn details.default.svc.cluster.local --direction inbound -o json
`,
		Aliases: []string{"clusters", "c"},
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			podName := args[0]
			ns := handleNamespace()
			configWriter, err := setupEnvoyConfigWriter(podName, ns)
			if err != nil {
				return err
			}
			filter := configdump.ClusterFilter{
				FQDN:      model.Hostname(fqdn),
				Port:      port,
				Subset:    subset,
				Direction: model.TrafficDirection(direction),
			}
			switch outputFormat {
			case summaryOutput:
				return configWriter.PrintClusterSummary(filter)
			case jsonOutput:
				return configWriter.PrintClusterDump(filter)
			default:
				return fmt.Errorf("output format %q not supported", outputFormat)
			}
		},
	}

	address, listenerType string

	listenerConfigCmd = &cobra.Command{
		Use:   "listener <pod-name>",
		Short: "Retrieves listener configuration for the Envoy in the specified pod",
		Long:  `Retrieve information about listener configuration for the Envoy instance in the specified pod.`,
		Example: `  # Retrieve summary about listener configuration for a given pod from Envoy.
  istioctl proxy-config listeners <pod-name>

  # Retrieve listener summary for listeners with port 9080.
  istioctl proxy-config listeners <pod-name> --port 9080

  # Retrieve full listener dump for HTTP listeners with a wildcard address (0.0.0.0).
  istioctl proxy-config listeners <pod-name> --type HTTP --address 0.0.0.0 -o json
`,
		Aliases: []string{"listeners", "l"},
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			podName := args[0]
			ns := handleNamespace()
			configWriter, err := setupEnvoyConfigWriter(podName, ns)
			if err != nil {
				return err
			}
			filter := configdump.ListenerFilter{
				Address: address,
				Port:    uint32(port),
				Type:    listenerType,
			}

			switch outputFormat {
			case summaryOutput:
				return configWriter.PrintListenerSummary(filter)
			case jsonOutput:
				return configWriter.PrintListenerDump(filter)
			default:
				return fmt.Errorf("output format %q not supported", outputFormat)
			}
		},
	}

	routeName string

	routeConfigCmd = &cobra.Command{
		Use:   "route <pod-name>",
		Short: "Retrieves route configuration for the Envoy in the specified pod",
		Long:  `Retrieve information about route configuration for the Envoy instance in the specified pod.`,
		Example: `  # Retrieve summary about route configuration for a given pod from Envoy.
  istioctl proxy-config routes <pod-name>

  # Retrieve route summary for route 9080.
  istioctl proxy-config route <pod-name> --name 9080

  # Retrieve full route dump for route 9080
  istioctl proxy-config route <pod-name> --name 9080 -o json
`,
		Aliases: []string{"routes", "r"},
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			podName := args[0]
			ns := handleNamespace()
			configWriter, err := setupEnvoyConfigWriter(podName, ns)
			if err != nil {
				return err
			}
			filter := configdump.RouteFilter{
				Name: routeName,
			}
			switch outputFormat {
			case summaryOutput:
				return configWriter.PrintRouteSummary(filter)
			case jsonOutput:
				return configWriter.PrintRouteDump(filter)
			default:
				return fmt.Errorf("output format %q not supported", outputFormat)
			}
		},
	}

	bootstrapConfigCmd = &cobra.Command{
		Use:   "bootstrap <pod-name>",
		Short: "Retrieves bootstrap configuration for the Envoy in the specified pod",
		Long:  `Retrieve information about bootstrap configuration for the Envoy instance in the specified pod.`,
		Example: `  # Retrieve full bootstrap configuration for a given pod from Envoy.
  istioctl proxy-config bootstrap <pod-name>
`,
		Aliases: []string{"b"},
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			podName := args[0]
			ns := handleNamespace()
			configWriter, err := setupEnvoyConfigWriter(podName, ns)
			if err != nil {
				return err
			}
			return configWriter.PrintBootstrapDump()
		},
	}
)

func handleNamespace() string {
	ns := namespace
	if ns == v1.NamespaceAll {
		ns = defaultNamespace
	}
	return ns
}

func setupEnvoyConfigWriter(podName, podNamespace string) (*configdump.ConfigWriter, error) {
	kubeClient, err := kubernetes.NewClient(kubeconfig, configContext)
	if err != nil {
		return nil, err
	}
	path := "config_dump"
	debug, err := kubeClient.EnvoyDo(podName, podNamespace, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	cw := &configdump.ConfigWriter{Stdout: os.Stdout}
	err = cw.Prime(debug)
	if err != nil {
		return nil, err
	}
	return cw, nil
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", summaryOutput, "Output format: one of json|short")

	clusterConfigCmd.PersistentFlags().StringVar(&fqdn, "fqdn", "", "Filter clusters by substring of Service FQDN field")
	clusterConfigCmd.PersistentFlags().StringVar(&direction, "direction", "", "Filter clusters by Direction field")
	clusterConfigCmd.PersistentFlags().StringVar(&subset, "subset", "", "Filter clusters by subtring of Subset field")
	clusterConfigCmd.PersistentFlags().IntVar(&port, "port", 0, "Filter clusters by Port field")

	listenerConfigCmd.PersistentFlags().StringVar(&address, "address", "", "Filter listeners by address field")
	listenerConfigCmd.PersistentFlags().StringVar(&listenerType, "type", "", "Filter listeners by type field")
	listenerConfigCmd.PersistentFlags().IntVar(&port, "port", 0, "Filter listeners by Port field")

	routeConfigCmd.PersistentFlags().StringVar(&routeName, "name", "", "Filter listeners by route name field")

	configCmd.AddCommand(clusterConfigCmd, listenerConfigCmd, routeConfigCmd, bootstrapConfigCmd)
}
