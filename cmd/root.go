package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/quay"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var integreatlyGHOrg string
var integreatlyOperatorRepo string
var quayRepo string
var releaseVersion string

const (
	GithubTokenKey                     = "github_token"
	GithubUserKey                      = "github_user"
	DefaultIntegreatlyGithubOrg        = "integr8ly"
	DefaultIntegreatlyOperatorRepo     = "integreatly-operator"
	QuayTokenKey                       = "quay_token"
	DefaultIntegreatlyOperatorQuayRepo = "integreatly/integreatly-operator"
)

type githubRepoInfo struct {
	owner string
	repo  string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "delorean",
	Short: "Delorean CLI",
	Long:  `Delorean CLI`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "RHMI release commands",
	Long:  `Commands for creating a RHMI release`,
}

// ewsCmd represents the release command
var ewsCmd = &cobra.Command{
	Use:   "ews",
	Short: "RHMI EWS Commands",
	Long:  `RHMI Early Warning System Commands`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	//flags for the root command (available for all subcommands)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.delorean.yaml)")

	//flags for the release command (available for all its subcommands)
	releaseCmd.PersistentFlags().StringP("token", "t", "", fmt.Sprintf("Github access token. Can be set via the %s env var.", strings.ToUpper(GithubTokenKey)))
	viper.BindPFlag(GithubTokenKey, releaseCmd.PersistentFlags().Lookup("token"))
	releaseCmd.PersistentFlags().StringP("user", "u", "", fmt.Sprintf("Github user. Can be set via the %s env var.", strings.ToUpper(GithubUserKey)))
	viper.BindPFlag(GithubUserKey, releaseCmd.PersistentFlags().Lookup("user"))
	releaseCmd.PersistentFlags().StringVarP(&releaseVersion, "version", "v", "", "Release version")
	releaseCmd.PersistentFlags().StringVarP(&integreatlyGHOrg, "owner", "o", DefaultIntegreatlyGithubOrg, "Github owner")
	releaseCmd.PersistentFlags().StringVarP(&integreatlyOperatorRepo, "repo", "r", DefaultIntegreatlyOperatorRepo, "Github repository")
	releaseCmd.PersistentFlags().String("quayToken", "", fmt.Sprintf("Access token for quay. Can be set via the %s env var", strings.ToUpper(QuayTokenKey)))
	viper.BindPFlag(QuayTokenKey, releaseCmd.PersistentFlags().Lookup("quayToken"))
	releaseCmd.PersistentFlags().StringVar(&quayRepo, "quayRepo", DefaultIntegreatlyOperatorQuayRepo, "Quay repository")

	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(ewsCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".delorean" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".delorean")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func requireValue(key string) (string, error) {
	token := viper.GetString(key)
	if token == "" {
		return "", fmt.Errorf("token for key %s is not defined. Please see usage.", key)
	}
	return token, nil
}

func newGithubClient(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return client
}

func newQuayClient(token string) *quay.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := quay.NewClient(tc)
	return client
}

func handleError(err error) {
	fmt.Println("Error:", err)
	os.Exit(1)
}
