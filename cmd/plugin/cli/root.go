package cli

import (
	"errors"
	"flag"
	"fmt"
	"github.com/hasanhakkaev/kubectl-view-quotas/pkg/plugin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"os"
	"strings"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
	allNamespaces         bool
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "kubectl-view-quotas",
		Short:         "",
		Long:          `.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRun: func(cmd *cobra.Command, args []string) {
			err := viper.BindPFlags(cmd.Flags())
			if err != nil {
				return
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			finishedCh := make(chan bool, 1)
			go func() {
				for range finishedCh {
					fmt.Printf("\r")
					return
				}
			}()

			defer func() {
				finishedCh <- true
			}()

			if err := plugin.RunPlugin(KubernetesConfigFlags, cmd); err != nil {
				return errors.Unwrap(err)
			}

			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	KubernetesConfigFlags = genericclioptions.NewConfigFlags(true)
	KubernetesConfigFlags.AddFlags(cmd.Flags())

	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false,
		"List resource quotas from all namespaces")

	klog.InitFlags(nil)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if f.Name != "v" {
			_ = cmd.Flags().MarkHidden(f.Name)
		}
	})
	_ = cmd.Flags().MarkHidden("as-group")
	_ = cmd.Flags().MarkHidden("as-uid")
	_ = cmd.Flags().MarkHidden("context")
	_ = cmd.Flags().MarkHidden("disable-compression")
	_ = cmd.Flags().MarkHidden("kubeconfig")
	_ = cmd.Flags().MarkHidden("tls-server-name")
	_ = cmd.Flags().MarkHidden("as")
	_ = cmd.Flags().MarkHidden("cache-dir")
	_ = cmd.Flags().MarkHidden("certificate-authority")
	_ = cmd.Flags().MarkHidden("client-certificate")
	_ = cmd.Flags().MarkHidden("client-key")
	_ = cmd.Flags().MarkHidden("cluster")
	_ = cmd.Flags().MarkHidden("insecure-skip-tls-verify")
	_ = cmd.Flags().MarkHidden("password")
	_ = cmd.Flags().MarkHidden("request-timeout")
	_ = cmd.Flags().MarkHidden("server")
	_ = cmd.Flags().MarkHidden("token")
	_ = cmd.Flags().MarkHidden("user")
	_ = cmd.Flags().MarkHidden("username")

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
}
