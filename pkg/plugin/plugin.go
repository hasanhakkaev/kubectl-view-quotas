package plugin

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/util"
)

const progressBarWidth = 10

// Three-tier severity palette with AdaptiveColor so bars stay readable on
// both light and dark terminal backgrounds.
var (
	colorGreen  = lipgloss.AdaptiveColor{Light: "#00875F", Dark: "#00D787"}
	colorYellow = lipgloss.AdaptiveColor{Light: "#AF8700", Dark: "#FFD75F"}
	colorRed    = lipgloss.AdaptiveColor{Light: "#AF0000", Dark: "#FF5F5F"}
	colorOver = lipgloss.AdaptiveColor{Light: "#870087", Dark: "#FF87FF"} // >100%

	labelStyle  = lipgloss.NewStyle().Bold(true)
	nameStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5CB8FF"))
	nsStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700"))
	headerStyle = lipgloss.NewStyle().Padding(0, 1).Bold(true)
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

// eighths provides sub-block precision: each rune fills 1/8 of a cell.
var eighths = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

func RunPlugin(configFlags *genericclioptions.ConfigFlags, cmd *cobra.Command) error {
	// Disable ANSI when piped or when NO_COLOR is set.
	if !term.IsTerminal(int(os.Stdout.Fd())) || os.Getenv("NO_COLOR") != "" {
		lipgloss.SetColorProfile(termenv.Ascii)
	}

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
	// usage is the raw visual string before colorization.
	usage string
	color lipgloss.TerminalColor
}

// PrintResourceQuotas renders all quotas in the list to stdout.
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

		// Skip resources with no hard limit — they carry no quota signal.
		if hardFloat == 0 {
			continue
		}

		color, usage := resourceUsage(used.AsApproximateFloat64(), hardFloat)
		rows = append(rows, quotaRow{
			resource: name,
			used:     used.String(),
			hard:     hard.String(),
			usage:    usage,
			color:    color,
		})
	}

	colors := make([]lipgloss.TerminalColor, len(rows))
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
				return cellStyle.Foreground(colors[row])
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

// resourceUsage returns the AdaptiveColor and a fixed-width usage string:
// "▉▉▉▉▉     50.0%". The percentage is right-aligned in a 6-char field.
func resourceUsage(used, hard float64) (lipgloss.TerminalColor, string) {
	pct := used / hard * 100
	return chooseColor(pct), fmt.Sprintf("%s %5.1f%%", progressBar(pct), pct)
}

// progressBar renders a 10-char bar using eighth-block characters for
// sub-cell precision (80 distinct steps over the full width).
func progressBar(pct float64) string {
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}
	total := pct / 100 * float64(progressBarWidth*8)
	full := int(total) / 8
	rem := int(total) % 8

	var b strings.Builder
	b.WriteString(strings.Repeat("█", full))
	if rem > 0 && full < progressBarWidth {
		b.WriteRune(eighths[rem])
		full++
	}
	b.WriteString(strings.Repeat(" ", progressBarWidth-full))
	return b.String()
}

// chooseColor returns one of three severity tiers using AdaptiveColor so the
// palette adapts to light and dark terminal backgrounds.
func chooseColor(pct float64) lipgloss.TerminalColor {
	switch {
	case pct > 100:
		return colorOver
	case pct >= 90:
		return colorRed
	case pct >= 75:
		return colorYellow
	default:
		return colorGreen
	}
}
