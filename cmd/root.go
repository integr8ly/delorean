package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v30/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

const (
	GITHUB_TOKEN_KEY          = "github_token"
	INTEGREATLY_GITHUB_ORG    = "integr8ly"
	INTEGREATLY_OPERATOR_REPO = "integreatly-operator"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "delorean",
	Short: "Delorean CLI",
	Long:  `Delorean CLI`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
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

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.delorean.yaml)")
	rootCmd.PersistentFlags().StringP("token", "t", "", fmt.Sprintf("Github access token. Can be set via the %s env var.", strings.ToUpper(GITHUB_TOKEN_KEY)))
	viper.BindPFlag(GITHUB_TOKEN_KEY, rootCmd.PersistentFlags().Lookup("token"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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

func requireGithubToken() (string, error) {
	githubToken := viper.GetString(GITHUB_TOKEN_KEY)
	if githubToken == "" {
		return "", errors.New("Github token is not defined. Please check usage instructions.")
	}
	return githubToken, nil
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
