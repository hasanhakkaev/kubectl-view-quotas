package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/util"
)

const progressBarWidth = 10

var (
	labelStyle  = lipgloss.NewStyle().Bold(true)
	nameStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5CB8FF"))
	nsStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700"))
	headerStyle = lipgloss.NewStyle().Padding(0, 1).Bold(true)
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

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

	PrintResourceQuotas(quotas)
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

type quotaRow struct {
	resource string
	used     string
	hard     string
	// usage is the raw visual string ("████░░░░░░  50.0%" or "N/A") before coloring.
	usage string
	color string
}

// PrintResourceQuotas renders all quotas in the list.
func PrintResourceQuotas(list *v1.ResourceQuotaList) {
	for i, quota := range list.Items {
		if i > 0 {
			fmt.Println()
		}
		printQuota(quota)
	}
}

func printQuota(quota v1.ResourceQuota) {
	names := make([]string, 0, len(quota.Status.Hard))
	for resourceName := range quota.Status.Hard {
		names = append(names, resourceName.String())
	}
	sort.Strings(names)

	rows := make([]quotaRow, 0, len(names))
	for _, name := range names {
		resourceName := v1.ResourceName(name)
		hard := quota.Status.Hard[resourceName]
		used := quota.Status.Used[resourceName]
		hardFloat := hard.AsApproximateFloat64()

		r := quotaRow{
			resource: name,
			used:     used.String(),
			hard:     hard.String(),
		}
		if hardFloat == 0 {
			r.usage = "N/A"
			r.color = "#808080"
		} else {
			r.color, r.usage = resourceUsage(used.AsApproximateFloat64(), hardFloat)
		}
		rows = append(rows, r)
	}

	// Capture per-row colors for the StyleFunc closure.
	colors := make([]string, len(rows))
	for i, r := range rows {
		colors[i] = r.color
	}

	t := table.New().
		BorderStyle(borderStyle).
		Headers("RESOURCE", "USED", "HARD", "USAGE").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if col == 3 {
				return cellStyle.Foreground(lipgloss.Color(colors[row]))
			}
			return cellStyle
		})

	for _, r := range rows {
		t = t.Row(r.resource, r.used, r.hard, r.usage)
	}

	fmt.Printf("%s %s\n",
		labelStyle.Render("Name:"),
		nameStyle.Render(quota.Name))
	fmt.Printf("%s %s\n\n",
		labelStyle.Render("Namespace:"),
		nsStyle.Render(quota.Namespace))
	fmt.Println(t.String())
}

// resourceUsage returns the hex color and a fixed-width usage string with a
// right-aligned percentage: "████░░░░░░  17.8%".
func resourceUsage(used, hard float64) (color, usage string) {
	pct := used / hard * 100
	return chooseColor(pct), fmt.Sprintf("%s %5.1f%%", progressBar(pct), pct)
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
