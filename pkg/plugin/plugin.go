package plugin

import (
	"context"
	"fmt"
	"github.com/gosuri/uitable"
	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/util"
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
		return errors.WithMessage(err, "Failed getting namespace")
	}

	if getFlagBool(cmd, "all-namespaces") {
		namespace = ""
	}

	quotas, err := getQuotas(clientSet, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to list resource quotas")
	}

	printResourceQuotas(quotas)
	return nil
}

// Gets the  flag value as a boolean, otherwise returns false if the flag value is nil
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
	table := uitable.New()
	table.Wrap = true

	// Register styles
	cfmt.RegisterStyle("url", func(s string) string {
		return cfmt.Sprintf("{{%s}}::yellow|underline", s)
	})

	for _, quota := range list.Items {
		table.AddRow("Name:", cfmt.Sprintf("{{%s}}::lightBlue|bold", quota.Name))
		table.AddRow("Namespace:", cfmt.Sprintf("{{%s}}::lightYellow|bold", quota.Namespace))
		table.AddRow("Resource", cfmt.Sprintf("{{Used}}::green"), cfmt.Sprintf("{{Hard}}::red"))
		table.AddRow("--------", "----", "----")

		for resourceName, hard := range quota.Status.Hard {
			used := quota.Status.Used[resourceName]
			color, percentage := chooseColour(used.Value(), hard.Value())
			table.AddRow(resourceName.String(), cfmt.Sprintf("{{%s (%s%%)}}::"+color, used.String(), percentage), hard.String())
		}
		table.AddRow("")
	}

	fmt.Println(table)
}

func chooseColour(used, hard int64) (string, string) {
	if hard == 0 {
		return "#FFFFFF", "0.00" // White for divide by zero scenario
	}

	percentage := float64(used) / float64(hard) * 100
	percentageStr := fmt.Sprintf("%.2f", percentage)

	var color string
	switch {
	case percentage >= 100:
		color = "#FF0000" // red
	case percentage >= 90:
		color = "#FF6347" // lightRed
	case percentage >= 80:
		color = "#FF4500" // orangeRed
	case percentage >= 70:
		color = "#FFA500" // orange
	case percentage >= 60:
		color = "#FFD700" // gold
	case percentage >= 50:
		color = "#FFFF00" // yellow
	case percentage >= 40:
		color = "#ADFF2F" // yellowGreen
	case percentage >= 30:
		color = "#9ACD32" // greenYellow
	case percentage >= 20:
		color = "#90EE90" // lightGreen
	case percentage >= 10:
		color = "#008000" // green
	default:
		color = "#FFFFFF" // white
	}
	return color, percentageStr
}
