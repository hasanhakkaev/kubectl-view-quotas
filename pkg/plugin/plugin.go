package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

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

type quotaRow struct {
	resource string
	used     string
	hard     string
	// usage is the pre-formatted visual string: "████░░░░░░  50.0%" or "N/A"
	// kept separate from color so we can pad before applying ANSI codes.
	usage string
	color string
}

func printResourceQuotas(list *v1.ResourceQuotaList) {
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

	const (
		hResource = "RESOURCE"
		hUsed     = "USED"
		hHard     = "HARD"
		hUsage    = "USAGE"
	)

	// Content column widths (text only, no surrounding spaces).
	cw := [4]int{len(hResource), len(hUsed), len(hHard), len(hUsage)}
	for _, r := range rows {
		if l := len(r.resource); l > cw[0] {
			cw[0] = l
		}
		if l := len(r.used); l > cw[1] {
			cw[1] = l
		}
		if l := len(r.hard); l > cw[2] {
			cw[2] = l
		}
		if l := len(r.usage); l > cw[3] {
			cw[3] = l
		}
	}

	// border prints a horizontal rule: left + (─×cw+2) + mid + … + right.
	border := func(left, mid, right string) {
		fmt.Printf("%s%s%s%s%s%s%s%s%s\n",
			left,
			strings.Repeat("─", cw[0]+2),
			mid,
			strings.Repeat("─", cw[1]+2),
			mid,
			strings.Repeat("─", cw[2]+2),
			mid,
			strings.Repeat("─", cw[3]+2),
			right)
	}

	// Name / Namespace header — printed above the box.
	fmt.Printf("%s %s\n",
		cfmt.Sprintf("{{Name:}}::white|bold"),
		cfmt.Sprintf("{{%s}}::lightBlue|bold", quota.Name))
	fmt.Printf("%s %s\n\n",
		cfmt.Sprintf("{{Namespace:}}::white|bold"),
		cfmt.Sprintf("{{%s}}::lightYellow|bold", quota.Namespace))

	border("┌", "┬", "┐")

	// Column headers: pre-pad to content width, then apply bold.
	// Pre-padding before cfmt ensures ANSI codes don't affect column width.
	fmt.Printf("│ %s │ %s │ %s │ %s │\n",
		cfmt.Sprintf("{{%s}}::white|bold", fmt.Sprintf("%-*s", cw[0], hResource)),
		cfmt.Sprintf("{{%s}}::white|bold", fmt.Sprintf("%-*s", cw[1], hUsed)),
		cfmt.Sprintf("{{%s}}::white|bold", fmt.Sprintf("%-*s", cw[2], hHard)),
		cfmt.Sprintf("{{%s}}::white|bold", fmt.Sprintf("%-*s", cw[3], hUsage)))

	border("├", "┼", "┤")

	for _, r := range rows {
		// Pre-pad usage string before colorizing so ANSI codes don't shift columns.
		coloredUsage := cfmt.Sprintf("{{%s}}::"+r.color, fmt.Sprintf("%-*s", cw[3], r.usage))
		fmt.Printf("│ %-*s │ %-*s │ %-*s │ %s │\n",
			cw[0], r.resource,
			cw[1], r.used,
			cw[2], r.hard,
			coloredUsage)
	}

	border("└", "┴", "┘")
}

// resourceUsage returns the cfmt color and a fixed-width usage string
// "████░░░░░░  50.0%" with a right-aligned percentage field.
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
