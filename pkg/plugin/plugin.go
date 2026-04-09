package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gosuri/uitable"
	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/util"
)

const progressBarWidth = 10

func RunPlugin(configFlags *genericclioptions.ConfigFlags, cmd *cobra.Command) error {
	factory := util.NewFactory(configFlags)
	clientConfig := factory.ToRawKubeConfigLoader()
	config, err := factory.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return fmt.Errorf("failed getting namespace: %w", err)
	}

	if getFlagBool(cmd, "all-namespaces") {
		namespace = ""
	}

	quotas, err := getQuotas(clientSet, namespace)
	if err != nil {
		return fmt.Errorf("failed to list resource quotas: %w", err)
	}

	printResourceQuotas(quotas)
	return nil
}

func getFlagBool(cmd *cobra.Command, flag string) bool {
	b, err := cmd.Flags().GetBool(flag)
	if err != nil {
		return false
	}
	return b
}

func getQuotas(clientSet *kubernetes.Clientset, namespace string) (*v1.ResourceQuotaList, error) {
	return clientSet.CoreV1().ResourceQuotas(namespace).List(context.TODO(), metav1.ListOptions{})
}

func printResourceQuotas(list *v1.ResourceQuotaList) {
	cfmt.RegisterStyle("url", func(s string) string {
		return cfmt.Sprintf("{{%s}}::yellow|underline", s)
	})

	for i, quota := range list.Items {
		if i > 0 {
			fmt.Println()
		}

		table := uitable.New()
		table.Wrap = true
		table.MaxColWidth = 60

		table.AddRow(
			cfmt.Sprintf("{{Name:}}::white|bold"),
			cfmt.Sprintf("{{%s}}::lightBlue|bold", quota.Name),
		)
		table.AddRow(
			cfmt.Sprintf("{{Namespace:}}::white|bold"),
			cfmt.Sprintf("{{%s}}::lightYellow|bold", quota.Namespace),
		)
		table.AddRow("", "", "", "")
		table.AddRow(
			cfmt.Sprintf("{{Resource}}::white|bold"),
			cfmt.Sprintf("{{Used}}::white|bold"),
			cfmt.Sprintf("{{Hard}}::white|bold"),
			cfmt.Sprintf("{{Usage}}::white|bold"),
		)
		table.AddRow(
			strings.Repeat("─", 28),
			strings.Repeat("─", 12),
			strings.Repeat("─", 12),
			strings.Repeat("─", 18),
		)

		names := make([]string, 0, len(quota.Status.Hard))
		for resourceName := range quota.Status.Hard {
			names = append(names, resourceName.String())
		}
		sort.Strings(names)

		for _, name := range names {
			resourceName := v1.ResourceName(name)
			hard := quota.Status.Hard[resourceName]
			used := quota.Status.Used[resourceName]

			color, pct, bar := resourceUsage(used.AsApproximateFloat64(), hard.AsApproximateFloat64())
			table.AddRow(
				name,
				used.String(),
				hard.String(),
				cfmt.Sprintf("{{%s %s%%}}::"+color, bar, pct),
			)
		}

		fmt.Println(table)
	}
}

func resourceUsage(used, hard float64) (color, percentage, bar string) {
	if hard == 0 {
		return "#FFFFFF", "0.0", progressBar(0)
	}
	pct := used / hard * 100
	return chooseColor(pct), fmt.Sprintf("%.1f", pct), progressBar(pct)
}

func progressBar(percentage float64) string {
	if percentage > 100 {
		percentage = 100
	}
	filled := int(percentage / 100 * progressBarWidth)
	return strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
}

func chooseColor(percentage float64) string {
	switch {
	case percentage >= 100:
		return "#FF0000"
	case percentage >= 90:
		return "#FF6347"
	case percentage >= 80:
		return "#FF4500"
	case percentage >= 70:
		return "#FFA500"
	case percentage >= 60:
		return "#FFD700"
	case percentage >= 50:
		return "#FFFF00"
	case percentage >= 40:
		return "#ADFF2F"
	case percentage >= 30:
		return "#9ACD32"
	case percentage >= 20:
		return "#90EE90"
	case percentage >= 10:
		return "#008000"
	default:
		return "#FFFFFF"
	}
}
